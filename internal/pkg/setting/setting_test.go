package setting

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSettingsFallbackSeedAndCacheFlow(t *testing.T) {
	setupSettingTestEnvironment(t)
	ctx := context.Background()

	if got := Get("site.name"); got != "Backplane Admin" {
		t.Fatalf("fallback site.name = %q", got)
	}
	if err := Init(ctx); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	var count int64
	if err := app.DB.Model(&model.SysConfig{}).Count(&count).Error; err != nil {
		t.Fatalf("count seeded settings: %v", err)
	}
	if count != int64(len(registry)) {
		t.Fatalf("seeded settings = %d, want %d", count, len(registry))
	}

	if err := Set(ctx, "site.name", "Control Plane"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := GetString("site.name"); got != "Control Plane" {
		t.Fatalf("GetString() = %q", got)
	}

	if err := cache.SetConfigCache("site.name", "Cached Name"); err != nil {
		t.Fatalf("SetConfigCache() error = %v", err)
	}
	if got := Get("site.name"); got != "Cached Name" {
		t.Fatalf("cache precedence value = %q", got)
	}
	if err := cache.DelConfigCache("site.name"); err != nil {
		t.Fatalf("DelConfigCache() error = %v", err)
	}
	if got := Get("site.name"); got != "Control Plane" {
		t.Fatalf("DB fallback value = %q", got)
	}

	if err := Set(ctx, "user.allow_register", "1"); err != nil {
		t.Fatalf("Set(bool) error = %v", err)
	}
	if !GetBool("user.allow_register") {
		t.Fatal("GetBool() did not parse 1 as true")
	}
	if err := Set(ctx, "log.operation_retain_days", "0"); err != nil {
		t.Fatalf("Set(int64) error = %v", err)
	}
	if got := GetInt64("log.operation_retain_days"); got != 0 {
		t.Fatalf("GetInt64() = %d", got)
	}

	if err := app.DB.Model(&model.SysConfig{}).Where("`key` = ?", "site.name").Update("value", "Database Name").Error; err != nil {
		t.Fatalf("direct DB update: %v", err)
	}
	if err := RefreshKey(ctx, "site.name"); err != nil {
		t.Fatalf("RefreshKey() error = %v", err)
	}
	if got := Get("site.name"); got != "Database Name" {
		t.Fatalf("refreshed value = %q", got)
	}
}

func setupSettingTestEnvironment(t *testing.T) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysConfig{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	previousDB := app.DB
	app.DB = db
	t.Cleanup(func() { app.DB = previousDB })

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	previousStore := cache.GetStore()
	cache.InitStore(cache.NewRedisStore(client, "setting-test:"))
	t.Cleanup(func() {
		cache.InitStore(previousStore)
		_ = client.Close()
	})
}
