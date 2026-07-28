package app

import (
	"fmt"
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
	Task     TaskConfig     `mapstructure:"task"`
}

type TaskConfig struct {
	Concurrency int      `mapstructure:"concurrency"`
	Queues      []string `mapstructure:"queues"`
}

type ServerConfig struct {
	Port         int      `mapstructure:"port"`
	Mode         string   `mapstructure:"mode"`
	AllowOrigins []string `mapstructure:"allow_origins"`
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
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("jwt.expire", 7200)
	viper.SetDefault("task.concurrency", 10)
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

// validateConfig fails fast on missing required fields and rejects weak or
// default JWT secrets when running in release mode.
func validateConfig() error {
	if Cfg.Database.Driver == "" {
		return fmt.Errorf("database.driver is required")
	}
	if Cfg.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}
	if Cfg.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	if Cfg.Server.Mode == "release" {
		if Cfg.JWT.Secret == defaultJWTSecret {
			return fmt.Errorf("jwt.secret must be changed from the default value in release mode")
		}
		if len(Cfg.JWT.Secret) < 32 {
			return fmt.Errorf("jwt.secret must be at least 32 bytes in release mode")
		}
	}
	return nil
}
