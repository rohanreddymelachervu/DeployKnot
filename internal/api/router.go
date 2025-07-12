package api

import (
	"deployknot/internal/database"
	"deployknot/internal/handlers"
	"deployknot/internal/middleware"
	"deployknot/internal/services"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SetupRouter configures the API routes
func SetupRouter(db *database.Database, queue *services.QueueService, logger *logrus.Logger, jwtSecret string) *gin.Engine {
	router := gin.New()

	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	// Recovery middleware
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // Set to false for AllowOrigins: ["*"]
		MaxAge:           12 * time.Hour,
	}))

	// Logging middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"timestamp": param.TimeStamp.Format(time.RFC3339),
			"status":    param.StatusCode,
			"latency":   param.Latency,
			"client_ip": param.ClientIP,
			"method":    param.Method,
			"path":      param.Path,
			"error":     param.ErrorMessage,
		}).Info("HTTP Request")
		return ""
	}))

	// Health check endpoint (no auth required)
	router.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			authHandler := handlers.NewAuthHandler(
				services.NewUserService(db.Repository, logger),
				middleware.NewAuthMiddleware(jwtSecret, logger),
				logger,
			)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected routes (auth required)
		protected := v1.Group("")
		protected.Use(middleware.NewAuthMiddleware(jwtSecret, logger).AuthRequired())
		{
			// Auth profile
			authHandler := handlers.NewAuthHandler(
				services.NewUserService(db.Repository, logger),
				middleware.NewAuthMiddleware(jwtSecret, logger),
				logger,
			)
			protected.GET("/auth/profile", authHandler.GetProfile)

			// Deployment routes
			deploymentHandler := handlers.NewDeploymentHandler(
				services.NewDeploymentService(db.Repository, queue, logger),
				logger,
			)
			protected.POST("/deployments", deploymentHandler.CreateDeployment)
			protected.GET("/deployments", deploymentHandler.GetDeployments)
			protected.GET("/deployments/:id", deploymentHandler.GetDeployment)
			protected.GET("/deployments/:id/logs", deploymentHandler.GetDeploymentLogs)
			protected.GET("/deployments/:id/steps", deploymentHandler.GetDeploymentSteps)
		}
	}

	return router
}
