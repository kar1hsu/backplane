package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kar1hsu/backplane"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/module/admin"
	"github.com/kar1hsu/backplane/internal/module/api"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/server"
)

func main() {
	cfgFile := flag.String("config", "", "config file path")
	flag.Parse()

	if err := app.Init(*cfgFile); err != nil {
		panic(fmt.Sprintf("init app failed: %v", err))
	}

	if err := app.AutoMigrate(); err != nil {
		panic(fmt.Sprintf("auto migrate failed: %v", err))
	}
	app.Log.Info("database migrated")

	if err := app.SeedData(); err != nil {
		app.Log.Warnf("seed data: %v", err)
	}

	// Seed default config values into DB and warm the Redis cache.
	if err := setting.Init(context.Background()); err != nil {
		app.Log.Warnf("init settings: %v", err)
	}

	router := server.NewRouter(
		backplane.AdminDist,
		admin.New(),
		api.New(),
	)

	addr := fmt.Sprintf(":%d", app.Cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		app.Log.Infof("server starting at %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Log.Fatalf("server run failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		app.Log.Errorf("server forced to shutdown: %v", err)
	}
	app.Close()
	app.Log.Info("server stopped")
}
