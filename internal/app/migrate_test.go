package app

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSeedDataGeneratesAdminPassword(t *testing.T) {
	output := setupSeedTest(t, "")

	if err := SeedData(); err != nil {
		t.Fatalf("SeedData() error = %v", err)
	}

	match := regexp.MustCompile(`初始密码：(\S+)`).FindStringSubmatch(output.String())
	if match == nil {
		t.Fatalf("no password banner in output %q", output.String())
	}
	hash := adminPasswordHash(t)
	if !utils.CheckPassword(match[1], hash) {
		t.Fatal("printed password does not match the stored hash")
	}
	if utils.CheckPassword("admin123", hash) {
		t.Fatal("seed still uses the old fixed password admin123")
	}
}

func TestSeedDataUsesConfiguredAdminPassword(t *testing.T) {
	output := setupSeedTest(t, "configured-secret")

	if err := SeedData(); err != nil {
		t.Fatalf("SeedData() error = %v", err)
	}
	if !utils.CheckPassword("configured-secret", adminPasswordHash(t)) {
		t.Fatal("admin password is not seed.admin_password")
	}
	if strings.Contains(output.String(), "configured-secret") {
		t.Fatal("a configured password must not be printed")
	}
}

func setupSeedTest(t *testing.T, adminPassword string) *bytes.Buffer {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	previousDB, previousLog, previousCfg, previousOutput := DB, Log, Cfg, seedPasswordOutput
	t.Cleanup(func() {
		DB, Log, Cfg, seedPasswordOutput = previousDB, previousLog, previousCfg, previousOutput
	})
	DB = db
	Log = zap.NewNop().Sugar()
	Cfg.Seed.AdminPassword = adminPassword
	output := &bytes.Buffer{}
	seedPasswordOutput = output

	if err := AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return output
}

func adminPasswordHash(t *testing.T) string {
	t.Helper()
	var admin model.SysUser
	if err := DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatalf("load admin: %v", err)
	}
	return admin.Password
}
