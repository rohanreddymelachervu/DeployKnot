package models

import (
	"fmt"
	"strconv"
	"strings"
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
	DeploymentStatusAborted   DeploymentStatus = "aborted"
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
	Port                 int                    `json:"port" db:"port"`
	ContainerName        *string                `json:"container_name,omitempty" db:"container_name"`
	StartedAt            *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage         *string                `json:"error_message,omitempty" db:"error_message"`
	CreatedBy            *string                `json:"created_by,omitempty" db:"created_by"`
	ProjectName          *string                `json:"project_name,omitempty" db:"project_name"`
	DeploymentName       *string                `json:"deployment_name,omitempty" db:"deployment_name"`
	UserID               *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
}

// CreateDeploymentRequest represents the request to create a deployment
// For multipart form: all fields are form fields except env_file, which is a file upload
// Use binding:"required" for required fields
type CreateDeploymentRequest struct {
	TargetIP       string  `form:"target_ip" binding:"required,ip"`
	SSHUsername    string  `form:"ssh_username" binding:"required"`
	SSHPassword    string  `form:"ssh_password" binding:"required"`
	GitHubRepoURL  string  `form:"github_repo_url" binding:"required"`
	GitHubPAT      string  `form:"github_pat" binding:"required"`
	GitHubBranch   string  `form:"github_branch" binding:"required"`
	Port           string  `form:"port" binding:"required"` // Will be converted to int
	ContainerName  *string `form:"container_name"`
	ProjectName    *string `form:"project_name"`
	DeploymentName *string `form:"deployment_name"`
	// env_file is handled as a file upload in the handler, not as a struct field
	// AdditionalVars can be handled as a JSON string if needed
	AdditionalVars map[string]interface{} `form:"additional_vars"`
}

// Validate validates the deployment request
func (req *CreateDeploymentRequest) Validate() error {
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
	if req.Port == "" {
		return fmt.Errorf("port is required")
	}
	return nil
}

// GetPortAsInt converts the Port string to int
func (r *CreateDeploymentRequest) GetPortAsInt() (int, error) {
	if r.Port == "" {
		return 0, fmt.Errorf("port is required")
	}

	port, err := strconv.Atoi(r.Port)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", r.Port)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}

	return port, nil
}

// EnvironmentVariable represents a single environment variable
type EnvironmentVariable struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// EnvironmentVariables represents a collection of environment variables
type EnvironmentVariables []EnvironmentVariable

// ToEnvFile converts environment variables to .env file format
func (ev EnvironmentVariables) ToEnvFile() string {
	var lines []string
	for _, env := range ev {
		lines = append(lines, fmt.Sprintf("%s=%s", env.Key, env.Value))
	}
	return strings.Join(lines, "\n")
}

// FromEnvFile parses .env file content into EnvironmentVariables
func FromEnvFile(content string) EnvironmentVariables {
	var envVars EnvironmentVariables
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove quotes if they exist
				value = strings.Trim(value, `"'`)

				envVars = append(envVars, EnvironmentVariable{
					Key:   key,
					Value: value,
				})
			}
		}
	}

	return envVars
}

// DeploymentResponse represents the response for a deployment
type DeploymentResponse struct {
	ID             uuid.UUID        `json:"id"`
	Status         DeploymentStatus `json:"status"`
	TargetIP       string           `json:"target_ip"`
	GitHubRepoURL  string           `json:"github_repo_url"`
	GitHubBranch   string           `json:"github_branch"`
	Port           int              `json:"port"`
	ContainerName  *string          `json:"container_name,omitempty"`
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
