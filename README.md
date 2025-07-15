# DeployKnot

A modern CI/CD automation platform built with Go, featuring a microservices architecture with server and worker components for automated deployment management.

## Features

- **🚀 Automated Deployments**: SSH-based deployment automation
- **🔐 Authentication**: JWT-based user authentication
- **📊 Real-time Logs**: Server-Sent Events for live deployment monitoring
- **🗄️ Database Integration**: PostgreSQL with automatic migrations
- **⚡ Job Queue**: Redis-based job queue for background processing
- **📝 Structured Logging**: JSON-formatted logs with configurable levels
- **🌐 CORS Support**: Cross-origin resource sharing enabled
- **🔄 Graceful Shutdown**: Proper server shutdown handling
- **🐳 Docker Support**: Containerized deployment with Docker
- **📦 Environment Variables**: Advanced environment variable management
- **🔍 Health Monitoring**: Comprehensive health check endpoints

## Architecture

```
┌──────────┐      HTTPS/SSE       ┌──────────────┐     Redis     ┌───────────────┐
│  Browser │◀────────────────────▶│   Server     │◀─────────────▶│   Worker      │
│  UI (FE) │  (REST + Server-Sent  │  (API)       │   Queue       │  (Background) │
└──────────┘     Events for logs) └──────────────┘                └───────────────┘
                                           │                              │
                                           ▼                              ▼
                                     ┌──────────────┐              ┌──────────────┐
                                     │ PostgreSQL   │              │ SSH Target   │
                                     │ (Database)   │              │ (Deploy)     │
                                     └──────────────┘              └──────────────┘
```

## Quick Start

### Prerequisites

- Go 1.24.4+
- PostgreSQL (Docker or local)
- Redis (Docker or local)
- Docker & Docker Compose (optional)

### Local Development

1. **Clone and setup**:
   ```bash
   git clone https://github.com/rohanreddymelachervu/DeployKnot.git
   cd DeployKnot
   cp sample.env .env
   ```

2. **Start services**:
   ```bash
   docker-compose up -d
   ```

3. **Run the application**:
   ```bash
   # Terminal 1: Server
   go run cmd/server/main.go
   
   # Terminal 2: Worker
   go run cmd/worker/main.go
   ```

4. **Test the API**:
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

For detailed setup instructions, see [LOCAL_SETUP.md](LOCAL_SETUP.md).

## API Endpoints

### Health & Status
- `GET /health` - Basic health check
- `GET /api/v1/health` - API health check with detailed status

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `GET /api/v1/auth/profile` - Get user profile (authenticated)

### Deployments
- `GET /api/v1/deployments` - List deployments (authenticated)
- `POST /api/v1/deployments` - Create deployment with environment variables (authenticated, multipart form)
- `GET /api/v1/deployments/:id` - Get deployment details (authenticated)
- `GET /api/v1/deployments/:id/logs` - Stream deployment logs (SSE)
- `GET /api/v1/deployments/:id/steps` - Get deployment steps (authenticated)

### Users
- `GET /api/v1/users/:id/deployments` - Get user's deployments (authenticated)

## Environment Variables

Create a `.env` file with the following variables:

```env
# Server Configuration
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=60s

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=deployknot
DB_SSLMODE=disable
DB_SCHEMA=deploy_knot

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

For detailed environment variable documentation, see [ENVIRONMENT_VARIABLES.md](ENVIRONMENT_VARIABLES.md).

## Environment Variable Management

DeployKnot supports uploading environment variables during deployment. The system will:

1. **Upload**: Accept `.env` files via multipart form upload
2. **Process**: Copy the environment file to the target server
3. **Inject**: Pass environment variables to Docker containers using `--env-file`
4. **Verify**: Ensure environment variables are available in the running container

### Example Usage

```bash
curl -X POST "http://localhost:8080/api/v1/deployments" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "target_ip=YOUR_SERVER_IP" \
  -F "ssh_username=root" \
  -F "ssh_password=YOUR_PASSWORD" \
  -F "github_repo_url=https://github.com/user/repo" \
  -F "github_pat=YOUR_GITHUB_TOKEN" \
  -F "github_branch=main" \
  -F "port=3000" \
  -F "container_name=my-app" \
  -F "project_name=my-project" \
  -F "deployment_name=Production" \
  -F "env_file=@/path/to/your/.env"
```

### Environment File Format

Your `.env` file should contain key-value pairs:

```env
NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@localhost:5432/mydb
API_KEY=your-api-key
DEBUG=false
```

## Example: Deploy with Environment File (Correct Curl)

```
curl --location 'http://localhost:8080/api/v1/deployments' \
--header 'Authorization: Bearer <YOUR_TOKEN>' \
-F target_ip=<IP> \
-F ssh_username=root \
-F ssh_password=password \
-F github_repo_url=https://github.com/user/repo \
-F github_pat=PAT \
-F github_branch=main \
-F port=3000 \
-F container_name=my-app-test \
-F project_name=my-project-test \
-F deployment_name="Production Deployment" \
-F env_file=@/absolute/path/to/sample.env
```

**Troubleshooting:**
- Do NOT quote the `-F` values (e.g. `-F target_ip=...`, not `-F 'target_ip="..."'`).
- Only use quotes if the value contains spaces (e.g. `deployment_name`).
- If the env file is not uploaded, the container will not have your environment variables.

## Project Structure

```
DeployKnot/
├── cmd/
│   ├── server/
│   │   └── main.go          # Server entry point
│   └── worker/
│       └── main.go          # Worker entry point
├── internal/
│   ├── api/
│   │   └── router.go        # API router setup
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── database/
│   │   ├── database.go      # PostgreSQL connection
│   │   ├── redis.go         # Redis connection
│   │   └── repository.go    # Database operations
│   ├── handlers/
│   │   ├── auth.go          # Authentication handlers
│   │   ├── deployment.go    # Deployment handlers
│   │   └── health.go        # Health check handler
│   ├── middleware/
│   │   └── auth.go          # Authentication middleware
│   ├── models/
│   │   ├── deployment.go    # Deployment models
│   │   └── user.go          # User models
│   └── services/
│       ├── deployment.go    # Deployment business logic
│       ├── queue.go         # Job queue service
│       └── user.go          # User service
├── migrations/              # Database migrations
├── pkg/
│   └── logger/
│       └── logger.go        # Logging utilities
├── Dockerfile.server        # Server Dockerfile
├── Dockerfile.worker        # Worker Dockerfile
├── docker-compose.yml       # Local development setup
├── go.mod                   # Go module file
├── go.sum                   # Go dependencies checksum
└── README.md
```

## Deployment

### Docker Deployment

```bash
# Build images
docker build -f Dockerfile.server -t deployknot-server .
docker build -f Dockerfile.worker -t deployknot-worker .

# Run with docker-compose
docker-compose up -d
```

### Production Deployment

For production deployment options, see:
- [Oracle Cloud Free VPS](ORACLE_CLOUD_DEPLOYMENT.md)
- [Render Free Tier](RENDER_DEPLOYMENT.md)
- [Fly.io Free Tier](FLY_IO_DEPLOYMENT.md)

## Development

### Running Tests
```bash
go test ./...
```

### Building
```bash
# Build server
go build -o bin/server cmd/server/main.go

# Build worker
go build -o bin/worker cmd/worker/main.go
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run
```

## API Documentation

### Postman Collection
Import `DeployKnot_API_Collection.json` into Postman for complete API testing.

### Example Requests

#### Create Deployment
```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "target_ip": "192.168.1.100",
    "ssh_username": "root",
    "ssh_password": "password",
    "github_repo_url": "https://github.com/example/repo",
    "github_pat": "your-github-pat",
    "github_branch": "main",
    "port": 3000,
    "container_name": "my-app",
    "project_name": "my-project",
    "deployment_name": "production-deploy",
    "environment_vars": "NODE_ENV=production\nPORT=3000"
  }'
```

#### Stream Deployment Logs
```bash
curl -N http://localhost:8080/api/v1/deployments/DEPLOYMENT_ID/logs
```

## Features in Detail

### 🔐 Authentication System
- JWT-based authentication
- User registration and login
- Protected API endpoints
- User-specific deployments

### 🚀 Deployment Automation
- SSH-based deployment to target servers
- GitHub repository integration
- Docker container deployment
- Environment variable management
- Real-time deployment monitoring

### 📊 Job Queue System
- Redis-based job queue
- Background worker processing
- Job status tracking
- Failed job handling

### 📝 Logging & Monitoring
- Structured JSON logging
- Real-time log streaming via SSE
- Deployment step tracking
- Error handling and reporting

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: Check the `.md` files in this repository
- **API Testing**: Use the Postman collection
- **Issues**: Report bugs and feature requests on GitHub
- **Local Setup**: See [LOCAL_SETUP.md](LOCAL_SETUP.md) for detailed setup instructions 
