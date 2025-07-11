package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"deployknot/internal/models"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://postgres:root@localhost:5432/postgres?sslmode=disable&search_path=deploy_knot")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test deployment
	deployment := &models.Deployment{
		ID:                   uuid.New(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		Status:               models.DeploymentStatusPending,
		TargetIP:             "192.168.1.100",
		SSHUsername:          "root",
		SSHPasswordEncrypted: stringPtr("password123"),
		GitHubRepoURL:        "https://github.com/example/repo",
		GitHubPATEncrypted:   stringPtr("ghp_example"),
		GitHubBranch:         "main",
		AdditionalVars:       map[string]interface{}{"test": "value"},
		ProjectName:          stringPtr("test-project"),
		DeploymentName:       stringPtr("test-deployment"),
	}

	// Convert AdditionalVars to JSON bytes
	var additionalVarsJSON []byte
	if deployment.AdditionalVars != nil {
		additionalVarsJSON, err = json.Marshal(deployment.AdditionalVars)
		if err != nil {
			log.Fatal("Failed to marshal additional_vars:", err)
		}
	} else {
		additionalVarsJSON = []byte("null")
	}

	fmt.Printf("AdditionalVars type: %T\n", additionalVarsJSON)
	fmt.Printf("AdditionalVars value: %s\n", string(additionalVarsJSON))

	// Insert deployment
	query := `
		INSERT INTO deploy_knot.deployments (
			id, created_at, updated_at, status, target_ip, ssh_username, 
			ssh_password_encrypted, github_repo_url, github_pat_encrypted, 
			github_branch, additional_vars, created_by, project_name, deployment_name
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	_, err = db.Exec(query,
		deployment.ID,
		deployment.CreatedAt,
		deployment.UpdatedAt,
		deployment.Status,
		deployment.TargetIP,
		deployment.SSHUsername,
		deployment.SSHPasswordEncrypted,
		deployment.GitHubRepoURL,
		deployment.GitHubPATEncrypted,
		deployment.GitHubBranch,
		additionalVarsJSON,
		deployment.CreatedBy,
		deployment.ProjectName,
		deployment.DeploymentName,
	)

	if err != nil {
		log.Fatal("Failed to create deployment:", err)
	}

	fmt.Println("Deployment created successfully!")
}

func stringPtr(s string) *string {
	return &s
}
