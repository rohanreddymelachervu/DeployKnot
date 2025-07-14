package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"deployknot/internal/config"
	"deployknot/internal/database"
	"deployknot/internal/models"
	"deployknot/internal/services"
	"deployknot/pkg/logger"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Worker represents the deployment worker
type Worker struct {
	queueService      *services.QueueService
	deploymentService *services.DeploymentService
	logger            *logrus.Logger
	sshClient         *ssh.Client
}

// NewWorker creates a new worker instance
func NewWorker(queueService *services.QueueService, deploymentService *services.DeploymentService, logger *logrus.Logger) *Worker {
	return &Worker{
		queueService:      queueService,
		deploymentService: deploymentService,
		logger:            logger,
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting deployment worker...")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker context cancelled, shutting down...")
			return nil
		default:
			// Dequeue a job
			job, err := w.queueService.DequeueJob(ctx)
			if err != nil {
				w.logger.WithError(err).Error("Failed to dequeue job")
				time.Sleep(5 * time.Second)
				continue
			}

			if job == nil {
				// No jobs available, wait a bit
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the job
			w.logger.WithField("job_id", job.ID).Info("Processing deployment job")
			if err := w.processDeploymentJob(ctx, job); err != nil {
				w.logger.WithError(err).Error("Failed to process deployment job")
				// Update job status to failed
				errorMsg := err.Error()
				w.queueService.UpdateJobStatus(ctx, job.ID, services.JobStatusFailed, &errorMsg)
			}
		}
	}
}

// processDeploymentJob processes a deployment job
func (w *Worker) processDeploymentJob(ctx context.Context, job *services.Job) error {
	w.logger.WithFields(logrus.Fields{
		"job_id":        job.ID,
		"deployment_id": job.DeploymentID,
	}).Info("Processing deployment job")

	// Update deployment status to running
	if err := w.deploymentService.UpdateDeploymentStatus(ctx, job.DeploymentID, models.DeploymentStatusRunning, nil); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Add log entry
	w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "info", "Starting deployment process", "deployment_start", nil)

	// Extract deployment data using robust helpers
	targetIP := getStringFromMap(job.Data, "target_ip")
	sshUsername := getStringFromMap(job.Data, "ssh_username")
	sshPassword := getStringFromMap(job.Data, "ssh_password")
	githubRepoURL := getStringFromMap(job.Data, "github_repo_url")
	githubPAT := getStringFromMap(job.Data, "github_pat")
	githubBranch := getStringFromMap(job.Data, "github_branch")
	port := getIntFromMap(job.Data, "port")
	containerName := getStringFromMap(job.Data, "container_name")
	// New: env_file_path
	envFilePath := getStringFromMap(job.Data, "env_file_path")
	environmentVars := getStringFromMap(job.Data, "environment_vars") // fallback only

	w.logger.WithFields(logrus.Fields{
		"target_ip":             targetIP,
		"ssh_username":          sshUsername,
		"ssh_password_length":   len(sshPassword),
		"github_repo_url":       githubRepoURL,
		"github_pat_length":     len(githubPAT),
		"github_branch":         githubBranch,
		"env_file_path":         envFilePath,
		"env_vars_length":       len(environmentVars),
		"port":                  port,
		"container_name":        containerName,
		"container_name_length": len(containerName),
		"job_data_keys":         getMapKeys(job.Data),
	}).Info("Extracted deployment credentials")

	// Validate required fields
	if targetIP == "" || sshUsername == "" || sshPassword == "" || githubRepoURL == "" || githubPAT == "" || githubBranch == "" {
		errorMsg := "missing required deployment parameters"
		w.markAllStepsAsFailed(ctx, job.DeploymentID, errorMsg)
		return fmt.Errorf("%s", errorMsg)
	}

	// Connect to target server via SSH
	sshClient, err := w.connectSSH(targetIP, sshUsername, sshPassword)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to connect to target server: %v", err)
		w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "error", errorMsg, "ssh_connect", nil)
		w.markStepAsFailed(ctx, 1, job.DeploymentID, errorMsg)
		w.markRemainingStepsAsFailed(ctx, job.DeploymentID, 1)
		// Update deployment status to failed
		if updateErr := w.deploymentService.UpdateDeploymentStatus(ctx, job.DeploymentID, models.DeploymentStatusFailed, &errorMsg); updateErr != nil {
			w.logger.WithError(updateErr).Error("Failed to update deployment status to failed")
		}
		return fmt.Errorf("failed to connect to target server: %w", err)
	}
	defer sshClient.Close()

	w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "info", "SSH connection established", "ssh_connect", nil)

	// Execute deployment steps (pass envFilePath and environmentVars)
	if err := w.executeDeploymentSteps(ctx, job.DeploymentID, sshClient, githubRepoURL, githubPAT, githubBranch, envFilePath, environmentVars, port, containerName); err != nil {
		errorMsg := fmt.Sprintf("Deployment failed: %v", err)
		w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "error", errorMsg, "deployment_failed", nil)

		// Update deployment status to failed
		if updateErr := w.deploymentService.UpdateDeploymentStatus(ctx, job.DeploymentID, models.DeploymentStatusFailed, &errorMsg); updateErr != nil {
			w.logger.WithError(updateErr).Error("Failed to update deployment status to failed")
		}

		return err
	}

	// Update deployment status to completed
	if err := w.deploymentService.UpdateDeploymentStatus(ctx, job.DeploymentID, models.DeploymentStatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "info", "Deployment completed successfully", "deployment_complete", nil)

	// Update job status to completed
	if err := w.queueService.UpdateJobStatus(ctx, job.ID, services.JobStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update job status to completed")
	}

	w.logger.WithField("deployment_id", job.DeploymentID).Info("Deployment completed successfully")
	return nil
}

// connectSSH establishes SSH connection to the target server
func (w *Worker) connectSSH(host, username, password string) (*ssh.Client, error) {
	w.logger.WithFields(logrus.Fields{
		"host":            host,
		"username":        username,
		"password_length": len(password),
	}).Info("Attempting SSH connection")

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		w.logger.WithError(err).Error("SSH connection failed")
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	w.logger.Info("SSH connection established successfully")
	return client, nil
}

// executeDeploymentSteps executes the deployment steps
func (w *Worker) executeDeploymentSteps(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, repoURL, pat, branch, envFilePath, envVars string, port int, containerName string) error {
	// Step 1: Clone the repository
	if err := w.cloneRepository(ctx, deploymentID, sshClient, repoURL, pat, branch); err != nil {
		w.markRemainingStepsAsFailed(ctx, deploymentID, 1)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Step 2: Build Docker image
	if err := w.buildDockerImage(ctx, deploymentID, sshClient, containerName); err != nil {
		w.markRemainingStepsAsFailed(ctx, deploymentID, 2)
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Step 3: Run Docker container
	if envFilePath != "" {
		// Copy env file to target instance
		if err := w.copyEnvFileToTarget(ctx, deploymentID, sshClient, envFilePath); err != nil {
			w.markRemainingStepsAsFailed(ctx, deploymentID, 3)
			return fmt.Errorf("failed to copy env file to target: %w", err)
		}
		if err := w.runDockerContainerWithEnvFile(ctx, deploymentID, sshClient, envFilePath, port, containerName); err != nil {
			w.markRemainingStepsAsFailed(ctx, deploymentID, 3)
			return fmt.Errorf("failed to run Docker container with env file: %w", err)
		}
	} else {
		if err := w.runDockerContainer(ctx, deploymentID, sshClient, envVars, port, containerName); err != nil {
			w.markRemainingStepsAsFailed(ctx, deploymentID, 3)
			return fmt.Errorf("failed to run Docker container: %w", err)
		}
	}

	// Step 4: Health check
	if err := w.healthCheck(ctx, deploymentID, sshClient, containerName); err != nil {
		w.markRemainingStepsAsFailed(ctx, deploymentID, 4)
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// cloneRepository clones the Git repository
func (w *Worker) cloneRepository(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, repoURL, pat, branch string) error {
	// Update step status to running
	if err := w.updateDeploymentStep(ctx, deploymentID, 1, models.DeploymentStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to running")
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting repository clone", "git_clone", intPtr(1))

	// First, clean up existing directory
	cleanupSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for cleanup"
		w.updateDeploymentStep(ctx, deploymentID, 1, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for cleanup: %w", err)
	}
	defer cleanupSession.Close()

	cleanupCmd := "rm -rf /tmp/deployknot-app"
	cleanupOutput, err := cleanupSession.CombinedOutput(cleanupCmd)
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Cleanup warning: %v, output: %s", err, string(cleanupOutput)), "git_cleanup", intPtr(1))
	} else {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Existing directory cleaned up", "git_cleanup", intPtr(1))
	}

	// Create session for cloning
	session, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for cloning"
		w.updateDeploymentStep(ctx, deploymentID, 1, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Normalize repository URL to the expected owner/repo format
	normalized := normalizeRepoURL(repoURL)

	// Prepare git clone command with PAT
	cloneCmd := fmt.Sprintf("git clone https://%s@github.com/%s.git /tmp/deployknot-app", pat, normalized)
	if branch != "main" {
		cloneCmd += fmt.Sprintf(" && cd /tmp/deployknot-app && git checkout %s", branch)
	}

	// Execute command
	output, err := session.CombinedOutput(cloneCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Git clone failed: %v, output: %s", err, string(output))
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "git_clone", intPtr(1))
		w.updateDeploymentStep(ctx, deploymentID, 1, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Repository cloned successfully: %s", string(output)), "git_clone", intPtr(1))

	// Update step status to completed
	if err := w.updateDeploymentStep(ctx, deploymentID, 1, models.DeploymentStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to completed")
	}

	return nil
}

// buildDockerImage builds the Docker image
func (w *Worker) buildDockerImage(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, containerName string) error {
	// Update step status to running
	if err := w.updateDeploymentStep(ctx, deploymentID, 2, models.DeploymentStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to running")
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting Docker build", "docker_build", intPtr(2))

	// Ensure we have a valid container name
	if containerName == "" {
		containerName = fmt.Sprintf("deployknot-%s", deploymentID.String())
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Using generated container name: %s", containerName), "docker_build", intPtr(2))
	}

	// Comprehensive cleanup to ensure fresh deployment
	// Step 1: Force remove existing container
	removeContainerSession, err := sshClient.NewSession()
	if err != nil {
		w.logger.WithError(err).Warn("Failed to create session for container removal")
	} else {
		defer removeContainerSession.Close()
		cleanupCmd := fmt.Sprintf("docker rm -f %s 2>/dev/null || true", containerName)
		cleanupOutput, err := removeContainerSession.CombinedOutput(cleanupCmd)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to remove existing container")
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Remove existing container warning: %v, output: %s", err, string(cleanupOutput)), "docker_rm", intPtr(2))
		} else {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Existing container removed successfully", "docker_rm", intPtr(2))
		}
	}

	// Step 2: Remove container image to force rebuild
	removeImageSession, err := sshClient.NewSession()
	if err != nil {
		w.logger.WithError(err).Warn("Failed to create session for image removal")
	} else {
		defer removeImageSession.Close()
		removeImageCmd := fmt.Sprintf("docker rmi %s:latest 2>/dev/null || true", containerName)
		removeImageOutput, err := removeImageSession.CombinedOutput(removeImageCmd)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to remove existing image")
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Remove existing image warning: %v, output: %s", err, string(removeImageOutput)), "docker_rmi", intPtr(2))
		} else {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Existing image removed successfully", "docker_rmi", intPtr(2))
		}
	}

	// Step 3: Clean up any dangling images and containers
	pruneSession, err := sshClient.NewSession()
	if err != nil {
		w.logger.WithError(err).Warn("Failed to create session for Docker prune")
	} else {
		defer pruneSession.Close()
		pruneCmd := "docker system prune -f"
		pruneOutput, err := pruneSession.CombinedOutput(pruneCmd)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to prune Docker system")
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Docker prune warning: %v, output: %s", err, string(pruneOutput)), "docker_prune", intPtr(2))
		} else {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Docker system cleaned successfully", "docker_prune", intPtr(2))
		}
	}
	time.Sleep(2 * time.Second)

	session, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for Docker build"
		w.updateDeploymentStep(ctx, deploymentID, 2, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Build Docker image with the container name as the image tag
	buildCmd := fmt.Sprintf("cd /tmp/deployknot-app && docker build -t %s:latest .", containerName)
	output, err := session.CombinedOutput(buildCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Docker build failed: %v, output: %s", err, string(output))
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "docker_build", intPtr(2))
		w.updateDeploymentStep(ctx, deploymentID, 2, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("docker build failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker image built successfully: %s", string(output)), "docker_build", intPtr(2))

	// Update step status to completed
	if err := w.updateDeploymentStep(ctx, deploymentID, 2, models.DeploymentStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to completed")
	}

	return nil
}

// runDockerContainer runs the Docker container
func (w *Worker) runDockerContainer(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, envVars string, port int, containerName string) error {
	// Update step status to running
	if err := w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to running")
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting Docker container", "docker_run", intPtr(3))

	// Ensure we have a valid container name
	if containerName == "" {
		containerName = fmt.Sprintf("deployknot-%s", deploymentID.String())
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Using generated container name: %s", containerName), "docker_run", intPtr(3))
	}

	// Stop and remove existing container if running
	stopSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for stop"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for stop: %w", err)
	}
	defer stopSession.Close()

	// More aggressive cleanup - stop, remove, and also remove any containers with the same name
	stopCmd := fmt.Sprintf("docker stop %s 2>/dev/null || true && docker rm %s 2>/dev/null || true && docker ps -a --filter name=%s --format '{{.Names}}' | xargs -r docker rm -f 2>/dev/null || true", containerName, containerName, containerName)
	stopOutput, err := stopSession.CombinedOutput(stopCmd)
	if err != nil {
		w.logger.WithError(err).Warn("Failed to stop existing container")
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Stop existing container warning: %v, output: %s", err, string(stopOutput)), "docker_stop", intPtr(3))
	} else {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Existing container cleanup completed: %s", string(stopOutput)), "docker_stop", intPtr(3))
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)

	// Run new container
	runSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for run"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for run: %w", err)
	}
	defer runSession.Close()

	// First check if Docker is available
	dockerCheckSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for docker check"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for docker check: %w", err)
	}
	defer dockerCheckSession.Close()

	dockerCheckCmd := "docker --version"
	dockerCheckOutput, err := dockerCheckSession.CombinedOutput(dockerCheckCmd)
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Docker not available: %v, output: %s", err, string(dockerCheckOutput)), "docker_check", intPtr(3))
		return fmt.Errorf("docker not available: %w, output: %s", err, string(dockerCheckOutput))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker available: %s", string(dockerCheckOutput)), "docker_check", intPtr(3))

	// Create .env file if environment variables are provided
	envFilePath := ""
	if envVars != "" {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Creating .env file with environment variables", "env_setup", intPtr(3))

		// Create a unique env file path for this deployment
		envFilePath = fmt.Sprintf("/tmp/deployknot-env-%s.env", deploymentID.String())

		envSession, err := sshClient.NewSession()
		if err != nil {
			errorMsg := "Failed to create SSH session for env file"
			w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
			return fmt.Errorf("failed to create SSH session for env file: %w", err)
		}
		defer envSession.Close()

		// Process and validate environment variables
		processedEnvVars := w.processEnvironmentVariables(envVars)

		// Create .env file with proper formatting
		envCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", envFilePath, processedEnvVars)
		envOutput, err := envSession.CombinedOutput(envCmd)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to create .env file: %v, output: %s", err, string(envOutput))
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "env_setup", intPtr(3))
			w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
			return fmt.Errorf("failed to create .env file: %w, output: %s", err, string(envOutput))
		}

		// Verify the .env file was created and has content
		verifySession, err := sshClient.NewSession()
		if err != nil {
			errorMsg := "Failed to create SSH session for env verification"
			w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
			return fmt.Errorf("failed to create SSH session for env verification: %w", err)
		}
		defer verifySession.Close()

		verifyCmd := fmt.Sprintf("ls -la %s && echo '--- ENV FILE CONTENT ---' && cat %s", envFilePath, envFilePath)
		verifyOutput, err := verifySession.CombinedOutput(verifyCmd)
		if err != nil {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Env file verification warning: %v, output: %s", err, string(verifyOutput)), "env_verify", intPtr(3))
		} else {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Environment file created and verified: %s", string(verifyOutput)), "env_verify", intPtr(3))
		}

		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Environment variables file created successfully", "env_setup", intPtr(3))
	}

	// Run container with environment file if available
	var runCmd string
	if envFilePath != "" {
		runCmd = fmt.Sprintf("docker run -d --name %s -p %d:%d --env-file %s %s:latest", containerName, port, port, envFilePath, containerName)
	} else {
		runCmd = fmt.Sprintf("docker run -d --name %s -p %d:%d %s:latest", containerName, port, port, containerName)
	}

	runOutput, err := runSession.CombinedOutput(runCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Docker run failed: %v, output: %s", err, string(runOutput))
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "docker_run", intPtr(3))
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("docker run failed: %w, output: %s", err, string(runOutput))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker container started successfully: %s", string(runOutput)), "docker_run", intPtr(3))

	// Update step status to completed
	if err := w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to completed")
	}

	return nil
}

// processEnvironmentVariables processes and validates environment variables
func (w *Worker) processEnvironmentVariables(envVars string) string {
	// Split by newlines and process each line
	lines := strings.Split(envVars, "\n")
	var processedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // Skip empty lines
		}

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Validate the format (should be KEY=VALUE)
		if !strings.Contains(line, "=") {
			continue // Skip invalid lines
		}

		// Ensure proper formatting
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if they exist
			value = strings.Trim(value, `"'`)

			// Reconstruct the line
			processedLines = append(processedLines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return strings.Join(processedLines, "\n")
}

// healthCheck performs a health check on the deployed application
func (w *Worker) healthCheck(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, containerName string) error {
	// Update step status to running
	if err := w.updateDeploymentStep(ctx, deploymentID, 4, models.DeploymentStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to running")
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting health check", "health_check", intPtr(4))

	// Ensure we have a valid container name
	if containerName == "" {
		containerName = fmt.Sprintf("deployknot-%s", deploymentID.String())
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Using generated container name for health check: %s", containerName), "health_check", intPtr(4))
	}

	session, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for health check"
		w.updateDeploymentStep(ctx, deploymentID, 4, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Check if container is running
	checkCmd := fmt.Sprintf("docker ps --filter name=%s --format 'table {{.Names}}\t{{.Status}}'", containerName)
	output, err := session.CombinedOutput(checkCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Health check failed: %v, output: %s", err, string(output))
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "health_check", intPtr(4))
		w.updateDeploymentStep(ctx, deploymentID, 4, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("health check failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Health check passed: %s", string(output)), "health_check", intPtr(4))

	// Update step status to completed
	if err := w.updateDeploymentStep(ctx, deploymentID, 4, models.DeploymentStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to completed")
	}

	return nil
}

// copyEnvFileToTarget copies the env file from the API server to the target instance via SCP
func (w *Worker) copyEnvFileToTarget(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, localEnvFilePath string) error {
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Copying uploaded .env file to target instance", "env_upload", intPtr(3))
	// Use SCP or SFTP to copy the file
	// For simplicity, use SFTP
	file, err := os.Open(localEnvFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local env file: %w", err)
	}
	defer file.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	remotePath := "/tmp/deployknot-uploaded.env"
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote env file: %w", err)
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, file); err != nil {
		return fmt.Errorf("failed to copy env file to remote: %w", err)
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Uploaded .env file to target instance", "env_upload", intPtr(3))
	return nil
}

// runDockerContainerWithEnvFile runs the Docker container using the uploaded env file
func (w *Worker) runDockerContainerWithEnvFile(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, envFilePath string, port int, containerName string) error {
	// Update step status to running
	if err := w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to running")
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting Docker container with uploaded .env file", "docker_run", intPtr(3))

	if containerName == "" {
		containerName = fmt.Sprintf("deployknot-%s", deploymentID.String())
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Using generated container name: %s", containerName), "docker_run", intPtr(3))
	}

	// Verify the env file exists and has content
	checkEnvSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for env file check"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for env file check: %w", err)
	}
	defer checkEnvSession.Close()

	remoteEnvPath := "/tmp/deployknot-uploaded.env"
	checkEnvCmd := fmt.Sprintf("ls -la %s && echo '---ENV FILE CONTENT---' && cat %s", remoteEnvPath, remoteEnvPath)
	checkEnvOutput, err := checkEnvSession.CombinedOutput(checkEnvCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Env file check failed: %v, output: %s", err, string(checkEnvOutput))
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "env_check", intPtr(3))
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("env file check failed: %w, output: %s", err, string(checkEnvOutput))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Env file verified: %s", string(checkEnvOutput)), "env_check", intPtr(3))

	// Check if the Docker image exists
	checkImageSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for image check"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for image check: %w", err)
	}
	defer checkImageSession.Close()

	checkImageCmd := fmt.Sprintf("docker images %s:latest --format '{{.Repository}}:{{.Tag}}'", containerName)
	checkImageOutput, err := checkImageSession.CombinedOutput(checkImageCmd)
	if err != nil || len(strings.TrimSpace(string(checkImageOutput))) == 0 {
		errorMsg := fmt.Sprintf("Docker image not found: %s:latest", containerName)
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "image_check", intPtr(3))
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("docker image not found: %s:latest", containerName)
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker image found: %s", string(checkImageOutput)), "image_check", intPtr(3))

	// Run new container with --env-file
	runSession, err := sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for run"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for run: %w", err)
	}
	defer runSession.Close()

	// Copy env file to a Docker-accessible location
	copyEnvCmd := fmt.Sprintf("cp %s ./deployknot.env", remoteEnvPath)
	_, err = runSession.CombinedOutput(copyEnvCmd)
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Failed to copy env file: %v", err), "env_copy", intPtr(3))
		errorMsg := fmt.Sprintf("Failed to copy env file: %v", err)
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to copy env file: %w", err)
	}
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Env file copied successfully", "env_copy", intPtr(3))

	// Build the docker run command with the copied env file
	runCmd := fmt.Sprintf("docker run -d --name %s -p %d:%d --env-file ./deployknot.env %s:latest", containerName, port, port, containerName)

	// Log the command being executed
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Executing Docker run command: %s", runCmd), "docker_run", intPtr(3))

	// Execute the actual docker run command with detailed error capture
	runSession, err = sshClient.NewSession()
	if err != nil {
		errorMsg := "Failed to create SSH session for docker run"
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("failed to create SSH session for docker run: %w", err)
	}
	defer runSession.Close()

	runOutput, err := runSession.CombinedOutput(runCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Docker run failed: %v", err)
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", errorMsg, "docker_run", intPtr(3))
		w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusFailed, &errorMsg)
		return fmt.Errorf("docker run failed: %w", err)
	}

	containerID := strings.TrimSpace(string(runOutput))
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker container started successfully with ID: %s", containerID), "docker_run", intPtr(3))

	// Verify the container is running
	verifySession, err := sshClient.NewSession()
	if err == nil {
		checkRunningCmd := fmt.Sprintf("docker ps --filter id=%s --format '{{.Names}} {{.Status}}'", containerID)
		_, err = verifySession.CombinedOutput(checkRunningCmd)
		if err != nil {
			w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", "Container verification failed", "container_check", intPtr(3))
		}
		verifySession.Close()
	}

	// Update step status to completed
	if err := w.updateDeploymentStep(ctx, deploymentID, 3, models.DeploymentStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update step status to completed")
	}

	return nil
}

// markRemainingStepsAsFailed marks all remaining steps as failed when a deployment fails
func (w *Worker) markRemainingStepsAsFailed(ctx context.Context, deploymentID uuid.UUID, failedStepOrder int) {
	// Get all steps for this deployment
	steps, err := w.deploymentService.GetDeploymentSteps(ctx, deploymentID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get deployment steps for marking as failed")
		return
	}

	// Mark all steps after the failed step as failed
	for _, step := range steps {
		if step.StepOrder > failedStepOrder && step.Status == models.DeploymentStatusPending || step.Status == models.DeploymentStatusRunning {
			errorMsg := fmt.Sprintf("Step abandoned due to failure in step %d", failedStepOrder)
			if err := w.updateDeploymentStep(ctx, deploymentID, step.StepOrder, models.DeploymentStatusFailed, &errorMsg); err != nil {
				w.logger.WithError(err).WithField("step_order", step.StepOrder).Error("Failed to mark step as failed")
			}
		}
	}

	w.logger.WithFields(logrus.Fields{
		"deployment_id":     deploymentID,
		"failed_step_order": failedStepOrder,
	}).Info("Marked remaining steps as failed")
}

// markAllStepsAsFailed marks all steps as failed with an error message
func (w *Worker) markAllStepsAsFailed(ctx context.Context, deploymentID uuid.UUID, errorMsg string) {
	steps, err := w.deploymentService.GetDeploymentSteps(ctx, deploymentID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get deployment steps for marking all as failed")
		return
	}
	for _, step := range steps {
		if step.Status != models.DeploymentStatusCompleted && step.Status != models.DeploymentStatusFailed {
			if err := w.updateDeploymentStep(ctx, deploymentID, step.StepOrder, models.DeploymentStatusFailed, &errorMsg); err != nil {
				w.logger.WithError(err).WithField("step_order", step.StepOrder).Error("Failed to mark step as failed (all)")
			}
		}
	}
	w.logger.WithFields(logrus.Fields{"deployment_id": deploymentID}).Info("Marked all steps as failed")
}

// markStepAsFailed with an error message
func (w *Worker) markStepAsFailed(ctx context.Context, stepOrder int, deploymentID uuid.UUID, errorMsg string) error {
	steps, err := w.deploymentService.GetDeploymentSteps(ctx, deploymentID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get deployment steps")
	}
	var targetStep *models.DeploymentStep
	for _, step := range steps {
		if step.StepOrder == stepOrder {
			targetStep = step
			break
		}
	}
	if targetStep == nil {
		w.logger.WithFields(logrus.Fields{
			"deployment_id": deploymentID,
			"step_order":    stepOrder,
		}).Error("Step not found")
		return fmt.Errorf("step not found")
	}

	// Update step status
	now := time.Now()
	targetStep.Status = models.DeploymentStatusFailed
	targetStep.ErrorMessage = &errorMsg
	targetStep.CompletedAt = &now

	if targetStep.StartedAt != nil {
		duration := int(now.Sub(*targetStep.StartedAt).Milliseconds())
		targetStep.DurationMs = &duration
	}

	// Update the step in the database
	if err := w.deploymentService.UpdateDeploymentStep(ctx, targetStep); err != nil {
		w.logger.WithError(err).Error("Failed to update deployment step")
		return err
	}

	w.logger.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"step_name":     targetStep.StepName,
		"step_order":    stepOrder,
		"status":        models.DeploymentStatusFailed,
	}).Info("Deployment step updated")

	return nil
}

// updateDeploymentStep updates a deployment step status
func (w *Worker) updateDeploymentStep(ctx context.Context, deploymentID uuid.UUID, stepOrder int, status models.DeploymentStatus, errorMessage *string) error {
	// Get the step by deployment ID and step order
	steps, err := w.deploymentService.GetDeploymentSteps(ctx, deploymentID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get deployment steps")
		return err
	}

	// Find the step with the matching order
	var targetStep *models.DeploymentStep
	for _, step := range steps {
		if step.StepOrder == stepOrder {
			targetStep = step
			break
		}
	}

	if targetStep == nil {
		w.logger.WithFields(logrus.Fields{
			"deployment_id": deploymentID,
			"step_order":    stepOrder,
		}).Error("Step not found")
		return fmt.Errorf("step not found")
	}

	// Update step status
	now := time.Now()
	targetStep.Status = status
	targetStep.ErrorMessage = errorMessage

	if status == models.DeploymentStatusRunning {
		targetStep.StartedAt = &now
	} else if status == models.DeploymentStatusCompleted || status == models.DeploymentStatusFailed {
		targetStep.CompletedAt = &now
		if targetStep.StartedAt != nil {
			duration := int(now.Sub(*targetStep.StartedAt).Milliseconds())
			targetStep.DurationMs = &duration
		}
	}

	// Update the step in the database
	if err := w.deploymentService.UpdateDeploymentStep(ctx, targetStep); err != nil {
		w.logger.WithError(err).Error("Failed to update deployment step")
		return err
	}

	w.logger.WithFields(logrus.Fields{
		"deployment_id": deploymentID,
		"step_name":     targetStep.StepName,
		"step_order":    stepOrder,
		"status":        status,
	}).Info("Deployment step updated")

	return nil
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// normalizeRepoURL converts various GitHub URL formats to "owner/repo"
func normalizeRepoURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		raw = strings.TrimPrefix(u.Path, "/")
	}
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSuffix(raw, ".git")
	return raw
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper functions for robust extraction from map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case fmt.Stringer:
			return val.String()
		case float64:
			// For numbers that should be strings
			return fmt.Sprintf("%v", val)
		case int:
			return fmt.Sprintf("%d", val)
		case nil:
			return ""
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		case string:
			var i int
			_, err := fmt.Sscanf(val, "%d", &i)
			if err == nil {
				return i
			}
		}
	}
	return 0
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	log := logger.New(cfg.Logging.Level)
	log.Info("Starting DeployKnot worker...")

	// Initialize database
	db, err := database.New(cfg.GetDatabaseURL(), log.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.NewRedis(cfg.GetRedisURL(), log.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redis.Close()

	// Initialize repository
	repo := database.NewRepository(db.DB, log.Logger)

	// Initialize queue service
	queueService := services.NewQueueService(redis.Client, log.Logger)

	// Initialize deployment service
	deploymentService := services.NewDeploymentService(repo, queueService, log.Logger)

	// Initialize worker
	worker := NewWorker(queueService, deploymentService, log.Logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in a goroutine
	go func() {
		if err := worker.Start(ctx); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Shutting down worker...")
	cancel()

	// Give some time for graceful shutdown
	time.Sleep(5 * time.Second)
	log.Info("Worker shutdown complete")
}
