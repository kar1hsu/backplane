package app

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase() error {
	cfg := Cfg.Database

	gormLogLevel := logger.Info
	switch cfg.LogLevel {
	case "silent":
		gormLogLevel = logger.Silent
	case "error":
		gormLogLevel = logger.Error
	case "warn":
		gormLogLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		// Bound values are only logged in debug mode (local development).
		Logger: newGormLogger(gormLogLevel, Cfg.Server.Mode != "debug"),
		// 把驱动层错误翻译成 gorm 哨兵错误，便于 errors.Is(err, gorm.ErrDuplicatedKey) 等判断
		TranslateError: true,
	}

	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.DSN(Cfg.Timezone))
	case "mysql":
		dialector = mysql.Open(cfg.DSN(Cfg.Timezone))
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("connect database failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	return nil
}
