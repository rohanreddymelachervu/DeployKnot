package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deployknot/internal/config"
	"deployknot/internal/database"
	"deployknot/internal/models"
	"deployknot/internal/services"
	"deployknot/pkg/logger"

	"github.com/google/uuid"
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

	// Extract deployment data
	targetIP, _ := job.Data["target_ip"].(string)
	sshUsername, _ := job.Data["ssh_username"].(string)
	sshPassword, _ := job.Data["ssh_password"].(string)
	githubRepoURL, _ := job.Data["github_repo_url"].(string)
	githubPAT, _ := job.Data["github_pat"].(string)
	githubBranch, _ := job.Data["github_branch"].(string)

	// Debug logging for credentials (masked)
	w.logger.WithFields(logrus.Fields{
		"target_ip":           targetIP,
		"ssh_username":        sshUsername,
		"ssh_password_length": len(sshPassword),
		"github_repo_url":     githubRepoURL,
		"github_pat_length":   len(githubPAT),
		"github_branch":       githubBranch,
	}).Info("Extracted deployment credentials")

	// Validate required fields
	if targetIP == "" || sshUsername == "" || sshPassword == "" || githubRepoURL == "" || githubPAT == "" || githubBranch == "" {
		return fmt.Errorf("missing required deployment parameters")
	}

	// Connect to target server via SSH
	sshClient, err := w.connectSSH(targetIP, sshUsername, sshPassword)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to connect to target server: %v", err)
		w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "error", errorMsg, "ssh_connect", nil)
		return fmt.Errorf("failed to connect to target server: %w", err)
	}
	defer sshClient.Close()

	w.deploymentService.AddDeploymentLog(ctx, job.DeploymentID, "info", "SSH connection established", "ssh_connect", nil)

	// Execute deployment steps
	if err := w.executeDeploymentSteps(ctx, job.DeploymentID, sshClient, githubRepoURL, githubPAT, githubBranch); err != nil {
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
func (w *Worker) executeDeploymentSteps(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, repoURL, pat, branch string) error {
	// Step 1: Clone the repository
	if err := w.cloneRepository(ctx, deploymentID, sshClient, repoURL, pat, branch); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Step 2: Build Docker image
	if err := w.buildDockerImage(ctx, deploymentID, sshClient); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Step 3: Run Docker container
	if err := w.runDockerContainer(ctx, deploymentID, sshClient); err != nil {
		return fmt.Errorf("failed to run Docker container: %w", err)
	}

	// Step 4: Health check
	if err := w.healthCheck(ctx, deploymentID, sshClient); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// cloneRepository clones the Git repository
func (w *Worker) cloneRepository(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client, repoURL, pat, branch string) error {
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting repository clone", "git_clone", intPtr(1))

	// Create session
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Prepare git clone command with PAT
	cloneCmd := fmt.Sprintf("git clone https://%s@github.com/%s.git /tmp/deployknot-app", pat, repoURL)
	if branch != "main" {
		cloneCmd += fmt.Sprintf(" && cd /tmp/deployknot-app && git checkout %s", branch)
	}

	// Execute command
	output, err := session.CombinedOutput(cloneCmd)
	session.Close() // Close immediately after use
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Git clone failed: %v, output: %s", err, string(output)), "git_clone", intPtr(1))
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Repository cloned successfully: %s", string(output)), "git_clone", intPtr(1))
	return nil
}

// buildDockerImage builds the Docker image
func (w *Worker) buildDockerImage(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client) error {
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting Docker build", "docker_build", intPtr(2))

	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Build Docker image
	buildCmd := "cd /tmp/deployknot-app && docker build -t deployknot-app:latest ."
	output, err := session.CombinedOutput(buildCmd)
	session.Close() // Close immediately after use
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Docker build failed: %v, output: %s", err, string(output)), "docker_build", intPtr(2))
		return fmt.Errorf("docker build failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker image built successfully: %s", string(output)), "docker_build", intPtr(2))
	return nil
}

// runDockerContainer runs the Docker container
func (w *Worker) runDockerContainer(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client) error {
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting Docker container", "docker_run", intPtr(3))

	// Stop existing container if running
	stopSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for stop: %w", err)
	}

	stopCmd := "docker stop deployknot-app || true && docker rm deployknot-app || true"
	stopOutput, err := stopSession.CombinedOutput(stopCmd)
	stopSession.Close() // Close immediately after use
	if err != nil {
		w.logger.WithError(err).Warn("Failed to stop existing container")
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "warn", fmt.Sprintf("Stop existing container warning: %v, output: %s", err, string(stopOutput)), "docker_stop", intPtr(3))
	} else {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Existing container stopped: %s", string(stopOutput)), "docker_stop", intPtr(3))
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)

	// Run new container
	runSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for run: %w", err)
	}

	// First check if Docker is available
	dockerCheckSession, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session for docker check: %w", err)
	}

	dockerCheckCmd := "docker --version"
	dockerCheckOutput, err := dockerCheckSession.CombinedOutput(dockerCheckCmd)
	dockerCheckSession.Close()
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Docker not available: %v, output: %s", err, string(dockerCheckOutput)), "docker_check", intPtr(3))
		return fmt.Errorf("docker not available: %w, output: %s", err, string(dockerCheckOutput))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker available: %s", string(dockerCheckOutput)), "docker_check", intPtr(3))

	runCmd := "docker run -d --name deployknot-app -p 3000:3000 deployknot-app:latest"
	runOutput, err := runSession.CombinedOutput(runCmd)
	runSession.Close() // Close immediately after use

	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Docker run failed: %v, output: %s", err, string(runOutput)), "docker_run", intPtr(3))
		return fmt.Errorf("docker run failed: %w, output: %s", err, string(runOutput))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Docker container started successfully: %s", string(runOutput)), "docker_run", intPtr(3))
	return nil
}

// healthCheck performs a health check on the deployed application
func (w *Worker) healthCheck(ctx context.Context, deploymentID uuid.UUID, sshClient *ssh.Client) error {
	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", "Starting health check", "health_check", intPtr(4))

	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Check if container is running
	checkCmd := "docker ps --filter name=deployknot-app --format 'table {{.Names}}\t{{.Status}}'"
	output, err := session.CombinedOutput(checkCmd)
	session.Close() // Close immediately after use
	if err != nil {
		w.deploymentService.AddDeploymentLog(ctx, deploymentID, "error", fmt.Sprintf("Health check failed: %v, output: %s", err, string(output)), "health_check", intPtr(4))
		return fmt.Errorf("health check failed: %w, output: %s", err, string(output))
	}

	w.deploymentService.AddDeploymentLog(ctx, deploymentID, "info", fmt.Sprintf("Health check passed: %s", string(output)), "health_check", intPtr(4))
	return nil
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
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
