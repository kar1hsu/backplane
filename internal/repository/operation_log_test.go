package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOperationLogPurge(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:oplog_purge_test?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysOperationLog{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	previous := app.DB
	app.DB = db
	t.Cleanup(func() { app.DB = previous })

	now := time.Now()
	for i := 0; i < 7; i++ {
		entry := model.SysOperationLog{Path: "/old"}
		entry.CreatedAt = now.AddDate(0, 0, -40)
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("create old log: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := db.Create(&model.SysOperationLog{Path: "/new"}).Error; err != nil {
			t.Fatalf("create new log: %v", err)
		}
	}
	repo := NewOperationLogRepo()
	ctx := context.Background()

	removed, err := repo.DeleteBefore(ctx, now.AddDate(0, 0, -30))
	if err != nil || removed != 7 {
		t.Fatalf("DeleteBefore() = %d, %v; want 7", removed, err)
	}
	if n := countLogs(t, db); n != 3 {
		t.Fatalf("logs after retention purge = %d, want 3", n)
	}

	if err := repo.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if n := countLogs(t, db); n != 0 {
		t.Fatalf("logs after Clear = %d, want 0", n)
	}
}

func countLogs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Unscoped().Model(&model.SysOperationLog{}).Count(&n).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	return n
}
