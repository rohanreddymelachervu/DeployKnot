package handlers

import (
	"net/http"
	"strconv"
	"time"

	"deployknot/internal/models"
	"deployknot/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DeploymentHandler handles deployment-related HTTP requests
type DeploymentHandler struct {
	deploymentService *services.DeploymentService
	logger            *logrus.Logger
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(deploymentService *services.DeploymentService, logger *logrus.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentService: deploymentService,
		logger:            logger,
	}
}

// CreateDeployment handles POST /api/v1/deployments
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	var req models.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind deployment request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.deploymentService.ValidateDeploymentRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Create deployment
	ctx := c.Request.Context()
	deployment, err := h.deploymentService.CreateDeployment(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create deployment",
			"message": err.Error(),
		})
		return
	}

	// Add initial log entry
	h.deploymentService.AddDeploymentLog(ctx, deployment.ID, "info", "Deployment created successfully", "create_deployment", nil)

	c.JSON(http.StatusCreated, deployment)
}

// GetDeployment handles GET /api/v1/deployments/:id
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid deployment ID",
			"message": "Deployment ID must be a valid UUID",
		})
		return
	}

	ctx := c.Request.Context()
	deployment, err := h.deploymentService.GetDeployment(ctx, id)
	if err != nil {
		if err.Error() == "deployment not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Deployment not found",
				"message": "The specified deployment does not exist",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to get deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get deployment",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// GetDeploymentLogs handles GET /api/v1/deployments/:id/logs
func (h *DeploymentHandler) GetDeploymentLogs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid deployment ID",
			"message": "Deployment ID must be a valid UUID",
		})
		return
	}

	// Check if client accepts SSE
	acceptHeader := c.GetHeader("Accept")
	if acceptHeader == "text/event-stream" {
		h.streamDeploymentLogs(c, id)
		return
	}

	// Return logs as JSON
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	ctx := c.Request.Context()
	logs, err := h.deploymentService.GetDeploymentLogs(ctx, id, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get deployment logs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get deployment logs",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deployment_id": id,
		"logs":          logs,
	})
}

// GetDeploymentSteps handles GET /api/v1/deployments/:id/steps
func (h *DeploymentHandler) GetDeploymentSteps(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid deployment ID",
			"message": "Deployment ID must be a valid UUID",
		})
		return
	}

	ctx := c.Request.Context()
	steps, err := h.deploymentService.GetDeploymentSteps(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get deployment steps")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get deployment steps",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deployment_id": id,
		"steps":         steps,
	})
}

// streamDeploymentLogs streams deployment logs via Server-Sent Events
func (h *DeploymentHandler) streamDeploymentLogs(c *gin.Context, deploymentID uuid.UUID) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create a channel to signal client disconnect
	notify := c.Writer.CloseNotify()

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"deployment_id": deploymentID.String(),
		"timestamp":     time.Now().Format(time.RFC3339),
	})
	c.Writer.Flush()

	// Get initial logs
	ctx := c.Request.Context()
	logs, err := h.deploymentService.GetDeploymentLogs(ctx, deploymentID, 50)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get initial deployment logs")
		c.SSEvent("error", gin.H{
			"message": "Failed to get deployment logs",
		})
		c.Writer.Flush()
		return
	}

	// Send initial logs
	for _, log := range logs {
		c.SSEvent("log", log)
		c.Writer.Flush()
	}

	// TODO: Implement real-time log streaming
	// For now, we'll just keep the connection alive
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-notify:
			h.logger.WithField("deployment_id", deploymentID).Info("Client disconnected from log stream")
			return
		case <-ticker.C:
			// Send heartbeat
			c.SSEvent("heartbeat", gin.H{
				"timestamp": time.Now().Format(time.RFC3339),
			})
			c.Writer.Flush()
		}
	}
}

// ListDeployments handles GET /api/v1/deployments
func (h *DeploymentHandler) ListDeployments(c *gin.Context) {
	// TODO: Implement pagination and filtering
	c.JSON(http.StatusOK, gin.H{
		"message":     "List deployments endpoint - to be implemented",
		"deployments": []gin.H{},
	})
}
