-- Remove port and container_name columns from deployments table
ALTER TABLE deploy_knot.deployments 
DROP COLUMN IF EXISTS port,
DROP COLUMN IF EXISTS container_name;

-- Drop index for container_name
DROP INDEX IF EXISTS idx_deployments_container_name; 