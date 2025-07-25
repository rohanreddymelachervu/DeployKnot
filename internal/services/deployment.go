package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"deployknot/internal/database"
	"deployknot/internal/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DeploymentService handles deployment business logic
type DeploymentService struct {
	repo   *database.Repository
	queue  *QueueService
	logger *logrus.Logger
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(repo *database.Repository, queue *QueueService, logger *logrus.Logger) *DeploymentService {
	return &DeploymentService{
		repo:   repo,
		queue:  queue,
		logger: logger,
	}
}

// CreateDeployment creates a new deployment
func (s *DeploymentService) CreateDeployment(ctx context.Context, req *models.CreateDeploymentRequest) (*models.DeploymentResponse, error) {
	// Convert port string to int
	port, err := req.GetPortAsInt()
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Generate deployment ID
	deploymentID := uuid.New()
	now := time.Now()

	// Generate container name if not provided
	containerName := s.generateContainerName(deploymentID, req.ContainerName, req.ProjectName, req.DeploymentName)

	// Create deployment record (no env vars stored in DB)
	deployment := &models.Deployment{
		ID:                   deploymentID,
		CreatedAt:            now,
		UpdatedAt:            now,
		Status:               models.DeploymentStatusPending,
		TargetIP:             req.TargetIP,
		SSHUsername:          req.SSHUsername,
		SSHPasswordEncrypted: &req.SSHPassword,
		GitHubRepoURL:        req.GitHubRepoURL,
		GitHubPATEncrypted:   &req.GitHubPAT,
		GitHubBranch:         req.GitHubBranch,
		Port:                 port,
		ContainerName:        &containerName,
		ProjectName:          req.ProjectName,
		DeploymentName:       req.DeploymentName,
		AdditionalVars:       req.AdditionalVars,
	}

	// Save to database
	if err := s.repo.CreateDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create initial deployment steps
	if err := s.createInitialSteps(deploymentID); err != nil {
		s.logger.WithError(err).Error("Failed to create initial deployment steps")
	}

	// Enqueue deployment job
	deploymentData := map[string]interface{}{
		"target_ip":       req.TargetIP,
		"ssh_username":    req.SSHUsername,
		"ssh_password":    req.SSHPassword,
		"github_repo_url": req.GitHubRepoURL,
		"github_pat":      req.GitHubPAT,
		"github_branch":   req.GitHubBranch,
		"port":            port,
		"container_name":  containerName,
		"project_name":    req.ProjectName,
		"deployment_name": req.DeploymentName,
		"additional_vars": req.AdditionalVars,
	}

	if err := s.queue.EnqueueDeploymentJob(ctx, deploymentID, deploymentData); err != nil {
		s.logger.WithError(err).Error("Failed to enqueue deployment job")
	}

	// Log the deployment creation
	s.logger.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"target_ip":     req.TargetIP,
		"repo_url":      req.GitHubRepoURL,
		"branch":        req.GitHubBranch,
	}).Info("Deployment created and enqueued successfully")

	// Return response
	response := &models.DeploymentResponse{
		ID:             deploymentID,
		Status:         models.DeploymentStatusPending,
		TargetIP:       req.TargetIP,
		GitHubRepoURL:  req.GitHubRepoURL,
		GitHubBranch:   req.GitHubBranch,
		Port:           port,
		ContainerName:  &containerName,
		CreatedAt:      now,
		ProjectName:    req.ProjectName,
		DeploymentName: req.DeploymentName,
	}

	return response, nil
}

// CreateDeploymentWithEnvFile creates a new deployment and handles env_file uploads
func (s *DeploymentService) CreateDeploymentWithEnvFile(ctx context.Context, req *models.CreateDeploymentRequest, envFilePath string, userID uuid.UUID) (*models.DeploymentResponse, error) {
	// Convert port string to int
	port, err := req.GetPortAsInt()
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Generate deployment ID
	deploymentID := uuid.New()
	now := time.Now()

	// Generate container name if not provided
	containerName := s.generateContainerName(deploymentID, req.ContainerName, req.ProjectName, req.DeploymentName)

	// Create deployment record (no env vars stored in DB)
	deployment := &models.Deployment{
		ID:                   deploymentID,
		CreatedAt:            now,
		UpdatedAt:            now,
		Status:               models.DeploymentStatusPending,
		TargetIP:             req.TargetIP,
		SSHUsername:          req.SSHUsername,
		SSHPasswordEncrypted: &req.SSHPassword,
		GitHubRepoURL:        req.GitHubRepoURL,
		GitHubPATEncrypted:   &req.GitHubPAT,
		GitHubBranch:         req.GitHubBranch,
		Port:                 port,
		ContainerName:        &containerName,
		ProjectName:          req.ProjectName,
		DeploymentName:       req.DeploymentName,
		AdditionalVars:       req.AdditionalVars,
		UserID:               &userID,
	}

	// Save to database
	if err := s.repo.CreateDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create initial deployment steps
	if err := s.createInitialSteps(deploymentID); err != nil {
		s.logger.WithError(err).Error("Failed to create initial deployment steps")
	}

	// Enqueue deployment job
	deploymentData := map[string]interface{}{
		"target_ip":       req.TargetIP,
		"ssh_username":    req.SSHUsername,
		"ssh_password":    req.SSHPassword,
		"github_repo_url": req.GitHubRepoURL,
		"github_pat":      req.GitHubPAT,
		"github_branch":   req.GitHubBranch,
		"port":            port,
		"container_name":  containerName,
		"project_name":    req.ProjectName,
		"deployment_name": req.DeploymentName,
		"additional_vars": req.AdditionalVars,
	}
	if envFilePath != "" {
		deploymentData["env_file_path"] = envFilePath
	}

	if err := s.queue.EnqueueDeploymentJob(ctx, deploymentID, deploymentData); err != nil {
		s.logger.WithError(err).Error("Failed to enqueue deployment job")
	}

	// Log the deployment creation
	s.logger.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"user_id":       userID,
		"target_ip":     req.TargetIP,
		"repo_url":      req.GitHubRepoURL,
		"branch":        req.GitHubBranch,
	}).Info("Deployment created and enqueued successfully")

	// Return response
	response := &models.DeploymentResponse{
		ID:             deploymentID,
		Status:         models.DeploymentStatusPending,
		TargetIP:       req.TargetIP,
		GitHubRepoURL:  req.GitHubRepoURL,
		GitHubBranch:   req.GitHubBranch,
		Port:           port,
		ContainerName:  &containerName,
		CreatedAt:      now,
		ProjectName:    req.ProjectName,
		DeploymentName: req.DeploymentName,
	}

	return response, nil
}

// GetDeployment retrieves a deployment by ID
func (s *DeploymentService) GetDeployment(ctx context.Context, id uuid.UUID) (*models.DeploymentResponse, error) {
	deployment, err := s.repo.GetDeployment(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Convert to response format
	response := &models.DeploymentResponse{
		ID:             deployment.ID,
		Status:         deployment.Status,
		TargetIP:       deployment.TargetIP,
		GitHubRepoURL:  deployment.GitHubRepoURL,
		GitHubBranch:   deployment.GitHubBranch,
		Port:           deployment.Port,
		ContainerName:  deployment.ContainerName,
		CreatedAt:      deployment.CreatedAt,
		StartedAt:      deployment.StartedAt,
		CompletedAt:    deployment.CompletedAt,
		ErrorMessage:   deployment.ErrorMessage,
		ProjectName:    deployment.ProjectName,
		DeploymentName: deployment.DeploymentName,
	}

	return response, nil
}

// GetDeploymentLogs retrieves logs for a deployment
func (s *DeploymentService) GetDeploymentLogs(ctx context.Context, deploymentID uuid.UUID, limit int) ([]*models.DeploymentLog, error) {
	logs, err := s.repo.GetDeploymentLogs(deploymentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}

	return logs, nil
}

// GetDeploymentSteps retrieves steps for a deployment
func (s *DeploymentService) GetDeploymentSteps(ctx context.Context, deploymentID uuid.UUID) ([]*models.DeploymentStep, error) {
	steps, err := s.repo.GetDeploymentSteps(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment steps: %w", err)
	}

	return steps, nil
}

// UpdateDeploymentStatus updates the deployment status
func (s *DeploymentService) UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status models.DeploymentStatus, errorMessage *string) error {
	if err := s.repo.UpdateDeploymentStatus(deploymentID, status, errorMessage); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"status":        status,
		"error":         errorMessage,
	}).Info("Deployment status updated")

	return nil
}

// AddDeploymentLog adds a log entry to a deployment
func (s *DeploymentService) AddDeploymentLog(ctx context.Context, deploymentID uuid.UUID, level, message, taskName string, stepOrder *int) error {
	log := &models.DeploymentLog{
		ID:           uuid.New(),
		DeploymentID: deploymentID,
		CreatedAt:    time.Now(),
		LogLevel:     level,
		Message:      message,
		TaskName:     &taskName,
		StepOrder:    stepOrder,
	}

	if err := s.repo.CreateDeploymentLog(log); err != nil {
		return fmt.Errorf("failed to create deployment log: %w", err)
	}

	return nil
}

// UpdateDeploymentStep updates a deployment step
func (s *DeploymentService) UpdateDeploymentStep(ctx context.Context, step *models.DeploymentStep) error {
	if err := s.repo.UpdateDeploymentStep(step); err != nil {
		return fmt.Errorf("failed to update deployment step: %w", err)
	}

	return nil
}

// createInitialSteps creates the initial deployment steps
func (s *DeploymentService) createInitialSteps(deploymentID uuid.UUID) error {
	steps := []struct {
		name  string
		order int
	}{
		{"validate_credentials", 1},
		{"git_clone", 2},
		{"docker_build", 3},
		{"docker_run", 4},
		{"health_check", 5},
	}

	for _, stepInfo := range steps {
		step := &models.DeploymentStep{
			ID:           uuid.New(),
			DeploymentID: deploymentID,
			StepName:     stepInfo.name,
			Status:       models.DeploymentStatusPending,
			StepOrder:    stepInfo.order,
		}

		if err := s.repo.CreateDeploymentStep(step); err != nil {
			return fmt.Errorf("failed to create step %s: %w", stepInfo.name, err)
		}
	}

	return nil
}

// ValidateDeploymentRequest validates the deployment request
func (s *DeploymentService) ValidateDeploymentRequest(req *models.CreateDeploymentRequest) error {
	if req.TargetIP == "" {
		return fmt.Errorf("target_ip is required")
	}

	if req.SSHUsername == "" {
		return fmt.Errorf("ssh_username is required")
	}

	if req.SSHPassword == "" {
		return fmt.Errorf("ssh_password is required")
	}

	if req.GitHubRepoURL == "" {
		return fmt.Errorf("github_repo_url is required")
	}

	if req.GitHubPAT == "" {
		return fmt.Errorf("github_pat is required")
	}

	if req.GitHubBranch == "" {
		return fmt.Errorf("github_branch is required")
	}

	// Validate port using the new conversion method
	if _, err := req.GetPortAsInt(); err != nil {
		return fmt.Errorf("port validation failed: %w", err)
	}

	return nil
}

// generateContainerName generates a unique container name for the deployment
func (s *DeploymentService) generateContainerName(deploymentID uuid.UUID, containerName, projectName, deploymentName *string) string {
	// If container name is provided, use it
	if containerName != nil && *containerName != "" {
		return *containerName
	}

	// Generate based on project and deployment name if available
	if projectName != nil && *projectName != "" && deploymentName != nil && *deploymentName != "" {
		// Sanitize names for Docker container naming (lowercase, alphanumeric, hyphens only)
		project := sanitizeContainerName(*projectName)
		deployment := sanitizeContainerName(*deploymentName)
		return fmt.Sprintf("deployknot-%s-%s", project, deployment)
	}

	// Fallback to deployment ID
	return fmt.Sprintf("deployknot-%s", deploymentID.String())
}

// sanitizeContainerName sanitizes a string for use as a Docker container name
func sanitizeContainerName(name string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	// Keep only alphanumeric and hyphens
	var result []rune
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		} else {
			result = append(result, '-')
		}
	}

	// Remove consecutive hyphens and trim
	sanitized := strings.Trim(string(result), "-")

	// Ensure it's not empty and has reasonable length
	if sanitized == "" {
		sanitized = "app"
	}
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}

// GetDeploymentsByUser gets deployments for a specific user
func (s *DeploymentService) GetDeploymentsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.DeploymentResponse, error) {
	deployments, err := s.repo.GetDeploymentsByUserID(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments by user: %w", err)
	}

	var responses []*models.DeploymentResponse
	for _, deployment := range deployments {
		response := &models.DeploymentResponse{
			ID:             deployment.ID,
			Status:         deployment.Status,
			TargetIP:       deployment.TargetIP,
			GitHubRepoURL:  deployment.GitHubRepoURL,
			GitHubBranch:   deployment.GitHubBranch,
			Port:           deployment.Port,
			ContainerName:  deployment.ContainerName,
			CreatedAt:      deployment.CreatedAt,
			StartedAt:      deployment.StartedAt,
			CompletedAt:    deployment.CompletedAt,
			ErrorMessage:   deployment.ErrorMessage,
			ProjectName:    deployment.ProjectName,
			DeploymentName: deployment.DeploymentName,
		}
		responses = append(responses, response)
	}

	return responses, nil
}
