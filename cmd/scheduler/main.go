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

	// Register cron jobs.
	tasks.RegisterCronJobs(mgr.Scheduler)
	app.Log.Info("cron jobs registered")

	// Start the scheduler (producer of periodic tasks).
	//
	// Deploy exactly ONE scheduler instance: asynq.Scheduler has no leader
	// election, so N instances would enqueue each cron task N times. Cron tasks
	// are also registered with a Unique TTL as a safety net against accidental
	// duplicate instances.
	go func() {
		if err := mgr.Scheduler.Start(); err != nil {
			app.Log.Errorf("scheduler start failed: %v", err)
		}
	}()
	app.Log.Info("scheduler started")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Log.Info("shutting down scheduler...")
	mgr.Scheduler.Stop()
	app.Close()
	app.Log.Info("scheduler stopped")
}
