package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeDeployment JobType = "deployment"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a job in the queue
type Job struct {
	ID           uuid.UUID              `json:"id"`
	Type         JobType                `json:"type"`
	Status       JobStatus              `json:"status"`
	Data         map[string]interface{} `json:"data"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	DeploymentID uuid.UUID              `json:"deployment_id"`
}

// QueueService handles job queue operations
type QueueService struct {
	redis  *redis.Client
	logger *logrus.Logger
}

// NewQueueService creates a new queue service
func NewQueueService(redis *redis.Client, logger *logrus.Logger) *QueueService {
	return &QueueService{
		redis:  redis,
		logger: logger,
	}
}

// EnqueueDeploymentJob enqueues a deployment job
func (q *QueueService) EnqueueDeploymentJob(ctx context.Context, deploymentID uuid.UUID, deploymentData map[string]interface{}) error {
	job := &Job{
		ID:           uuid.New(),
		Type:         JobTypeDeployment,
		Status:       JobStatusPending,
		Data:         deploymentData,
		CreatedAt:    time.Now(),
		DeploymentID: deploymentID,
	}

	// Serialize job to JSON
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Add to Redis queue
	queueKey := "deployknot:queue:deployments"
	err = q.redis.LPush(ctx, queueKey, jobJSON).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Store job details for tracking
	jobKey := fmt.Sprintf("deployknot:job:%s", job.ID.String())
	err = q.redis.Set(ctx, jobKey, jobJSON, 24*time.Hour).Err()
	if err != nil {
		q.logger.WithError(err).Error("Failed to store job details")
	}

	q.logger.WithFields(logrus.Fields{
		"job_id":        job.ID,
		"deployment_id": deploymentID,
		"type":          job.Type,
	}).Info("Job enqueued successfully")

	return nil
}

// DequeueJob dequeues a job from the queue
func (q *QueueService) DequeueJob(ctx context.Context) (*Job, error) {
	queueKey := "deployknot:queue:deployments"

	// Use BRPOP to block until a job is available
	result, err := q.redis.BRPop(ctx, 30*time.Second, queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue result")
	}

	// Parse job JSON
	var job Job
	err = json.Unmarshal([]byte(result[1]), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now

	// Update job in Redis
	jobJSON, _ := json.Marshal(job)
	jobKey := fmt.Sprintf("deployknot:job:%s", job.ID.String())
	q.redis.Set(ctx, jobKey, jobJSON, 24*time.Hour)

	q.logger.WithFields(logrus.Fields{
		"job_id":        job.ID,
		"deployment_id": job.DeploymentID,
		"type":          job.Type,
	}).Info("Job dequeued and started")

	return &job, nil
}

// UpdateJobStatus updates the status of a job
func (q *QueueService) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status JobStatus, errorMessage *string) error {
	jobKey := fmt.Sprintf("deployknot:job:%s", jobID.String())

	// Get current job
	jobJSON, err := q.redis.Get(ctx, jobKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	err = json.Unmarshal([]byte(jobJSON), &job)
	if err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	job.Status = status
	job.ErrorMessage = errorMessage

	if status == JobStatusCompleted || status == JobStatusFailed {
		now := time.Now()
		job.CompletedAt = &now
	}

	// Save updated job
	updatedJobJSON, _ := json.Marshal(job)
	err = q.redis.Set(ctx, jobKey, updatedJobJSON, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	q.logger.WithFields(logrus.Fields{
		"job_id":        jobID,
		"deployment_id": job.DeploymentID,
		"status":        status,
		"error":         errorMessage,
	}).Info("Job status updated")

	return nil
}

// GetJob retrieves a job by ID
func (q *QueueService) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	jobKey := fmt.Sprintf("deployknot:job:%s", jobID.String())

	jobJSON, err := q.redis.Get(ctx, jobKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	err = json.Unmarshal([]byte(jobJSON), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetQueueLength returns the number of jobs in the queue
func (q *QueueService) GetQueueLength(ctx context.Context) (int64, error) {
	queueKey := "deployknot:queue:deployments"
	length, err := q.redis.LLen(ctx, queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}
