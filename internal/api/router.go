package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"deployknot/internal/handlers"
)

// Router represents the API router
type Router struct {
	engine     *gin.Engine
	logger     *logrus.Logger
	health     *handlers.HealthHandler
	deployment *handlers.DeploymentHandler
}

// NewRouter creates a new API router
func NewRouter(logger *logrus.Logger, health *handlers.HealthHandler, deployment *handlers.DeploymentHandler) *Router {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	router := &Router{
		engine:     gin.New(),
		logger:     logger,
		health:     health,
		deployment: deployment,
	}

	router.setupMiddleware()
	router.setupRoutes()

	return router
}

// setupMiddleware configures middleware for the router
func (r *Router) setupMiddleware() {
	// Recovery middleware
	r.engine.Use(gin.Recovery())

	// CORS middleware
	r.engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // Set to false for AllowOrigins: ["*"]
		MaxAge:           12 * time.Hour,
	}))

	// Logging middleware
	r.engine.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		r.logger.WithFields(logrus.Fields{
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
}

// setupRoutes configures the API routes
func (r *Router) setupRoutes() {
	// Health check endpoint (no auth required)
	r.engine.GET("/health", r.health.HealthCheck)

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Health check endpoint in API group
		v1.GET("/health", r.health.HealthCheck)

		// Deployment routes
		deployments := v1.Group("/deployments")
		{
			deployments.GET("", r.deployment.ListDeployments)
			deployments.POST("", r.deployment.CreateDeployment)
			deployments.GET("/:id", r.deployment.GetDeployment)
			deployments.GET("/:id/logs", r.deployment.GetDeploymentLogs)
			deployments.GET("/:id/steps", r.deployment.GetDeploymentSteps)
		}
	}
}

// notImplemented is a placeholder for endpoints not yet implemented
func (r *Router) notImplemented(c *gin.Context) {
	c.JSON(501, gin.H{
		"error":   "Not Implemented",
		"message": "This endpoint is not yet implemented",
	})
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
