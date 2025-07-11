-- Add port and container_name columns to deployments table
ALTER TABLE deploy_knot.deployments 
ADD COLUMN port INTEGER DEFAULT 3000,
ADD COLUMN container_name VARCHAR(200);

-- Create index for container_name queries
CREATE INDEX idx_deployments_container_name ON deploy_knot.deployments(container_name); 