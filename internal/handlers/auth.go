package handlers

import (
	"net/http"

	"deployknot/internal/middleware"
	"deployknot/internal/models"
	"deployknot/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	userService    *services.UserService
	authMiddleware *middleware.AuthMiddleware
	logger         *logrus.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *services.UserService, authMiddleware *middleware.AuthMiddleware, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		userService:    userService,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind register request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	user, err := h.userService.RegisterUser(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to register user")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Registration failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	loginResponse, err := h.userService.LoginUser(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to login user")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Login failed",
			"message": err.Error(),
		})
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.authMiddleware.GenerateToken(&models.User{
		ID:       loginResponse.User.ID,
		Username: loginResponse.User.Username,
		Email:    loginResponse.User.Email,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate JWT token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Authentication failed",
			"message": "Failed to generate token",
		})
		return
	}

	loginResponse.Token = token
	loginResponse.ExpiresAt = expiresAt

	c.JSON(http.StatusOK, loginResponse)
}

// GetProfile handles GET /api/v1/auth/profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not found in context",
		})
		return
	}

	ctx := c.Request.Context()
	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get profile",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
