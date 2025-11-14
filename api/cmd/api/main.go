package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bmachimbira/loyalty/api/internal/config"
	httputil "github.com/bmachimbira/loyalty/api/internal/http"
	"github.com/bmachimbira/loyalty/api/internal/logging"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize structured logger
	logger := logging.New()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"log_level", os.Getenv("LOG_LEVEL"),
	)

	// Initialize database connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("Unable to create database connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		logger.Error("Unable to ping database", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully connected to database")

	// Set up router with all routes and middleware
	router := httputil.SetupRouter(pool, cfg.JWTSecret, cfg.HMACKeys)

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handler
	go func() {
		logger.Info("Starting HTTP server",
			"port", cfg.Port,
			"read_timeout", srv.ReadTimeout,
			"write_timeout", srv.WriteTimeout,
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received shutdown signal",
		"signal", sig.String(),
	)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	logger.Info("Shutting down server gracefully...")

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown complete")
	slog.SetDefault(nil) // Flush any remaining logs
}
