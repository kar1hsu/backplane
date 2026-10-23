package app

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

// Enforcer is shared by every request (Enforce) and the role admin APIs
// (policy writes), so it must be the synchronized variant: the plain Enforcer
// mutates unguarded maps and concurrent use can crash the process.
var Enforcer *casbin.SyncedEnforcer

func InitCasbin() error {
	adapter, err := gormadapter.NewAdapterByDB(DB)
	if err != nil {
		return fmt.Errorf("create casbin adapter failed: %w", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(Cfg.Casbin.ModelPath, adapter)
	if err != nil {
		return fmt.Errorf("create casbin enforcer failed: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("load casbin policy failed: %w", err)
	}

	Enforcer = enforcer
	return nil
}
