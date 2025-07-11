# DeployKnot API Postman Collection

This Postman collection provides a complete set of requests for testing and interacting with the DeployKnot API. The collection includes all available endpoints including health checks, deployment management, and Server-Sent Events (SSE) for real-time log streaming.

## 📋 Collection Overview

### Health Check Endpoints
- **Health Check - Root**: `GET /health`
- **Health Check - API v1**: `GET /api/v1/health`

### Deployment Management Endpoints
- **List Deployments**: `GET /api/v1/deployments`
- **Create Deployment**: `POST /api/v1/deployments`
- **Get Deployment**: `GET /api/v1/deployments/{id}`
- **Get Deployment Logs (JSON)**: `GET /api/v1/deployments/{id}/logs`
- **Get Deployment Logs (SSE)**: `GET /api/v1/deployments/{id}/logs` (with SSE headers)
- **Get Deployment Steps**: `GET /api/v1/deployments/{id}/steps`

## 🚀 Getting Started

### 1. Import the Collection
1. Open Postman
2. Click "Import" button
3. Select the `DeployKnot_API_Collection.json` file
4. The collection will be imported with all endpoints

### 2. Configure Environment Variables
The collection uses the following variables that you should configure:

#### Collection Variables
- `base_url`: The base URL of your DeployKnot API server (default: `http://localhost:8080`)
- `deployment_id`: A deployment UUID to use in requests (set this after creating a deployment)

#### How to Set Variables
1. Click on the collection name in Postman
2. Go to the "Variables" tab
3. Update the `base_url` to match your server
4. Set `deployment_id` after creating your first deployment

## 📝 Request Examples

### Health Check
```bash
GET http://localhost:8080/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-07-11T22:30:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy"
  }
}
```

### Create Deployment

#### Full Deployment with Custom Container Name
```bash
POST http://localhost:8080/api/v1/deployments
Content-Type: application/json

{
  "target_ip": "172.235.15.164",
  "ssh_username": "root",
  "ssh_password": "your_ssh_password",
  "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
  "github_pat": "your_github_pat",
  "github_branch": "main",
  "environment_vars": "NODE_ENV=production\nPORT=3000",
  "port": 3000,
  "container_name": "my-app",
  "project_name": "My Project",
  "deployment_name": "Production Deployment"
}
```

#### Auto-Generated Container Name
```bash
POST http://localhost:8080/api/v1/deployments
Content-Type: application/json

{
  "target_ip": "172.235.15.164",
  "ssh_username": "root",
  "ssh_password": "your_ssh_password",
  "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
  "github_pat": "your_github_pat",
  "github_branch": "main",
  "environment_vars": "NODE_ENV=production\nPORT=3000",
  "port": 3001,
  "project_name": "My Project",
  "deployment_name": "Production Deployment"
}
```

#### Minimal Deployment
```bash
POST http://localhost:8080/api/v1/deployments
Content-Type: application/json

{
  "target_ip": "172.235.15.164",
  "ssh_username": "root",
  "ssh_password": "your_ssh_password",
  "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
  "github_pat": "your_github_pat",
  "github_branch": "main",
  "port": 3002
}
```

**Expected Response:**
```json
{
  "id": "67cfd2e8-0917-45b7-8747-482730c62ef1",
  "status": "pending",
  "target_ip": "172.235.15.164",
  "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
  "github_branch": "main",
  "port": 3000,
  "container_name": "my-app",
  "created_at": "2025-07-11T22:04:58.632+05:30",
  "project_name": "My Project",
  "deployment_name": "Production Deployment"
}
```

### Get Deployment Details
```bash
GET http://localhost:8080/api/v1/deployments/67cfd2e8-0917-45b7-8747-482730c62ef1
```

### Get Deployment Logs (JSON)
```bash
GET http://localhost:8080/api/v1/deployments/67cfd2e8-0917-45b7-8747-482730c62ef1/logs?limit=100
```

**Expected Response:**
```json
{
  "deployment_id": "67cfd2e8-0917-45b7-8747-482730c62ef1",
  "logs": [
    {
      "id": "log-uuid-here",
      "deployment_id": "67cfd2e8-0917-45b7-8747-482730c62ef1",
      "created_at": "2025-07-11T22:04:58.632+05:30",
      "log_level": "info",
      "message": "Deployment created successfully",
      "task_name": "create_deployment",
      "step_order": 1
    }
  ]
}
```

### Get Deployment Steps
```bash
GET http://localhost:8080/api/v1/deployments/67cfd2e8-0917-45b7-8747-482730c62ef1/steps
```

**Expected Response:**
```json
{
  "deployment_id": "67cfd2e8-0917-45b7-8747-482730c62ef1",
  "steps": [
    {
      "id": "step-uuid-here",
      "deployment_id": "67cfd2e8-0917-45b7-8747-482730c62ef1",
      "step_name": "SSH Connection",
      "status": "completed",
      "started_at": "2025-07-11T22:08:31.106+05:30",
      "completed_at": "2025-07-11T22:08:31.701+05:30",
      "duration_ms": 595,
      "step_order": 1
    }
  ]
}
```

## 🔄 Server-Sent Events (SSE)

The SSE endpoint provides real-time log streaming during deployment execution. This is particularly useful for monitoring deployment progress in real-time.

### SSE Request
```bash
GET http://localhost:8080/api/v1/deployments/67cfd2e8-0917-45b7-8747-482730c62ef1/logs
Accept: text/event-stream
Cache-Control: no-cache
```

### SSE Response Format
The SSE endpoint returns events in the following format:

```
event: connected
data: {"deployment_id":"67cfd2e8-0917-45b7-8747-482730c62ef1","timestamp":"2025-07-11T22:08:31.106+05:30"}

event: log
data: {"id":"log-uuid","deployment_id":"67cfd2e8-0917-45b7-8747-482730c62ef1","created_at":"2025-07-11T22:08:31.106+05:30","log_level":"info","message":"Processing deployment job","task_name":"deployment","step_order":1}

event: log
data: {"id":"log-uuid-2","deployment_id":"67cfd2e8-0917-45b7-8747-482730c62ef1","created_at":"2025-07-11T22:08:31.701+05:30","log_level":"info","message":"SSH connection established successfully","task_name":"ssh_connection","step_order":2}
```

### Testing SSE in Postman
1. Use the "Get Deployment Logs - SSE" request
2. Send the request
3. Postman will display the streaming events in the response body
4. The connection will remain open until the deployment completes or you close the request

## 🔧 Field Descriptions

### Create Deployment Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target_ip` | string | Yes | IP address of the target server |
| `ssh_username` | string | Yes | SSH username for target server |
| `ssh_password` | string | Yes | SSH password for target server |
| `github_repo_url` | string | Yes | GitHub repository URL |
| `github_pat` | string | Yes | GitHub Personal Access Token |
| `github_branch` | string | Yes | GitHub branch to deploy |
| `environment_vars` | string | No | Environment variables (newline-separated) |
| `additional_vars` | object | No | Additional variables as key-value pairs |
| `port` | integer | Yes | Port number (1-65535) |
| `container_name` | string | No | Custom container name (auto-generated if not provided) |
| `project_name` | string | No | Project name for organization |
| `deployment_name` | string | No | Deployment name for identification |

### Deployment Status Values

- `pending`: Deployment is queued and waiting to start
- `running`: Deployment is currently executing
- `completed`: Deployment completed successfully
- `failed`: Deployment failed with an error
- `cancelled`: Deployment was cancelled

## 🧪 Testing Workflow

### 1. Health Check
Start by testing the health endpoint to ensure the server is running:
```
GET /health
```

### 2. Create Deployment
Create a new deployment using one of the provided examples:
```
POST /api/v1/deployments
```

### 3. Monitor Deployment
Use the deployment ID from the create response to monitor progress:

#### Get Deployment Status
```
GET /api/v1/deployments/{deployment_id}
```

#### Stream Real-time Logs
```
GET /api/v1/deployments/{deployment_id}/logs (with SSE headers)
```

#### Get Deployment Steps
```
GET /api/v1/deployments/{deployment_id}/steps
```

### 4. List All Deployments
```
GET /api/v1/deployments
```

## 🔒 Security Notes

- **SSH Credentials**: Never commit real SSH passwords to version control
- **GitHub PAT**: Use GitHub Personal Access Tokens with minimal required permissions
- **Environment Variables**: Sensitive data in environment variables should be encrypted
- **Network Security**: Ensure your API server is properly secured in production

## 🐛 Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure the DeployKnot server is running on the correct port
2. **Invalid UUID**: Make sure deployment IDs are valid UUIDs
3. **SSE Not Working**: Ensure you're using the correct headers (`Accept: text/event-stream`)
4. **Authentication Errors**: Verify SSH credentials and GitHub PAT are correct

### Error Responses

```json
{
  "error": "Validation failed",
  "message": "Field 'target_ip' is required"
}
```

```json
{
  "error": "Deployment not found",
  "message": "The specified deployment does not exist"
}
```

## 📚 Additional Resources

- [DeployKnot Documentation](README.md)
- [API Source Code](internal/api/)
- [Deployment Service](internal/services/deployment.go)
- [Database Models](internal/models/deployment.go)

## 🤝 Contributing

To add new endpoints or improve the collection:

1. Update the `DeployKnot_API_Collection.json` file
2. Add corresponding examples to this README
3. Test all endpoints to ensure they work correctly
4. Update the collection description and documentation

---

**Note**: This collection is designed for DeployKnot API v1. When new API versions are released, create separate collections for each version. 