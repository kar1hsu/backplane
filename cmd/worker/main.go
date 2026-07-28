package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/tasks"
)

func main() {
	cfgFile := flag.String("config", "", "config file path")
	flag.Parse()

	if err := app.Init(*cfgFile); err != nil {
		panic(fmt.Sprintf("init app failed: %v", err))
	}

	mgr := app.TaskMgr

	// Register task handlers
	tasks.RegisterHandlers(mgr.Worker)
	app.Log.Info("task handlers registered")

	// Start worker (consumer). Safe to run multiple instances — Redis
	// load-balances tasks across all running workers.
	// NOTE: periodic tasks are scheduled by a separate process (cmd/scheduler).
	go func() {
		if err := mgr.Worker.Start(); err != nil {
			app.Log.Errorf("worker start failed: %v", err)
		}
	}()
	app.Log.Info("worker started, waiting for tasks...")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Log.Info("shutting down worker...")
	mgr.Worker.Stop()
	app.Close()
	app.Log.Info("worker stopped")
}
