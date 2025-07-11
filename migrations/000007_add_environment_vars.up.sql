-- Add environment_vars field to deployments table
ALTER TABLE deploy_knot.deployments 
ADD COLUMN environment_vars TEXT; 