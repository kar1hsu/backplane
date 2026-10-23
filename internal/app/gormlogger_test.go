package app

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGormLoggerParameterizedHidesBoundValues(t *testing.T) {
	const secret = "bcrypt-hash-value"

	tests := []struct {
		name          string
		parameterized bool
		wantSecret    bool
	}{
		{name: "parameterized", parameterized: true, wantSecret: false},
		{name: "debug mode keeps values", parameterized: false, wantSecret: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			previous := Log
			Log = zap.New(core).Sugar()
			t.Cleanup(func() { Log = previous })

			db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
				Logger: newGormLogger(gormlogger.Info, tt.parameterized),
			})
			if err != nil {
				t.Fatalf("open SQLite: %v", err)
			}
			if err := db.Exec("CREATE TABLE secrets (value TEXT)").Error; err != nil {
				t.Fatalf("create table: %v", err)
			}
			if err := db.Exec("INSERT INTO secrets (value) VALUES (?)", secret).Error; err != nil {
				t.Fatalf("insert: %v", err)
			}

			var logged strings.Builder
			for _, entry := range logs.All() {
				logged.WriteString(entry.ContextMap()["sql"].(string))
			}
			if got := strings.Contains(logged.String(), secret); got != tt.wantSecret {
				t.Fatalf("logged SQL %q contains value = %v, want %v", logged.String(), got, tt.wantSecret)
			}
		})
	}
}
