package models

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusCompleted DeploymentStatus = "completed"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

// Deployment represents a deployment record
type Deployment struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	Status               DeploymentStatus       `json:"status" db:"status"`
	TargetIP             string                 `json:"target_ip" db:"target_ip"`
	SSHUsername          string                 `json:"ssh_username" db:"ssh_username"`
	SSHPasswordEncrypted *string                `json:"-" db:"ssh_password_encrypted"`
	GitHubRepoURL        string                 `json:"github_repo_url" db:"github_repo_url"`
	GitHubPATEncrypted   *string                `json:"-" db:"github_pat_encrypted"`
	GitHubBranch         string                 `json:"github_branch" db:"github_branch"`
	EnvironmentVars      *string                `json:"environment_vars,omitempty" db:"environment_vars"`
	AdditionalVars       map[string]interface{} `json:"additional_vars,omitempty" db:"additional_vars"`
	StartedAt            *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage         *string                `json:"error_message,omitempty" db:"error_message"`
	CreatedBy            *string                `json:"created_by,omitempty" db:"created_by"`
	ProjectName          *string                `json:"project_name,omitempty" db:"project_name"`
	DeploymentName       *string                `json:"deployment_name,omitempty" db:"deployment_name"`
}

// CreateDeploymentRequest represents the request to create a deployment
type CreateDeploymentRequest struct {
	TargetIP        string                 `json:"target_ip" binding:"required,ip"`
	SSHUsername     string                 `json:"ssh_username" binding:"required"`
	SSHPassword     string                 `json:"ssh_password" binding:"required"`
	GitHubRepoURL   string                 `json:"github_repo_url" binding:"required"`
	GitHubPAT       string                 `json:"github_pat" binding:"required"`
	GitHubBranch    string                 `json:"github_branch" binding:"required"`
	EnvironmentVars string                 `json:"environment_vars,omitempty"`
	AdditionalVars  map[string]interface{} `json:"additional_vars,omitempty"`
	ProjectName     *string                `json:"project_name,omitempty"`
	DeploymentName  *string                `json:"deployment_name,omitempty"`
}

// DeploymentResponse represents the response for a deployment
type DeploymentResponse struct {
	ID             uuid.UUID        `json:"id"`
	Status         DeploymentStatus `json:"status"`
	TargetIP       string           `json:"target_ip"`
	GitHubRepoURL  string           `json:"github_repo_url"`
	GitHubBranch   string           `json:"github_branch"`
	CreatedAt      time.Time        `json:"created_at"`
	StartedAt      *time.Time       `json:"started_at,omitempty"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	ErrorMessage   *string          `json:"error_message,omitempty"`
	ProjectName    *string          `json:"project_name,omitempty"`
	DeploymentName *string          `json:"deployment_name,omitempty"`
}

// DeploymentLog represents a deployment log entry
type DeploymentLog struct {
	ID           uuid.UUID `json:"id" db:"id"`
	DeploymentID uuid.UUID `json:"deployment_id" db:"deployment_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	LogLevel     string    `json:"log_level" db:"log_level"`
	Message      string    `json:"message" db:"message"`
	TaskName     *string   `json:"task_name,omitempty" db:"task_name"`
	StepOrder    *int      `json:"step_order,omitempty" db:"step_order"`
}

// DeploymentStep represents a deployment step
type DeploymentStep struct {
	ID           uuid.UUID        `json:"id" db:"id"`
	DeploymentID uuid.UUID        `json:"deployment_id" db:"deployment_id"`
	StepName     string           `json:"step_name" db:"step_name"`
	Status       DeploymentStatus `json:"status" db:"status"`
	StartedAt    *time.Time       `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs   *int             `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage *string          `json:"error_message,omitempty" db:"error_message"`
	StepOrder    int              `json:"step_order" db:"step_order"`
}
