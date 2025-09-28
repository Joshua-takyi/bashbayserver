package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/joshua-takyi/ww/internal/config"
	"github.com/joshua-takyi/ww/internal/connect"
	"github.com/joshua-takyi/ww/internal/container"
	"github.com/joshua-takyi/ww/internal/routes"
)

func main() {
	// Load environment variables
	_ = godotenv.Load(".env.local")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	cld, err := connect.CloudinaryCredentials()
	if err != nil {
		slog.Error("Failed to connect to Cloudinary", "error", err)
		os.Exit(1)
	}
	connect.Cld = cld

	// Setup logger
	logger := setupLogger(cfg)
	logger.Info("Starting Bashbay API server", "environment", cfg.Environment)

	// Initialize database connections
	supaClient, supaUrl, supaKey, err := connect.InitSupabase()
	if err != nil {
		logger.Error("Failed to connect to Supabase", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to Supabase successfully")

	mongoClient, err := connect.MongoDBConnect()
	if err != nil {
		logger.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to MongoDB successfully")

	// Initialize dependency container
	appContainer := container.NewContainer(logger, cld, supaClient, mongoClient, supaUrl, supaKey)

	// Setup routes
	router := routes.SetupRoutes(appContainer)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server is shutting down...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	// Close database connections
	connect.Disconnect()
	if err := connect.MongoDBDisconnect(); err != nil {
		logger.Error("Error disconnecting from MongoDB", "error", err)
	}

	logger.Info("Server exited")
}

func setupLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	if cfg.IsProduction() {
		// JSON logging for production
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		// Human-readable logging for development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)
}
