# DeployKnot

A CI/CD automation tool for Webknot built with Go + Gin + Ansible.

## Features

- **Health Check Endpoint**: `/health` and `/api/v1/health`
- **Database Integration**: PostgreSQL with migrations
- **Redis Integration**: For job queue functionality
- **Structured Logging**: JSON-formatted logs
- **CORS Support**: Cross-origin resource sharing
- **Graceful Shutdown**: Proper server shutdown handling

## Architecture

```
┌──────────┐      HTTPS/SSE       ┌──────────────┐     MQ/HTTP     ┌───────────────┐
│  Browser │◀────────────────────▶│   Go API     │◀─────────────▶│ Job Queue     │
│  UI (FE) │  (REST + Server-Sent  │  Service     │                │ (Redis)       │
└──────────┘     Events for logs) └──────────────┘                └───────────────┘
                                           │
                                           ▼
                                     ┌──────────────┐
                                     │  Ansible     │
                                     │  Worker      │
                                     └──────────────┘
```

## Prerequisites

- Go 1.21+
- PostgreSQL (Docker or local)
- Redis (Docker or local)
- golang-migrate CLI tool

## Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd DeployKnot
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Set up PostgreSQL**
   ```bash
   # Using Docker
   docker run --name postgres-deployknot \
     -e POSTGRES_PASSWORD=root \
     -e POSTGRES_DB=postgres \
     -p 5432:5432 \
     -d postgres:15
   
   # Create schema
   psql -h localhost -U postgres -d postgres -c "CREATE SCHEMA IF NOT EXISTS deploy_knot;"
   ```

4. **Set up Redis**
   ```bash
   # Using Docker
   docker run --name redis-deployknot \
     -p 6379:6379 \
     -d redis:7-alpine
   ```

5. **Run migrations**
   ```bash
   migrate -path migrations -database "postgres://postgres:root@localhost:5432/postgres?sslmode=disable&search_path=deploy_knot" up
   ```

6. **Create .env file**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

7. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

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
DB_PASSWORD=root
DB_NAME=postgres
DB_SSLMODE=disable
DB_SCHEMA=deploy_knot

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info
```

## API Endpoints

### Health Check
- `GET /health` - Health check endpoint
- `GET /api/v1/health` - Health check endpoint (API versioned)

### Deployment (Coming Soon)
- `GET /api/v1/deployments` - List deployments
- `POST /api/v1/deployments` - Create deployment
- `GET /api/v1/deployments/:id` - Get deployment details
- `GET /api/v1/deployments/:id/logs` - Stream deployment logs (SSE)

## Project Structure

```
DeployKnot/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── api/
│   │   └── router.go        # API router setup
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── database/
│   │   ├── database.go      # PostgreSQL connection
│   │   └── redis.go         # Redis connection
│   ├── handlers/
│   │   └── health.go        # Health check handler
│   └── models/
│       └── deployment.go    # Data models
├── migrations/              # Database migrations
├── pkg/
│   └── logger/
│       └── logger.go        # Logging utilities
└── README.md
```

## Development

### Running Tests
```bash
go test ./...
```

### Building
```bash
go build -o bin/server cmd/server/main.go
```

### Running with Docker
```bash
# Build image
docker build -t deployknot .

# Run container
docker run -p 8080:8080 deployknot
```

## Next Steps

1. **Deployment Handlers**: Implement deployment creation and management
2. **Ansible Integration**: Create Ansible worker for deployment execution
3. **Job Queue**: Implement Redis-based job queue
4. **Frontend**: Create React/Vue frontend for deployment management
5. **Authentication**: Add RBAC and authentication middleware
6. **Monitoring**: Add Prometheus metrics and Grafana dashboards

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License 