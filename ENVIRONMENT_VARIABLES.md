# Environment Variables in DeployKnot

## Overview

DeployKnot supports comprehensive environment variable management for both application configuration and deployment targets. This document covers all environment variables used throughout the application.

## Application Environment Variables

### Server Configuration

```env
# Server Configuration
SERVER_PORT=8080                    # Port for the HTTP server
SERVER_READ_TIMEOUT=30s            # HTTP read timeout
SERVER_WRITE_TIMEOUT=30s           # HTTP write timeout
SERVER_IDLE_TIMEOUT=60s            # HTTP idle timeout
```

### Database Configuration

```env
# Database Configuration
DB_HOST=localhost                   # PostgreSQL host
DB_PORT=5432                       # PostgreSQL port
DB_USER=postgres                   # Database username
DB_PASSWORD=password               # Database password
DB_NAME=deployknot                # Database name
DB_SSLMODE=disable                # SSL mode (disable/require/verify-ca/verify-full)
DB_SCHEMA=deploy_knot             # Database schema (important!)
```

### Redis Configuration

```env
# Redis Configuration
REDIS_HOST=localhost               # Redis host
REDIS_PORT=6379                   # Redis port
REDIS_PASSWORD=                   # Redis password (empty for local)
REDIS_DB=0                        # Redis database number
```

### Logging Configuration

```env
# Logging Configuration
LOG_LEVEL=info                    # Log level (debug, info, warn, error)
```

### JWT Configuration

```env
# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

## Deployment Environment Variables

### Environment Variables Format

The `environment_vars` field accepts a string in `.env` file format for deployment targets:

```json
{
  "environment_vars": "NODE_ENV=production\nPORT=3000\nDATABASE_URL=postgresql://user:pass@localhost:5432/db"
}
```

### Supported Formats

#### 1. Simple Key-Value Pairs
```
NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@localhost:5432/db
API_KEY=sk-1234567890abcdef
```

#### 2. Quoted Values
```
NODE_ENV="production"
DATABASE_URL="postgresql://user:pass@localhost:5432/db"
API_KEY='sk-1234567890abcdef'
MESSAGE="Hello, World!"
```

#### 3. With Comments
```
# Production environment variables
NODE_ENV=production
PORT=3000

# Database configuration
DATABASE_URL=postgresql://user:pass@localhost:5432/db

# API Keys
API_KEY=sk-1234567890abcdef
```

#### 4. Complex Configurations
```
# Production environment variables
NODE_ENV=production
PORT=3000

# Database configuration
DATABASE_URL=postgresql://user:pass@localhost:5432/db

# API Keys
API_KEY=sk-1234567890abcdef

# Redis configuration
REDIS_URL=redis://localhost:6379

# AWS Configuration
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
AWS_REGION=us-west-2

# Stripe configuration
STRIPE_SECRET_KEY=sk_test_51H1234567890abcdef

# JWT and Security
JWT_SECRET=your-super-secret-jwt-key-here
SESSION_SECRET=another-super-secret-key

# CORS and Logging
CORS_ORIGIN=https://example.com
LOG_LEVEL=debug

# File upload settings
MAX_FILE_SIZE=10485760

# Email configuration
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_USER=user@example.com
EMAIL_PASS=email-password-here
```

## API Examples

### Basic Deployment with Environment Variables

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "target_ip": "192.168.1.100",
    "ssh_username": "root",
    "ssh_password": "your-ssh-password",
    "github_repo_url": "https://github.com/example/repo",
    "github_pat": "your-github-pat",
    "github_branch": "main",
    "environment_vars": "NODE_ENV=production\nPORT=3000\nDATABASE_URL=postgresql://user:pass@localhost:5432/db\nAPI_KEY=sk-1234567890abcdef",
    "port": 3001,
    "container_name": "my-app-env",
    "project_name": "my-project",
    "deployment_name": "production-deploy-env"
  }'
```

### Complex Environment Variables

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "target_ip": "192.168.1.100",
    "ssh_username": "root",
    "ssh_password": "your-ssh-password",
    "github_repo_url": "https://github.com/example/repo",
    "github_pat": "your-github-pat",
    "github_branch": "main",
    "environment_vars": "# Production environment variables\nNODE_ENV=production\nPORT=3000\n\n# Database configuration\nDATABASE_URL=postgresql://user:pass@localhost:5432/db\n\n# API Keys\nAPI_KEY=sk-1234567890abcdef\n\n# Redis configuration\nREDIS_URL=redis://localhost:6379\n\n# AWS Configuration\nAWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nAWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\nAWS_REGION=us-west-2",
    "port": 3002,
    "container_name": "my-app-complex-env",
    "project_name": "my-project",
    "deployment_name": "complex-env-deploy"
  }'
```

### Environment File Upload

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "target_ip=192.168.1.100" \
  -F "ssh_username=root" \
  -F "ssh_password=your-ssh-password" \
  -F "github_repo_url=https://github.com/example/repo" \
  -F "github_pat=your-github-pat" \
  -F "github_branch=main" \
  -F "port=3000" \
  -F "container_name=my-app" \
  -F "project_name=my-project" \
  -F "deployment_name=production-deploy" \
  -F "env_file=@/path/to/your/.env"
```

## How It Works

### 1. Environment Variable Processing

The system processes environment variables through several steps:

1. **Input Validation**: Checks for proper KEY=VALUE format
2. **Comment Removal**: Filters out lines starting with `#`
3. **Quote Processing**: Handles single and double quotes
4. **Empty Line Filtering**: Removes blank lines
5. **Format Standardization**: Ensures consistent formatting

### 2. .env File Creation

For each deployment with environment variables:

1. **Unique File Path**: Creates `/tmp/deployknot-env-{deployment-id}.env`
2. **Content Writing**: Writes processed environment variables to file
3. **Verification**: Confirms file creation and content
4. **Logging**: Records environment setup process

### 3. Docker Integration

The Docker container is started with:

```bash
docker run -d --name {container-name} -p {port}:{port} --env-file /tmp/deployknot-env-{deployment-id}.env {image-name}:latest
```

## Database Schema Requirements

### PostgreSQL Schema Setup

The application requires the `deploy_knot` schema to be created in PostgreSQL:

```sql
-- Create schema
CREATE SCHEMA IF NOT EXISTS deploy_knot;

-- Verify schema exists
SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'deploy_knot';
```

### Docker Setup

When using Docker for PostgreSQL, ensure the schema is created:

```bash
# Start PostgreSQL
docker run --name postgres-deployknot \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=deployknot \
  -e POSTGRES_USER=postgres \
  -p 5432:5432 \
  -d postgres:15-alpine

# Create schema (wait for PostgreSQL to start)
sleep 5
docker exec postgres-deployknot psql -U postgres -d deployknot -c "CREATE SCHEMA IF NOT EXISTS deploy_knot;"
```

## Monitoring & Logs

### Environment Variable Logs

The system logs environment variable processing:

```
🔍 Environment file created and verified: /tmp/deployknot-env-{deployment-id}.env
🔍 Environment variables file created successfully
🔍 Environment file content:
NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@localhost:5432/db
API_KEY=sk-1234567890abcdef
```

### Deployment Logs

Monitor environment variable processing through deployment logs:

```bash
curl -N http://localhost:8080/api/v1/deployments/{deployment-id}/logs
```

## Best Practices

### ✅ Do's

1. **Use descriptive variable names**
   ```
   DATABASE_URL=postgresql://user:pass@localhost:5432/db
   API_KEY=sk-1234567890abcdef
   ```

2. **Include comments for organization**
   ```
   # Database configuration
   DATABASE_URL=postgresql://user:pass@localhost:5432/db
   
   # API Keys
   API_KEY=sk-1234567890abcdef
   ```

3. **Use quotes for values with spaces or special characters**
   ```
   MESSAGE="Hello, World!"
   PATH="/usr/local/bin:/usr/bin:/bin"
   ```

4. **Group related variables**
   ```
   # AWS Configuration
   AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
   AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
   AWS_REGION=us-west-2
   ```

5. **Always set DB_SCHEMA=deploy_knot**
   ```
   DB_SCHEMA=deploy_knot
   ```

### ❌ Don'ts

1. **Don't include sensitive data in comments**
   ```
   # Bad: API_KEY=sk-1234567890abcdef  # This is my API key
   ```

2. **Don't use invalid characters in variable names**
   ```
   # Bad: MY-VAR=value
   # Good: MY_VAR=value
   ```

3. **Don't forget to create the deploy_knot schema**
   ```sql
   -- Always run this for new databases
   CREATE SCHEMA IF NOT EXISTS deploy_knot;
   ```

## Troubleshooting

### Common Issues

1. **Database Schema Not Found**
   ```bash
   # Error: schema "deploy_knot" does not exist
   # Solution: Create the schema
   psql -h localhost -U postgres -d deployknot -c "CREATE SCHEMA IF NOT EXISTS deploy_knot;"
   ```

2. **Environment Variables Not Applied**
   - Check the deployment logs for environment file creation
   - Verify the environment variables format
   - Ensure the target container supports environment variables

3. **Database Connection Failed**
   - Verify DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
   - Check if PostgreSQL is running
   - Ensure DB_SCHEMA=deploy_knot is set

4. **Redis Connection Failed**
   - Verify REDIS_HOST, REDIS_PORT, REDIS_PASSWORD
   - Check if Redis is running
   - Test connection: `redis-cli -h localhost -p 6379 ping`

### Debug Environment Variables

```bash
# Check application environment variables
echo $DATABASE_URL
echo $REDIS_URL
echo $JWT_SECRET

# Check deployment environment variables in logs
curl -N http://localhost:8080/api/v1/deployments/{deployment-id}/logs | grep -i "environment"
```

## Security Considerations

1. **Never commit sensitive environment variables to version control**
2. **Use strong JWT secrets in production**
3. **Rotate API keys and secrets regularly**
4. **Use environment-specific configurations**
5. **Validate environment variables on application startup**

## Production Checklist

- [ ] Set `JWT_SECRET` to a strong, unique value
- [ ] Configure `DB_SCHEMA=deploy_knot`
- [ ] Set appropriate log levels (`LOG_LEVEL=info` or `LOG_LEVEL=warn`)
- [ ] Configure production database credentials
- [ ] Set up Redis with authentication if needed
- [ ] Configure server timeouts for production load
- [ ] Test environment variable processing
- [ ] Verify database schema creation 