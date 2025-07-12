# Enhanced Environment Variables in DeployKnot

## Overview

DeployKnot now supports enhanced environment variable handling with proper `.env` file creation and Docker `--env-file` integration. This replaces the previous basic string-based approach with a more robust and secure solution.

## Features

### ✅ Enhanced Environment Variable Processing
- **Automatic .env file creation** with unique paths per deployment
- **Comment filtering** - automatically removes lines starting with `#`
- **Quote handling** - properly processes single and double quotes
- **Validation** - ensures proper KEY=VALUE format
- **Empty line filtering** - removes blank lines for cleaner files

### ✅ Docker Integration
- **`--env-file` support** - uses Docker's native environment file feature
- **Unique file paths** - each deployment gets its own environment file
- **Verification** - confirms file creation and content
- **Cleanup** - environment files are created in `/tmp/` for easy cleanup

### ✅ Security & Validation
- **Type-safe extraction** - robust handling of different data types
- **Input validation** - ensures proper environment variable format
- **Error handling** - comprehensive error reporting and logging

## API Usage

### Environment Variables Format

The `environment_vars` field accepts a string in `.env` file format:

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
  -d '{
    "target_ip": "172.235.15.164",
    "ssh_username": "root",
    "ssh_password": "your-ssh-password",
    "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
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
  -d '{
    "target_ip": "172.235.15.164",
    "ssh_username": "root",
    "ssh_password": "your-ssh-password",
    "github_repo_url": "https://github.com/rohanreddymelachervu/DeployKnot-test",
    "github_pat": "your-github-pat",
    "github_branch": "main",
    "environment_vars": "# Production environment variables\nNODE_ENV=production\nPORT=3000\n\n# Database configuration\nDATABASE_URL=postgresql://user:pass@localhost:5432/db\n\n# API Keys\nAPI_KEY=sk-1234567890abcdef\n\n# Redis configuration\nREDIS_URL=redis://localhost:6379\n\n# AWS Configuration\nAWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\nAWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\nAWS_REGION=us-west-2",
    "port": 3002,
    "container_name": "my-app-complex-env",
    "project_name": "my-project",
    "deployment_name": "complex-env-deploy"
  }'
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
curl http://localhost:8080/api/v1/deployments/{deployment-id}/logs
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

3. **Don't forget to escape quotes properly in JSON**
   ```json
   {
     "environment_vars": "MESSAGE=\"Hello, World!\""
   }
   ```

## Troubleshooting

### Common Issues

1. **Environment variables not being set**
   - Check the deployment logs for environment processing errors
   - Verify the .env file was created successfully
   - Ensure proper KEY=VALUE format

2. **Quotes not being handled correctly**
   - The system automatically removes quotes from values
   - Use proper JSON escaping for quotes in the API request

3. **Comments not being filtered**
   - Comments starting with `#` are automatically removed
   - Check that comments are on their own lines

### Debug Commands

Check environment file creation:
```bash
# On the target server
ls -la /tmp/deployknot-env-*
cat /tmp/deployknot-env-{deployment-id}.env
```

Verify Docker container environment:
```bash
# On the target server
docker exec {container-name} env
```

## Migration from Old Format

The enhanced environment variable handling is backward compatible. Existing deployments will continue to work, but you can now take advantage of:

- Better error handling
- Improved logging
- Comment support
- Quote handling
- Validation

## Security Considerations

1. **Environment files are created in `/tmp/`** for easy cleanup
2. **Unique file names** prevent conflicts between deployments
3. **Input validation** prevents malformed environment variables
4. **Logging** provides audit trail for environment variable processing

## Performance

- **Minimal overhead**: Environment variable processing is fast
- **Efficient file handling**: Uses Docker's native `--env-file` feature
- **Memory efficient**: Processes variables line by line
- **Cleanup**: Environment files are in `/tmp/` for automatic cleanup 