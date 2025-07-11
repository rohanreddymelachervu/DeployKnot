-- Create deployments table
CREATE TABLE deploy_knot.deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    target_ip VARCHAR(45) NOT NULL,
    ssh_username VARCHAR(100) NOT NULL,
    ssh_password_encrypted TEXT,
    github_repo_url VARCHAR(500) NOT NULL,
    github_pat_encrypted TEXT,
    github_branch VARCHAR(100) NOT NULL DEFAULT 'main',
    additional_vars JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_by VARCHAR(100),
    project_name VARCHAR(200),
    deployment_name VARCHAR(200)
);

-- Create index for status queries
CREATE INDEX idx_deployments_status ON deploy_knot.deployments(status);
CREATE INDEX idx_deployments_created_at ON deploy_knot.deployments(created_at);
