-- Create deployment_logs table
CREATE TABLE deploy_knot.deployment_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deploy_knot.deployments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    log_level VARCHAR(10) NOT NULL DEFAULT 'info' CHECK (log_level IN ('info', 'warn', 'error', 'debug')),
    message TEXT NOT NULL,
    task_name VARCHAR(100),
    step_order INTEGER
);

-- Create indexes for performance
CREATE INDEX idx_deployment_logs_deployment_id ON deploy_knot.deployment_logs(deployment_id);
CREATE INDEX idx_deployment_logs_created_at ON deploy_knot.deployment_logs(created_at);
