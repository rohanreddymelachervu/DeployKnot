package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CreateDeploymentRequest struct {
	TargetIP        string                 `json:"target_ip"`
	SSHUsername     string                 `json:"ssh_username"`
	SSHPassword     string                 `json:"ssh_password"`
	GitHubRepoURL   string                 `json:"github_repo_url"`
	GitHubPAT       string                 `json:"github_pat"`
	GitHubBranch    string                 `json:"github_branch"`
	EnvironmentVars string                 `json:"environment_vars,omitempty"`
	AdditionalVars  map[string]interface{} `json:"additional_vars,omitempty"`
	Port            int                    `json:"port"`
	ContainerName   *string                `json:"container_name,omitempty"`
	ProjectName     *string                `json:"project_name,omitempty"`
	DeploymentName  *string                `json:"deployment_name,omitempty"`
}

type DeploymentResponse struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	TargetIP       string  `json:"target_ip"`
	GitHubRepoURL  string  `json:"github_repo_url"`
	GitHubBranch   string  `json:"github_branch"`
	Port           int     `json:"port"`
	ContainerName  *string `json:"container_name,omitempty"`
	CreatedAt      string  `json:"created_at"`
	ProjectName    *string `json:"project_name,omitempty"`
	DeploymentName *string `json:"deployment_name,omitempty"`
}

func main() {
	// Test credentials (replace with your actual values)
	targetIP := "172.235.15.164"
	sshUsername := "root"
	sshPassword := "your_ssh_password"
	githubRepoURL := "https://github.com/rohanreddymelachervu/DeployKnot-test"
	githubPAT := "your_github_pat"

	fmt.Println("🚀 Testing Multi-Container Deployment Support")
	fmt.Println("=============================================")

	// Test 1: Deploy with custom container name and port
	fmt.Println("\n📦 Test 1: Deploy with custom container name and port")

	projectName := "test-project"
	deploymentName := "api-service"
	containerName := "my-api-service"
	port := 3001

	req1 := CreateDeploymentRequest{
		TargetIP:        targetIP,
		SSHUsername:     sshUsername,
		SSHPassword:     sshPassword,
		GitHubRepoURL:   githubRepoURL,
		GitHubPAT:       githubPAT,
		GitHubBranch:    "main",
		EnvironmentVars: "NODE_ENV=production\nAPI_KEY=test123",
		Port:            port,
		ContainerName:   &containerName,
		ProjectName:     &projectName,
		DeploymentName:  &deploymentName,
	}

	deployment1 := createDeployment(req1)
	if deployment1 != nil {
		fmt.Printf("✅ Deployment 1 created: %s\n", deployment1.ID)
		fmt.Printf("   Container name: %s\n", *deployment1.ContainerName)
		fmt.Printf("   Port: %d\n", deployment1.Port)
	}

	// Wait a bit before next deployment
	time.Sleep(2 * time.Second)

	// Test 2: Deploy another service with different container name and port
	fmt.Println("\n📦 Test 2: Deploy another service with different container name and port")

	projectName2 := "test-project"
	deploymentName2 := "web-service"
	containerName2 := "my-web-service"
	port2 := 3002

	req2 := CreateDeploymentRequest{
		TargetIP:        targetIP,
		SSHUsername:     sshUsername,
		SSHPassword:     sshPassword,
		GitHubRepoURL:   githubRepoURL,
		GitHubPAT:       githubPAT,
		GitHubBranch:    "main",
		EnvironmentVars: "NODE_ENV=production\nWEB_PORT=3002",
		Port:            port2,
		ContainerName:   &containerName2,
		ProjectName:     &projectName2,
		DeploymentName:  &deploymentName2,
	}

	deployment2 := createDeployment(req2)
	if deployment2 != nil {
		fmt.Printf("✅ Deployment 2 created: %s\n", deployment2.ID)
		fmt.Printf("   Container name: %s\n", *deployment2.ContainerName)
		fmt.Printf("   Port: %d\n", deployment2.Port)
	}

	// Wait a bit before next deployment
	time.Sleep(2 * time.Second)

	// Test 3: Deploy with auto-generated container name
	fmt.Println("\n📦 Test 3: Deploy with auto-generated container name")

	projectName3 := "auto-test"
	deploymentName3 := "backend-api"
	port3 := 3003

	req3 := CreateDeploymentRequest{
		TargetIP:        targetIP,
		SSHUsername:     sshUsername,
		SSHPassword:     sshPassword,
		GitHubRepoURL:   githubRepoURL,
		GitHubPAT:       githubPAT,
		GitHubBranch:    "main",
		EnvironmentVars: "NODE_ENV=production\nDB_HOST=localhost",
		Port:            port3,
		ProjectName:     &projectName3,
		DeploymentName:  &deploymentName3,
		// ContainerName is nil - will be auto-generated
	}

	deployment3 := createDeployment(req3)
	if deployment3 != nil {
		fmt.Printf("✅ Deployment 3 created: %s\n", deployment3.ID)
		fmt.Printf("   Auto-generated container name: %s\n", *deployment3.ContainerName)
		fmt.Printf("   Port: %d\n", deployment3.Port)
	}

	// Wait a bit before next deployment
	time.Sleep(2 * time.Second)

	// Test 4: Deploy with fallback to deployment ID
	fmt.Println("\n📦 Test 4: Deploy with fallback to deployment ID")

	deploymentName4 := "simple-app"
	port4 := 3004

	req4 := CreateDeploymentRequest{
		TargetIP:        targetIP,
		SSHUsername:     sshUsername,
		SSHPassword:     sshPassword,
		GitHubRepoURL:   githubRepoURL,
		GitHubPAT:       githubPAT,
		GitHubBranch:    "main",
		EnvironmentVars: "NODE_ENV=production\nSIMPLE_APP=true",
		Port:            port4,
		DeploymentName:  &deploymentName4,
		// No ProjectName or ContainerName - will use deployment ID
	}

	deployment4 := createDeployment(req4)
	if deployment4 != nil {
		fmt.Printf("✅ Deployment 4 created: %s\n", deployment4.ID)
		fmt.Printf("   Fallback container name: %s\n", *deployment4.ContainerName)
		fmt.Printf("   Port: %d\n", deployment4.Port)
	}

	fmt.Println("\n🎉 All deployments completed!")
	fmt.Println("=============================================")
	fmt.Println("You can now run multiple containers on the same host:")
	if deployment1 != nil {
		fmt.Printf("   📦 Container 1: %s on port %d\n", *deployment1.ContainerName, deployment1.Port)
	}
	if deployment2 != nil {
		fmt.Printf("   📦 Container 2: %s on port %d\n", *deployment2.ContainerName, deployment2.Port)
	}
	if deployment3 != nil {
		fmt.Printf("   📦 Container 3: %s on port %d\n", *deployment3.ContainerName, deployment3.Port)
	}
	if deployment4 != nil {
		fmt.Printf("   📦 Container 4: %s on port %d\n", *deployment4.ContainerName, deployment4.Port)
	}

	fmt.Println("\n🔧 Docker commands that would be executed:")
	fmt.Println("   docker run -d --name my-api-service -p 3001:3001 my-api-service:latest")
	fmt.Println("   docker run -d --name my-web-service -p 3002:3002 my-web-service:latest")
	fmt.Println("   docker run -d --name deployknot-auto-test-backend-api -p 3003:3003 deployknot-auto-test-backend-api:latest")
	fmt.Println("   docker run -d --name deployknot-[deployment-id] -p 3004:3004 deployknot-[deployment-id]:latest")

	fmt.Println("\n✨ Key Benefits:")
	fmt.Println("   ✅ Multiple containers on same host")
	fmt.Println("   ✅ Independent port mapping")
	fmt.Println("   ✅ Unique container names")
	fmt.Println("   ✅ No conflicts between deployments")
	fmt.Println("   ✅ Selective container management")
}

func createDeployment(req CreateDeploymentRequest) *DeploymentResponse {
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("❌ Error marshaling request: %v\n", err)
		return nil
	}

	resp, err := http.Post("http://localhost:8080/api/v1/deployments", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error creating deployment: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return nil
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("❌ Error: %s - %s\n", resp.Status, string(body))
		return nil
	}

	var deployment DeploymentResponse
	if err := json.Unmarshal(body, &deployment); err != nil {
		fmt.Printf("❌ Error unmarshaling response: %v\n", err)
		return nil
	}

	return &deployment
}
