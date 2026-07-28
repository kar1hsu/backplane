package app

import "fmt"

func Init(cfgFile string) error {
	if err := InitConfig(cfgFile); err != nil {
		return fmt.Errorf("init config: %w", err)
	}

	if err := InitTimezone(); err != nil {
		return fmt.Errorf("init timezone: %w", err)
	}

	InitLogger()
	Log.Info("config loaded successfully")
	Log.Infof("timezone set to %s", Cfg.Timezone)

	if Cfg.JWT.Secret == defaultJWTSecret {
		Log.Warn("jwt.secret 仍为默认值，部署生产前请务必修改")
	}

	if err := InitStorage(); err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	Log.Infof("storage initialized at %s", Cfg.Storage.Directory)

	if err := InitDatabase(); err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	Log.Info("database connected")

	if err := InitRedis(); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	Log.Info("redis connected")

	if err := InitCasbin(); err != nil {
		return fmt.Errorf("init casbin: %w", err)
	}
	Log.Info("casbin initialized")

	InitTask()

	return nil
}

// Close releases all global resources in reverse order of initialization.
// Safe to call on partially-initialized state (each handle is nil-checked).
func Close() {
	if TaskMgr != nil {
		TaskMgr.Close()
	}
	if Redis != nil {
		_ = Redis.Close()
	}
	if DB != nil {
		if sqlDB, err := DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if Log != nil {
		_ = Log.Sync()
	}
}
