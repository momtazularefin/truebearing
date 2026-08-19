// Package main is the entry point for the TrueBearing server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/momtazularefin/truebearing/internal/api"
	"github.com/momtazularefin/truebearing/internal/config"
	"github.com/momtazularefin/truebearing/internal/database"
	"github.com/momtazularefin/truebearing/internal/metrics"
	"github.com/momtazularefin/truebearing/internal/queue"
)

func main() {
	// Load configuration.
	cfg := config.Load()

	// Initialise structured logger.
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Connect to PostgreSQL.
	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to database")

	// Run migrations.
	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations complete")

	// Connect to Redis.
	rdb, err := queue.Connect(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("connected to redis")

	// Build router and primary HTTP server.
	handler := api.NewRouter(db, rdb, logger)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start main API server in a goroutine.
	go func() {
		logger.Info("main server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server listen error", "error", err)
			os.Exit(1)
		}
	}()

	// Start dedicated metrics server on secondary listener (pod-local, NFR4).
	metricsSrv := metrics.NewServer(":"+cfg.MetricsPort, logger)
	go func() {
		logger.Info("metrics server starting", "addr", metricsSrv.Addr)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server listen error", "error", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics server forced to shutdown", "error", err)
	}

	logger.Info("server stopped gracefully")
}
