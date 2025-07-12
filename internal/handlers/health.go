package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db     DatabaseHealthChecker
	redis  RedisHealthChecker
	logger *logrus.Logger
}

// DatabaseHealthChecker interface for database health checks
type DatabaseHealthChecker interface {
	HealthCheck() error
}

// RedisHealthChecker interface for Redis health checks
type RedisHealthChecker interface {
	HealthCheck() error
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db DatabaseHealthChecker, redis RedisHealthChecker, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// HealthCheck handles the health check endpoint
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
	}

	// Check database health
	if err := h.db.HealthCheck(); err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "unhealthy"
		h.logger.WithError(err).Error("Database health check failed")
	} else {
		response.Services["database"] = "healthy"
	}

	// Check Redis health
	if err := h.redis.HealthCheck(); err != nil {
		response.Status = "unhealthy"
		response.Services["redis"] = "unhealthy"
		h.logger.WithError(err).Error("Redis health check failed")
	} else {
		response.Services["redis"] = "healthy"
	}

	// Set appropriate HTTP status code
	if response.Status == "healthy" {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// HealthCheck is a simple health check function for the router
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"message":   "DeployKnot API is running",
	})
}
