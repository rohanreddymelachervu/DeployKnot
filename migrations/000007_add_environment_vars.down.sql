-- Remove environment_vars field from deployments table
ALTER TABLE deploy_knot.deployments 
DROP COLUMN environment_vars; 