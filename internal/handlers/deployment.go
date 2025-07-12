package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"deployknot/internal/middleware"
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
	// Get user ID from context
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not found in context",
		})
		return
	}

	var req models.CreateDeploymentRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind deployment request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Validate required fields
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Handle .env file upload
	var envFilePath string
	if file, err := c.FormFile("env_file"); err == nil && file != nil {
		// Create temp directory if it doesn't exist
		tempDir := "temp_env_files"
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			h.logger.WithError(err).Error("Failed to create temp directory")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to process environment file",
			})
			return
		}

		// Save uploaded file
		envFilePath = filepath.Join(tempDir, fmt.Sprintf("%s_%s", uuid.New().String(), file.Filename))
		if err := c.SaveUploadedFile(file, envFilePath); err != nil {
			h.logger.WithError(err).Error("Failed to save uploaded file")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Failed to save environment file",
			})
			return
		}

		h.logger.WithField("env_file_path", envFilePath).Info("Environment file uploaded successfully")
	}

	ctx := c.Request.Context()
	deployment, err := h.deploymentService.CreateDeploymentWithEnvFile(ctx, &req, envFilePath, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create deployment",
			"message": err.Error(),
		})
		return
	}

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

	ctx := c.Request.Context()
	var lastLogID uuid.UUID

	// Send initial logs
	logs, err := h.deploymentService.GetDeploymentLogs(ctx, deploymentID, 50)
	if err == nil {
		for _, log := range logs {
			c.SSEvent("log", log)
			c.Writer.Flush()
			if log.ID.String() > lastLogID.String() {
				lastLogID = log.ID
			}
		}
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-notify:
			h.logger.WithField("deployment_id", deploymentID).Info("Client disconnected from log stream")
			return
		case <-ticker.C:
			// Poll for new logs
			newLogs, err := h.deploymentService.GetDeploymentLogs(ctx, deploymentID, 100)
			if err == nil {
				for _, log := range newLogs {
					if log.ID.String() > lastLogID.String() {
						c.SSEvent("log", log)
						c.Writer.Flush()
						lastLogID = log.ID
					}
				}
			}
			// Send heartbeat
			c.SSEvent("heartbeat", gin.H{"timestamp": time.Now().Format(time.RFC3339)})
			c.Writer.Flush()
		}
	}
}

// GetDeployments handles GET /api/v1/deployments
func (h *DeploymentHandler) GetDeployments(c *gin.Context) {
	// Get user ID from context
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not found in context",
		})
		return
	}

	// Parse query parameters
	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := c.Request.Context()
	deployments, err := h.deploymentService.GetDeploymentsByUser(ctx, userID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get deployments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get deployments",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"deployments": deployments,
		"limit":       limit,
		"offset":      offset,
		"count":       len(deployments),
	})
}
