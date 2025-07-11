package services

import (
	"context"
	"fmt"
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
	// Generate deployment ID
	deploymentID := uuid.New()
	now := time.Now()

	// Create deployment record
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
		EnvironmentVars:      &req.EnvironmentVars,
		AdditionalVars:       req.AdditionalVars,
		ProjectName:          req.ProjectName,
		DeploymentName:       req.DeploymentName,
	}

	// Save to database
	if err := s.repo.CreateDeployment(deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create initial deployment steps
	if err := s.createInitialSteps(deploymentID); err != nil {
		s.logger.WithError(err).Error("Failed to create initial deployment steps")
		// Don't fail the entire request if steps creation fails
	}

	// Enqueue deployment job
	deploymentData := map[string]interface{}{
		"target_ip":        req.TargetIP,
		"ssh_username":     req.SSHUsername,
		"ssh_password":     req.SSHPassword,
		"github_repo_url":  req.GitHubRepoURL,
		"github_pat":       req.GitHubPAT,
		"github_branch":    req.GitHubBranch,
		"environment_vars": req.EnvironmentVars,
		"additional_vars":  req.AdditionalVars,
		"project_name":     req.ProjectName,
		"deployment_name":  req.DeploymentName,
	}

	if err := s.queue.EnqueueDeploymentJob(ctx, deploymentID, deploymentData); err != nil {
		s.logger.WithError(err).Error("Failed to enqueue deployment job")
		// Don't fail the entire request if queue fails
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

	return nil
}
