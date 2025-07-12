package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"deployknot/internal/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Repository handles database operations
type Repository struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewRepository creates a new repository instance
func NewRepository(db *sql.DB, logger *logrus.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// CreateDeployment creates a new deployment record
func (r *Repository) CreateDeployment(deployment *models.Deployment) error {
	query := `
		INSERT INTO deploy_knot.deployments (
			id, created_at, updated_at, status, target_ip, ssh_username, 
			ssh_password_encrypted, github_repo_url, github_pat_encrypted, 
			github_branch, additional_vars, port, container_name, created_by, 
			project_name, deployment_name, user_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	// For now, we'll store passwords as-is (in production, encrypt these)
	sshPasswordEncrypted := deployment.SSHPasswordEncrypted
	githubPATEncrypted := deployment.GitHubPATEncrypted

	// Convert AdditionalVars to JSON bytes
	var additionalVarsJSON []byte
	if deployment.AdditionalVars != nil {
		var err error
		additionalVarsJSON, err = json.Marshal(deployment.AdditionalVars)
		if err != nil {
			return fmt.Errorf("failed to marshal additional_vars: %w", err)
		}
		r.logger.WithField("additional_vars", string(additionalVarsJSON)).Debug("Marshaled additional_vars")
	} else {
		additionalVarsJSON = []byte("null")
		r.logger.Debug("Using null for additional_vars")
	}

	r.logger.WithFields(logrus.Fields{
		"additional_vars_type":            fmt.Sprintf("%T", additionalVarsJSON),
		"additional_vars_value":           string(additionalVarsJSON),
		"deployment_id":                   deployment.ID,
		"deployment_additional_vars_type": fmt.Sprintf("%T", deployment.AdditionalVars),
		"deployment_additional_vars_nil":  deployment.AdditionalVars == nil,
	}).Debug("About to execute deployment insert")

	// Log all parameters being passed to Exec
	params := []interface{}{
		deployment.ID,
		deployment.CreatedAt,
		deployment.UpdatedAt,
		deployment.Status,
		deployment.TargetIP,
		deployment.SSHUsername,
		sshPasswordEncrypted,
		deployment.GitHubRepoURL,
		githubPATEncrypted,
		deployment.GitHubBranch,
		additionalVarsJSON,
		deployment.Port,
		deployment.ContainerName,
		deployment.CreatedBy,
		deployment.ProjectName,
		deployment.DeploymentName,
		deployment.UserID,
	}

	r.logger.WithField("param_count", len(params)).Debug("Exec parameters prepared")

	for i, param := range params {
		r.logger.WithFields(logrus.Fields{
			"param_index": i + 1,
			"param_type":  fmt.Sprintf("%T", param),
			"param_value": fmt.Sprintf("%v", param),
		}).Debug("Parameter details")
	}

	_, err := r.db.Exec(query, params...)

	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// GetDeployment retrieves a deployment by ID
func (r *Repository) GetDeployment(id uuid.UUID) (*models.Deployment, error) {
	query := `
		SELECT id, created_at, updated_at, status, target_ip, ssh_username,
		       ssh_password_encrypted, github_repo_url, github_pat_encrypted,
		       github_branch, additional_vars, port, container_name, started_at, 
		       completed_at, error_message, created_by, project_name, deployment_name
		FROM deploy_knot.deployments
		WHERE id = $1
	`

	deployment := &models.Deployment{}
	var additionalVarsJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&deployment.ID,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
		&deployment.Status,
		&deployment.TargetIP,
		&deployment.SSHUsername,
		&deployment.SSHPasswordEncrypted,
		&deployment.GitHubRepoURL,
		&deployment.GitHubPATEncrypted,
		&deployment.GitHubBranch,
		&additionalVarsJSON,
		&deployment.Port,
		&deployment.ContainerName,
		&deployment.StartedAt,
		&deployment.CompletedAt,
		&deployment.ErrorMessage,
		&deployment.CreatedBy,
		&deployment.ProjectName,
		&deployment.DeploymentName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deployment not found")
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Parse additional_vars JSON
	if additionalVarsJSON != nil {
		if err := json.Unmarshal(additionalVarsJSON, &deployment.AdditionalVars); err != nil {
			r.logger.WithError(err).Warn("Failed to parse additional_vars JSON")
		}
	}

	return deployment, nil
}

// UpdateDeploymentStatus updates the deployment status
func (r *Repository) UpdateDeploymentStatus(id uuid.UUID, status models.DeploymentStatus, errorMessage *string) error {
	query := `
		UPDATE deploy_knot.deployments
		SET status = $2, updated_at = $3, error_message = $4
		WHERE id = $1
	`

	_, err := r.db.Exec(query, id, status, time.Now(), errorMessage)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	return nil
}

// UpdateDeploymentTiming updates deployment timing fields
func (r *Repository) UpdateDeploymentTiming(id uuid.UUID, startedAt, completedAt *time.Time) error {
	query := `
		UPDATE deploy_knot.deployments
		SET started_at = $2, completed_at = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.Exec(query, id, startedAt, completedAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update deployment timing: %w", err)
	}

	return nil
}

// CreateDeploymentLog creates a new deployment log entry
func (r *Repository) CreateDeploymentLog(log *models.DeploymentLog) error {
	query := `
		INSERT INTO deploy_knot.deployment_logs (
			id, deployment_id, created_at, log_level, message, task_name, step_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(query,
		log.ID,
		log.DeploymentID,
		log.CreatedAt,
		log.LogLevel,
		log.Message,
		log.TaskName,
		log.StepOrder,
	)

	if err != nil {
		return fmt.Errorf("failed to create deployment log: %w", err)
	}

	return nil
}

// GetDeploymentLogs retrieves logs for a deployment
func (r *Repository) GetDeploymentLogs(deploymentID uuid.UUID, limit int) ([]*models.DeploymentLog, error) {
	query := `
		SELECT id, deployment_id, created_at, log_level, message, task_name, step_order
		FROM deploy_knot.deployment_logs
		WHERE deployment_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(query, deploymentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.DeploymentLog
	for rows.Next() {
		log := &models.DeploymentLog{}
		err := rows.Scan(
			&log.ID,
			&log.DeploymentID,
			&log.CreatedAt,
			&log.LogLevel,
			&log.Message,
			&log.TaskName,
			&log.StepOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// CreateDeploymentStep creates a new deployment step
func (r *Repository) CreateDeploymentStep(step *models.DeploymentStep) error {
	query := `
		INSERT INTO deploy_knot.deployment_steps (
			id, deployment_id, step_name, status, started_at, completed_at,
			duration_ms, error_message, step_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(query,
		step.ID,
		step.DeploymentID,
		step.StepName,
		step.Status,
		step.StartedAt,
		step.CompletedAt,
		step.DurationMs,
		step.ErrorMessage,
		step.StepOrder,
	)

	if err != nil {
		return fmt.Errorf("failed to create deployment step: %w", err)
	}

	return nil
}

// UpdateDeploymentStep updates a deployment step
func (r *Repository) UpdateDeploymentStep(step *models.DeploymentStep) error {
	query := `
		UPDATE deploy_knot.deployment_steps
		SET status = $2, started_at = $3, completed_at = $4,
		    duration_ms = $5, error_message = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(query,
		step.ID,
		step.Status,
		step.StartedAt,
		step.CompletedAt,
		step.DurationMs,
		step.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to update deployment step: %w", err)
	}

	return nil
}

// GetDeploymentSteps retrieves steps for a deployment
func (r *Repository) GetDeploymentSteps(deploymentID uuid.UUID) ([]*models.DeploymentStep, error) {
	query := `
		SELECT id, deployment_id, step_name, status, started_at, completed_at,
		       duration_ms, error_message, step_order
		FROM deploy_knot.deployment_steps
		WHERE deployment_id = $1
		ORDER BY step_order ASC
	`

	rows, err := r.db.Query(query, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment steps: %w", err)
	}
	defer rows.Close()

	var steps []*models.DeploymentStep
	for rows.Next() {
		step := &models.DeploymentStep{}
		err := rows.Scan(
			&step.ID,
			&step.DeploymentID,
			&step.StepName,
			&step.Status,
			&step.StartedAt,
			&step.CompletedAt,
			&step.DurationMs,
			&step.ErrorMessage,
			&step.StepOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment step: %w", err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// CreateUser creates a new user
func (r *Repository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO deploy_knot.users (
			id, username, email, password_hash, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM deploy_knot.users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM deploy_knot.users
		WHERE username = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM deploy_knot.users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetDeploymentsByUserID retrieves deployments for a specific user
func (r *Repository) GetDeploymentsByUserID(userID uuid.UUID, limit, offset int) ([]*models.Deployment, error) {
	query := `
		SELECT id, created_at, updated_at, status, target_ip, ssh_username,
		       ssh_password_encrypted, github_repo_url, github_pat_encrypted,
		       github_branch, additional_vars, port, container_name, started_at, 
		       completed_at, error_message, created_by, project_name, deployment_name, user_id
		FROM deploy_knot.deployments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments by user: %w", err)
	}
	defer rows.Close()

	var deployments []*models.Deployment
	for rows.Next() {
		deployment := &models.Deployment{}
		var additionalVarsJSON []byte

		err := rows.Scan(
			&deployment.ID,
			&deployment.CreatedAt,
			&deployment.UpdatedAt,
			&deployment.Status,
			&deployment.TargetIP,
			&deployment.SSHUsername,
			&deployment.SSHPasswordEncrypted,
			&deployment.GitHubRepoURL,
			&deployment.GitHubPATEncrypted,
			&deployment.GitHubBranch,
			&additionalVarsJSON,
			&deployment.Port,
			&deployment.ContainerName,
			&deployment.StartedAt,
			&deployment.CompletedAt,
			&deployment.ErrorMessage,
			&deployment.CreatedBy,
			&deployment.ProjectName,
			&deployment.DeploymentName,
			&deployment.UserID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}

		// Parse additional_vars JSON
		if additionalVarsJSON != nil {
			if err := json.Unmarshal(additionalVarsJSON, &deployment.AdditionalVars); err != nil {
				r.logger.WithError(err).Warn("Failed to parse additional_vars JSON")
			}
		}

		deployments = append(deployments, deployment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deployments: %w", err)
	}

	return deployments, nil
}
