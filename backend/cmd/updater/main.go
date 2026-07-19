package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"container-updater/backend/internal/api"
	"container-updater/backend/internal/config"
	"container-updater/backend/internal/db"
	"container-updater/backend/internal/logger"
	"container-updater/backend/internal/monitor"
)

func main() {
	// Parse CLI options if any
	dbPathFlag := flag.String("db", "", "Path to SQLite database file")
	portFlag := flag.String("port", "", "HTTP port to listen on")
	flag.Parse()

	// 1. Load Configurations
	config.Load()

	// Override config with flags if provided
	if *dbPathFlag != "" {
		config.GlobalConfig.DatabasePath = *dbPathFlag
	}
	if *portFlag != "" {
		config.GlobalConfig.Port = *portFlag
	}

	// 2. Initialize Logger
	logger.Init(config.GlobalConfig.LogLevel)
	logger.Log.Info("Starting Container Updater Backend...")

	// 3. Initialize SQLite Database
	if err := db.Init(config.GlobalConfig.DatabasePath); err != nil {
		logger.Log.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.DB.Close()

	// 4. Initialize WebSocket Hub
	api.InitHub()

	// 5. Setup Router
	router := api.NewRouter()

	// 5. Initialize and Start Cron Scheduler
	scheduler, err := monitor.NewMonitorScheduler()
	if err != nil {
		logger.Log.Error("failed to create monitor scheduler", "error", err)
		os.Exit(1)
	}
	
	backgroundCtx := context.Background()
	if err := scheduler.Start(backgroundCtx); err != nil {
		logger.Log.Error("failed to start monitor scheduler", "error", err)
		os.Exit(1)
	}

	// 6. Start HTTP Server
	serverAddr := ":" + config.GlobalConfig.Port
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	go func() {
		logger.Log.Info("HTTP server running", "addr", serverAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down backend server...")

	// Stop scheduler
	scheduler.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", "error", err)
	}

	logger.Log.Info("Backend stopped gracefully.")
}
