package app

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Timezone string         `mapstructure:"timezone"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Casbin   CasbinConfig   `mapstructure:"casbin"`
	Log      LogConfig      `mapstructure:"log"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Task     TaskConfig     `mapstructure:"task"`
	Seed     SeedConfig     `mapstructure:"seed"`
}

// SeedConfig drives the one-time fresh-install seed.
type SeedConfig struct {
	// AdminPassword is the initial password of the built-in "admin" account.
	// Empty generates a random one and prints it once to stdout.
	AdminPassword string `mapstructure:"admin_password"`
}

// minSeedAdminPasswordLen is the minimum length of a configured seed password.
const minSeedAdminPasswordLen = 8

type TaskConfig struct {
	Concurrency int      `mapstructure:"concurrency"`
	Queues      []string `mapstructure:"queues"`
}

type ServerConfig struct {
	Port         int      `mapstructure:"port"`
	Mode         string   `mapstructure:"mode"`
	AllowOrigins []string `mapstructure:"allow_origins"`
	// TrustedProxies lists the reverse proxies (IP or CIDR) whose
	// X-Forwarded-For / X-Real-IP headers are honored. Empty trusts none.
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
}

// DSN builds the database connection string. timezone (an IANA name like
// Asia/Shanghai) is applied to Postgres directly; MySQL uses loc=Local, which
// follows the process-wide time.Local set by InitTimezone.
func (d *DatabaseConfig) DSN(timezone string) string {
	switch d.Driver {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=%s",
			d.Host, d.Port, d.Username, d.Password, d.DBName, timezone,
		)
	default:
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			d.Username, d.Password, d.Host, d.Port, d.DBName, d.Charset,
		)
	}
}

type RedisConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int64  `mapstructure:"expire"`
	Issuer string `mapstructure:"issuer"`
}

type CasbinConfig struct {
	ModelPath string `mapstructure:"model_path"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Directory  string `mapstructure:"directory"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type StorageConfig struct {
	Directory    string            `mapstructure:"directory"`
	PublicURL    string            `mapstructure:"public_url"`
	MaxSize      int64             `mapstructure:"max_size"`
	AllowedTypes map[string]string `mapstructure:"allowed_types"`
}

var Cfg Config

// defaultJWTSecret is the placeholder secret shipped in config.yaml.example.
// It must never be used in production.
const defaultJWTSecret = "backplane-jwt-secret-key-change-in-production"

func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(".")
	}

	setConfigDefaults()

	// Allow nested config keys to be overridden via env vars, e.g. the env var
	// DATABASE_HOST overrides database.host (used by deploy/docker-compose.yaml).
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config failed: %w", err)
	}
	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("unmarshal config failed: %w", err)
	}
	if err := validateConfig(); err != nil {
		return err
	}
	return nil
}

func setConfigDefaults() {
	viper.SetDefault("timezone", "Asia/Shanghai")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	// Registered so SERVER_TRUSTED_PROXIES (comma-separated) can override it.
	viper.SetDefault("server.trusted_proxies", []string{})
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("jwt.expire", 7200)
	viper.SetDefault("storage.directory", "storage/uploads")
	viper.SetDefault("storage.public_url", "/uploads")
	viper.SetDefault("storage.max_size", 10)
	viper.SetDefault("storage.allowed_types", map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
	})
	viper.SetDefault("task.concurrency", 10)
	// Registered so SEED_ADMIN_PASSWORD can be supplied by env alone.
	viper.SetDefault("seed.admin_password", "")
}

// InitTimezone sets the process-wide time.Local from config so that every
// time.Now()/time.Local usage — logs, JWT, GORM timestamps, MySQL loc=Local,
// the asynq scheduler — shares one timezone. Call right after InitConfig.
func InitTimezone() error {
	loc, err := time.LoadLocation(Cfg.Timezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", Cfg.Timezone, err)
	}
	time.Local = loc
	return nil
}

// minJWTSecretLen is the minimum HS256 secret length accepted at startup.
const minJWTSecretLen = 32

// validateConfig fails fast on missing required fields, weak or default JWT
// secrets, and malformed trusted proxies. The secret checks apply in every
// mode: the example config runs in debug mode, so gating them on release
// would let a public placeholder secret sign tokens in real deployments.
func validateConfig() error {
	if Cfg.Database.Driver == "" {
		return fmt.Errorf("database.driver is required")
	}
	if Cfg.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}
	if Cfg.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required; generate one with `openssl rand -hex 32` (or set JWT_SECRET)")
	}
	if Cfg.JWT.Secret == defaultJWTSecret {
		return fmt.Errorf("jwt.secret must not be the default placeholder; generate one with `openssl rand -hex 32`")
	}
	if len(Cfg.JWT.Secret) < minJWTSecretLen {
		return fmt.Errorf("jwt.secret must be at least %d bytes", minJWTSecretLen)
	}
	if p := Cfg.Seed.AdminPassword; p != "" && len(p) < minSeedAdminPasswordLen {
		return fmt.Errorf("seed.admin_password must be at least %d characters", minSeedAdminPasswordLen)
	}
	for _, proxy := range Cfg.Server.TrustedProxies {
		if net.ParseIP(proxy) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("server.trusted_proxies: %q is not a valid IP or CIDR", proxy)
		}
	}
	return nil
}
