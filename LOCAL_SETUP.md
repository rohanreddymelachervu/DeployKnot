# DeployKnot Local Setup Guide

This guide will help you set up DeployKnot locally for development and testing.

## Prerequisites

### Required Software

1. **Go 1.24.4+**
   ```bash
   # Check Go version
   go version
   
   # Install Go (if not installed)
   # macOS: brew install go
   # Ubuntu: sudo apt install golang-go
   # Windows: Download from https://golang.org/dl/
   ```

2. **Docker & Docker Compose**
   ```bash
   # Check Docker version
   docker --version
   docker-compose --version
   
   # Install Docker (if not installed)
   # macOS: brew install docker
   # Ubuntu: sudo apt install docker.io docker-compose
   # Windows: Download Docker Desktop
   ```

3. **Git**
   ```bash
   # Check Git version
   git --version
   
   # Install Git (if not installed)
   # macOS: brew install git
   # Ubuntu: sudo apt install git
   # Windows: Download from https://git-scm.com/
   ```

4. **PostgreSQL Client (Optional)**
   ```bash
   # macOS
   brew install postgresql
   
   # Ubuntu
   sudo apt install postgresql-client
   
   # Windows
   # Download from https://www.postgresql.org/download/windows/
   ```

### Optional Tools

5. **golang-migrate CLI** (for manual migrations)
   ```bash
   # macOS
   brew install golang-migrate
   
   # Ubuntu
   sudo apt install golang-migrate
   
   # Or install via Go
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

## Step 1: Clone Repository

```bash
# Clone the repository
git clone https://github.com/rohanreddymelachervu/DeployKnot.git
cd DeployKnot

# Verify the structure
ls -la
```

## Step 2: Install Dependencies

```bash
# Install Go dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Step 3: Set Up Environment Variables

```bash
# Copy sample environment file
cp sample.env .env

# Edit the environment file
# macOS/Linux
nano .env
# or
code .env

# Windows
notepad .env
```

### Environment Variables Configuration

Edit `.env` with your configuration:

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

# JWT Configuration (for authentication)
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

### Required Environment Variables for Deployment

When deploying DeployKnot to production, you'll need these environment variables:

#### **Server Environment Variables**
```env
# Application Configuration
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_IDLE_TIMEOUT=60s

# Database Configuration
DB_HOST=your-database-host
DB_PORT=5432
DB_USER=your-database-user
DB_PASSWORD=your-database-password
DB_NAME=deployknot
DB_SSLMODE=disable
DB_SCHEMA=deploy_knot

# Redis Configuration
REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this
```

#### **Worker Environment Variables**
```env
# Database Configuration (same as server)
DB_HOST=your-database-host
DB_PORT=5432
DB_USER=your-database-user
DB_PASSWORD=your-database-password
DB_NAME=deployknot
DB_SSLMODE=disable
DB_SCHEMA=deploy_knot

# Redis Configuration (same as server)
REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0

# Logging Configuration
LOG_LEVEL=info
```

#### **Environment Variables Checklist**

- [ ] `SERVER_PORT` - HTTP server port (default: 8080)
- [ ] `DB_HOST` - PostgreSQL host address
- [ ] `DB_PORT` - PostgreSQL port (default: 5432)
- [ ] `DB_USER` - PostgreSQL username
- [ ] `DB_PASSWORD` - PostgreSQL password
- [ ] `DB_NAME` - Database name (default: deployknot)
- [ ] `DB_SCHEMA` - Database schema (must be: deploy_knot)
- [ ] `REDIS_HOST` - Redis host address
- [ ] `REDIS_PORT` - Redis port (default: 6379)
- [ ] `REDIS_PASSWORD` - Redis password (if required)
- [ ] `REDIS_DB` - Redis database number (default: 0)
- [ ] `LOG_LEVEL` - Logging level (debug, info, warn, error)
- [ ] `JWT_SECRET` - Secret key for JWT tokens (server only)

#### **Environment-Specific Examples**

**Development:**
```env
DB_HOST=localhost
REDIS_HOST=localhost
JWT_SECRET=dev-secret-key
```

**Production:**
```env
DB_HOST=your-production-db.region.rds.amazonaws.com
REDIS_HOST=your-production-redis.region.cache.amazonaws.com
JWT_SECRET=your-super-secure-production-jwt-secret
```

**Docker Compose:**
```env
DB_HOST=postgres
REDIS_HOST=redis
JWT_SECRET=your-jwt-secret
```

## Step 4: Set Up Database and Redis

### Option A: Using Docker (Recommended)

```bash
# Start PostgreSQL
docker run --name postgres-deployknot \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=deployknot \
  -e POSTGRES_USER=postgres \
  -p 5432:5432 \
  -d postgres:15-alpine

# Create schema (wait a moment for PostgreSQL to start)
sleep 5
docker exec postgres-deployknot psql -U postgres -d deployknot -c "CREATE SCHEMA IF NOT EXISTS deploy_knot;"

# Start Redis
docker run --name redis-deployknot \
  -p 6379:6379 \
  -d redis:7-alpine

# Verify containers are running
docker ps
```

### Option B: Using Docker Compose

```bash
# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

### Option C: Local Installation

#### PostgreSQL (Ubuntu)
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database
sudo -u postgres psql
CREATE DATABASE deployknot;
CREATE USER deployknot WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE deployknot TO deployknot;
CREATE SCHEMA IF NOT EXISTS deploy_knot;
\q
```

#### Redis (Ubuntu)
```bash
sudo apt install redis-server

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

## Step 5: Run Database Migrations

### Option A: Using Docker (if using Docker for database)

```bash
# Run migrations
migrate -path migrations -database "postgres://postgres:password@localhost:5432/deployknot?sslmode=disable&search_path=deploy_knot" up
```

### Option B: Using Docker Compose

```bash
# Run migrations
docker-compose exec server go run cmd/server/main.go
# The server will automatically run migrations on startup
```

### Option C: Manual Migration

```bash
# Create schema first
psql -h localhost -U postgres -d deployknot -c "CREATE SCHEMA IF NOT EXISTS deploy_knot;"

# Run migrations
migrate -path migrations -database "postgres://postgres:password@localhost:5432/deployknot?sslmode=disable&search_path=deploy_knot" up
```

## Step 6: Build and Run the Application

### Option A: Run Server Only

```bash
# Run the server
go run cmd/server/main.go

# Or build and run
go build -o bin/server cmd/server/main.go
./bin/server
```

### Option B: Run Both Server and Worker

```bash
# Terminal 1: Run server
go run cmd/server/main.go

# Terminal 2: Run worker
go run cmd/worker/main.go
```

### Option C: Using Docker Compose

```bash
# Build and run all services
docker-compose up --build

# Or run in background
docker-compose up -d --build
```

## Step 7: Verify Installation

### Check Server Health

```bash
# Health check
curl http://localhost:8080/health

# API health check
curl http://localhost:8080/api/v1/health

# Expected response:
# {"status":"healthy","timestamp":"2024-01-01T12:00:00Z"}
```

### Check Database Connection

```bash
# Connect to PostgreSQL
psql -h localhost -U postgres -d deployknot

# Check tables
\dt deploy_knot.*

# Exit
\q
```

### Check Redis Connection

```bash
# Connect to Redis
redis-cli -h localhost -p 6379

# Test connection
ping

# Exit
exit
```

## Step 8: Test the API

### Using curl

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Create a deployment (example)
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "target_ip": "192.168.1.100",
    "ssh_username": "root",
    "ssh_password": "password",
    "github_repo_url": "https://github.com/example/repo",
    "github_pat": "your-github-pat",
    "github_branch": "main",
    "port": 3000,
    "container_name": "test-app",
    "project_name": "test-project",
    "deployment_name": "test-deployment"
  }'
```

### Using Postman

1. **Import the Postman Collection**:
   - Open Postman
   - Import `DeployKnot_API_Collection.json`
   - Set the base URL to `http://localhost:8080`

2. **Test the endpoints**:
   - Health Check: `GET /api/v1/health`
   - Create Deployment: `POST /api/v1/deployments`
   - Get Deployments: `GET /api/v1/deployments`

### Testing Environment Variable Uploads

1. **Create a test environment file**:
   ```bash
   cat > test.env << EOF
   NODE_ENV=production
   PORT=3000
   DATABASE_URL=postgresql://user:pass@localhost:5432/mydb
   API_KEY=test-api-key
   DEBUG=false
   EOF
   ```

2. **Test deployment with environment variables**:
   ```bash
   # First, register a user and get a JWT token
   curl -X POST "http://localhost:8080/api/v1/auth/register" \
     -H "Content-Type: application/json" \
     -d '{
       "username": "testuser",
       "email": "test@example.com",
       "password": "password123"
     }'

   # Login to get JWT token
   curl -X POST "http://localhost:8080/api/v1/auth/login" \
     -H "Content-Type: application/json" \
     -d '{
       "username": "testuser",
       "password": "password123"
     }'

   # Use the JWT token to create a deployment with environment variables
   curl -X POST "http://localhost:8080/api/v1/deployments" \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -F "target_ip=YOUR_SERVER_IP" \
     -F "ssh_username=root" \
     -F "ssh_password=YOUR_PASSWORD" \
     -F "github_repo_url=https://github.com/user/repo" \
     -F "github_pat=YOUR_GITHUB_TOKEN" \
     -F "github_branch=main" \
     -F "port=3000" \
     -F "container_name=test-app" \
     -F "project_name=test-project" \
     -F "deployment_name=Test Deployment" \
     -F "env_file=@test.env"
   ```

3. **Verify environment variables in container**:
   ```bash
   # SSH to your target server and check the container
   ssh root@YOUR_SERVER_IP
   docker exec CONTAINER_ID env | grep -E "(NODE_ENV|PORT|DATABASE_URL|API_KEY|DEBUG)"
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/handlers -v
```

### Code Formatting

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run
```

### Building for Production

```bash
# Build server
go build -o bin/server cmd/server/main.go

# Build worker
go build -o bin/worker cmd/worker/main.go

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o bin/server-linux cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o bin/server-windows.exe cmd/server/main.go
```

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```bash
   # Check if PostgreSQL is running
   docker ps | grep postgres
   
   # Check connection
   psql -h localhost -U postgres -d deployknot
   ```

2. **Redis Connection Failed**
   ```bash
   # Check if Redis is running
   docker ps | grep redis
   
   # Check connection
   redis-cli -h localhost -p 6379 ping
   ```

3. **Port Already in Use**
   ```bash
   # Check what's using port 8080
   lsof -i :8080
   
   # Kill the process
   kill -9 <PID>
   ```

4. **Migration Errors**
   ```bash
   # Check migration status
   migrate -path migrations -database "postgres://postgres:password@localhost:5432/deployknot?sslmode=disable&search_path=deploy_knot" version
   
   # Force migration version
   migrate -path migrations -database "postgres://postgres:password@localhost:5432/deployknot?sslmode=disable&search_path=deploy_knot" force <version>
   ```

### Debug Mode

```bash
# Run with debug logging
LOG_LEVEL=debug go run cmd/server/main.go

# Run with verbose output
go run -v cmd/server/main.go
```

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

## Next Steps

1. **Set up authentication**: Configure JWT tokens
2. **Test deployments**: Create test deployments
3. **Monitor logs**: Set up log monitoring
4. **Add tests**: Write comprehensive tests
5. **Deploy to production**: Use the deployment guides

## Support

- **GitHub Issues**: For bug reports and feature requests
- **Documentation**: Check other `.md` files in the repository
- **API Reference**: Use the Postman collection for API testing 