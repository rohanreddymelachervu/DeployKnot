# Docker Hub Setup Guide

This guide will help you set up Docker Hub credentials for the GitHub Actions workflow to automatically build and push Docker images.

## Prerequisites

1. A Docker Hub account (you mentioned you just created one)
2. A GitHub repository with your DeployKnot project
3. Admin access to your GitHub repository

## Step 1: Create Docker Hub Access Token

1. **Log in to Docker Hub** at https://hub.docker.com
2. **Go to Account Settings**:
   - Click on your username in the top right
   - Select "Account Settings"
3. **Navigate to Security**:
   - Click on "Security" in the left sidebar
4. **Create Access Token**:
   - Click "New Access Token"
   - Give it a name like "GitHub Actions DeployKnot"
   - Set expiration (recommend 90 days or longer)
   - Click "Generate"
5. **Copy the token** - you'll need this for the next step

## Step 2: Add Secrets to GitHub Repository

1. **Go to your GitHub repository**
2. **Navigate to Settings**:
   - Click on "Settings" tab in your repository
3. **Go to Secrets and variables**:
   - Click on "Secrets and variables" in the left sidebar
   - Select "Actions"
4. **Add the following secrets**:

   ### DOCKERHUB_USERNAME
   - **Name**: `DOCKERHUB_USERNAME`
   - **Value**: Your Docker Hub username
   - **Description**: Docker Hub username for authentication

   ### DOCKERHUB_TOKEN
   - **Name**: `DOCKERHUB_TOKEN`
   - **Value**: The access token you created in Step 1
   - **Description**: Docker Hub access token for authentication

## Step 3: Understanding the Workflow

The GitHub Actions workflow (`/.github/workflows/docker-build.yml`) will:

1. **Trigger on**:
   - Pushes to `main` or `develop` branches
   - Tags starting with `v*` (e.g., `v1.0.0`)
   - Pull requests to `main` branch

2. **Build two Docker images**:
   - `deployknot-server`: Your API server
   - `deployknot-worker`: Your deployment worker

3. **Push to Docker Hub** with tags:
   - Branch names (e.g., `main`, `develop`)
   - Git commit SHA (e.g., `main-abc123`)
   - Semantic version tags (e.g., `v1.0.0`, `v1.0`)

## Step 4: Testing the Workflow

1. **Push to main branch**:
   ```bash
   git add .
   git commit -m "Add Docker build workflow"
   git push origin main
   ```

2. **Check the workflow**:
   - Go to your GitHub repository
   - Click on "Actions" tab
   - You should see the "Build and Push Docker Images" workflow running

3. **Verify on Docker Hub**:
   - Go to https://hub.docker.com
   - You should see your images: `yourusername/deployknot-server` and `yourusername/deployknot-worker`

## Step 5: Using the Images

Once the workflow runs successfully, you can pull and run the images:

### Server Image
```bash
docker pull yourusername/deployknot-server:main
docker run -p 8080:8080 \
  -e DATABASE_URL="your_db_url" \
  -e REDIS_URL="your_redis_url" \
  -e JWT_SECRET="your_jwt_secret" \
  yourusername/deployknot-server:main
```

### Worker Image
```bash
docker pull yourusername/deployknot-worker:main
docker run \
  -e DATABASE_URL="your_db_url" \
  -e REDIS_URL="your_redis_url" \
  yourusername/deployknot-worker:main
```

## Step 6: Creating a Release

To create a tagged release:

1. **Create and push a tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **The workflow will automatically**:
   - Build images with tags: `v1.0.0`, `v1.0`, `latest`
   - Push them to Docker Hub

## Troubleshooting

### Common Issues

1. **Authentication failed**:
   - Verify your `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets are correct
   - Ensure the token hasn't expired

2. **Build fails**:
   - Check the GitHub Actions logs for specific error messages
   - Ensure all dependencies are properly specified in `go.mod`

3. **Images not appearing on Docker Hub**:
   - Wait a few minutes for the push to complete
   - Check that your Docker Hub account has the necessary permissions

### Security Best Practices

1. **Use Access Tokens**: Never use your Docker Hub password in GitHub secrets
2. **Rotate Tokens**: Regularly update your access tokens
3. **Limit Permissions**: Create tokens with minimal required permissions
4. **Monitor Usage**: Regularly check your Docker Hub account for unexpected activity

## Next Steps

1. **Set up environment variables** for your deployment
2. **Configure your deployment infrastructure** to use these Docker images
3. **Set up monitoring** for your containerized applications
4. **Consider using Docker Compose** for local development

## Support

If you encounter issues:
1. Check the GitHub Actions logs for detailed error messages
2. Verify your Docker Hub credentials
3. Ensure your repository has the correct permissions
4. Review the workflow file for any syntax errors 