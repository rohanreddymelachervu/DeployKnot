# Multipart Form Upload for Environment Variables

## Overview

DeployKnot now supports **multipart form upload** for environment variables, allowing you to upload your existing `.env` files directly to the deployment API. This is the **recommended approach** for environment variable handling as it provides better security, easier management, and eliminates encoding issues.

## 🚀 Key Benefits

### ✅ **Enhanced Security**
- **No sensitive data in JSON** - Environment variables are uploaded as files
- **Secure file handling** - Files are processed securely and cleaned up after deployment
- **No encoding issues** - Eliminates problems with special characters, quotes, and newlines

### ✅ **Better User Experience**
- **Direct file upload** - Upload your existing `.env` files without modification
- **No string escaping** - No need to escape quotes, newlines, or special characters
- **Preserves formatting** - Maintains your original `.env` file structure

### ✅ **Improved Reliability**
- **Robust processing** - Handles complex environment files with comments and special characters
- **Automatic validation** - Validates environment variable format
- **Error handling** - Clear error messages for malformed files

## 📋 API Usage

### Multipart Form Upload (Recommended)

```bash
curl --location 'http://localhost:8080/api/v1/deployments' \
--form 'target_ip="172.235.15.164"' \
--form 'ssh_username="root"' \
--form 'ssh_password="your-password"' \
--form 'github_repo_url="https://github.com/your-username/your-repo"' \
--form 'github_pat="***REMOVED***"' \
--form 'github_branch="main"' \
--form 'port="3000"' \
--form 'container_name="my-app"' \
--form 'project_name="my-project"' \
--form 'deployment_name="Production Deployment"' \
--form 'env_file=@/path/to/your/.env'
```

### Using with Postman

1. **Set request method to `POST`**
2. **Set URL to `{{base_url}}/api/v1/deployments`**
3. **Set body type to `form-data`**
4. **Add form fields:**
   - `target_ip` (text)
   - `ssh_username` (text)
   - `ssh_password` (text)
   - `github_repo_url` (text)
   - `github_pat` (text)
   - `github_branch` (text)
   - `port` (text)
   - `container_name` (text, optional)
   - `project_name` (text, optional)
   - `deployment_name` (text, optional)
   - `env_file` (file) - Upload your `.env` file here

## 📁 .env File Format

Your `.env` file should follow standard environment variable format:

```env
# Production environment variables
NODE_ENV=production
PORT=3000

# Database configuration
DATABASE_URL=postgresql://user:pass@localhost:5432/mydb

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

## 🔧 Processing Features

### ✅ **Automatic Processing**
- **Comment filtering** - Lines starting with `#` are automatically removed
- **Empty line filtering** - Blank lines are removed for cleaner files
- **Quote handling** - Single and double quotes are properly processed
- **Validation** - Ensures proper `KEY=VALUE` format

### ✅ **Docker Integration**
- **`--env-file` support** - Uses Docker's native environment file feature
- **Unique file paths** - Each deployment gets a unique `.env` file path
- **Secure transfer** - Files are securely copied to target instances via SFTP

## 🔄 Migration from Legacy JSON

### Old Approach (Legacy)
```json
{
  "target_ip": "172.235.15.164",
  "ssh_username": "root",
  "ssh_password": "password",
  "github_repo_url": "https://github.com/user/repo",
  "github_pat": "ghp_token",
  "github_branch": "main",
  "environment_vars": "NODE_ENV=production\nPORT=3000\nDATABASE_URL=postgresql://user:pass@localhost:5432/mydb",
  "port": 3000,
  "container_name": "my-app"
}
```

### New Approach (Recommended)
```bash
curl --location 'http://localhost:8080/api/v1/deployments' \
--form 'target_ip="172.235.15.164"' \
--form 'ssh_username="root"' \
--form 'ssh_password="password"' \
--form 'github_repo_url="https://github.com/user/repo"' \
--form 'github_pat="ghp_token"' \
--form 'github_branch="main"' \
--form 'port="3000"' \
--form 'container_name="my-app"' \
--form 'env_file=@.env'
```

## 🛠️ Technical Implementation

### File Upload Process
1. **File Upload** - `.env` file is uploaded via multipart form
2. **Temporary Storage** - File is saved to a unique temporary location
3. **SFTP Transfer** - File is securely copied to target instance
4. **Docker Integration** - Container is started with `--env-file` flag
5. **Cleanup** - Temporary files are cleaned up after deployment

### Worker Processing
```go
// The worker processes the uploaded .env file
if envFilePath := getStringFromMap(job.Data, "env_file_path"); envFilePath != "" {
    // Copy file to target instance via SFTP
    // Use Docker --env-file flag
    // Clean up temporary files
}
```

## 📊 Response Format

Successful deployment response:
```json
{
  "id": "f20e8fc3-42f3-4f7a-91e4-dff6649af518",
  "status": "pending",
  "target_ip": "172.235.15.164",
  "github_repo_url": "https://github.com/your-username/your-repo",
  "github_branch": "main",
  "port": 3005,
  "container_name": "multipart-env-test",
  "created_at": "2025-07-11T23:32:37.504075+05:30",
  "project_name": "multipart-test-project",
  "deployment_name": "Multipart Env Test"
}
```

## 🔍 Monitoring and Logs

### Deployment Logs
The deployment logs will show the `.env` file processing:

```
"Copying uploaded .env file to target instance"
"Uploaded .env file to target instance"
"Starting Docker container with uploaded .env file"
```

### Health Check
Monitor your container to ensure environment variables are loaded:
```bash
docker exec your-container-name env | grep NODE_ENV
```

## 🚨 Error Handling

### Common Issues
1. **Invalid .env format** - Ensure `KEY=VALUE` format
2. **File too large** - Keep `.env` files under reasonable size
3. **Network issues** - Check SFTP connectivity to target instance
4. **Permission issues** - Ensure proper file permissions

### Error Responses
```json
{
  "error": "Invalid request",
  "message": "port validation failed: invalid port number: invalid"
}
```

## 🔒 Security Considerations

### File Security
- **Temporary storage** - Files are stored in secure temporary locations
- **Automatic cleanup** - Files are deleted after deployment
- **Unique paths** - Each deployment gets a unique file path
- **SFTP encryption** - Files are transferred securely

### Best Practices
1. **Use HTTPS** - Always use HTTPS for API communication
2. **Secure credentials** - Store sensitive credentials securely
3. **File permissions** - Ensure proper file permissions on target instances
4. **Regular cleanup** - Monitor and clean up temporary files

## 📈 Performance

### Benefits
- **Faster processing** - No string parsing overhead
- **Reduced memory usage** - Direct file handling
- **Better scalability** - Handles large environment files efficiently
- **Improved reliability** - Fewer encoding-related failures

## 🔄 Backward Compatibility

The legacy JSON approach with `environment_vars` string is still supported for backward compatibility, but the multipart form upload is **strongly recommended** for new deployments.

## 📝 Examples

### Simple .env File
```env
NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@localhost:5432/mydb
API_KEY=sk-1234567890abcdef
DEBUG=false
```

### Complex .env File
```env
# Production environment variables
NODE_ENV=production
PORT=3000

# Database configuration
DATABASE_URL="postgresql://user:pass@localhost:5432/mydb"

# API Keys
API_KEY='sk-1234567890abcdef'

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

## 🎯 Summary

The multipart form upload feature provides a **superior approach** to environment variable handling in DeployKnot:

- ✅ **More secure** - No sensitive data in JSON
- ✅ **Easier to use** - Direct file upload
- ✅ **More reliable** - No encoding issues
- ✅ **Better performance** - Efficient file processing
- ✅ **Enhanced monitoring** - Clear logging of file processing

**Start using multipart form upload for your deployments today!** 