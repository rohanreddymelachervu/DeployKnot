-- Create deployment_templates table
CREATE TABLE deploy_knot.deployment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    playbook_template TEXT NOT NULL,
    default_vars JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

-- Create index for active templates
CREATE INDEX idx_deployment_templates_is_active ON deploy_knot.deployment_templates(is_active);
