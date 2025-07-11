-- Create deployment_steps table
CREATE TABLE deploy_knot.deployment_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deploy_knot.deployments(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    error_message TEXT,
    step_order INTEGER NOT NULL
);

-- Create indexes for performance
CREATE INDEX idx_deployment_steps_deployment_id ON deploy_knot.deployment_steps(deployment_id);
CREATE INDEX idx_deployment_steps_status ON deploy_knot.deployment_steps(status);
