package app

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// zapGormLogger adapts GORM's logger to our zap logger, so SQL logs are
// structured, written to the same files/stdout as the rest of the app, and
// level-controlled — instead of going to the standard library log with ANSI
// color codes (which become garbage when written to a file).
type zapGormLogger struct {
	level         gormlogger.LogLevel
	slowThreshold time.Duration
	// parameterized logs SQL with ? placeholders instead of the bound values,
	// keeping password hashes, config values and audit payloads out of logs.
	parameterized bool
}

func newGormLogger(level gormlogger.LogLevel, parameterized bool) gormlogger.Interface {
	return &zapGormLogger{level: level, slowThreshold: 200 * time.Millisecond, parameterized: parameterized}
}

// ParamsFilter implements gorm's logger.ParamsFilter: GORM calls it before
// rendering the SQL handed to Trace.
func (l *zapGormLogger) ParamsFilter(_ context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.parameterized {
		return sql, nil
	}
	return sql, params
}

func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	nl := *l
	nl.level = level
	return &nl
}

func (l *zapGormLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		Log.Infof(msg, args...)
	}
}

func (l *zapGormLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		Log.Warnf(msg, args...)
	}
}

func (l *zapGormLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		Log.Errorf(msg, args...)
	}
}

func (l *zapGormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil && l.level >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		Log.Errorw("gorm sql error", "elapsed", elapsed.String(), "rows", rows, "sql", sql, "err", err)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn:
		Log.Warnw("gorm slow sql", "elapsed", elapsed.String(), "rows", rows, "sql", sql)
	case l.level >= gormlogger.Info:
		Log.Infow("gorm sql", "elapsed", elapsed.String(), "rows", rows, "sql", sql)
	}
}
