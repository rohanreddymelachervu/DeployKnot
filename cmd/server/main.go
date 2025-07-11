package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deployknot/internal/api"
	"deployknot/internal/config"
	"deployknot/internal/database"
	"deployknot/internal/handlers"
	"deployknot/internal/services"
	"deployknot/pkg/logger"

	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	log := logger.New(cfg.Logging.Level)
	log.Info("Starting DeployKnot server...")

	// Initialize database
	db, err := database.New(cfg.GetDatabaseURL(), log.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := db.RunMigrations("migrations"); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize Redis
	redis, err := database.NewRedis(cfg.GetRedisURL(), log.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redis.Close()

	// Initialize repository
	repo := database.NewRepository(db.DB, log.Logger)

	// Initialize queue service
	queueService := services.NewQueueService(redis.Client, log.Logger)

	// Initialize services
	deploymentService := services.NewDeploymentService(repo, queueService, log.Logger)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db, redis, log.Logger)
	deploymentHandler := handlers.NewDeploymentHandler(deploymentService, log.Logger)

	// Initialize router
	router := api.NewRouter(log.Logger, healthHandler, deploymentHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router.GetEngine(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}
