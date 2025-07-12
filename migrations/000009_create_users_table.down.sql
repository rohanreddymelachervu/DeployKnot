-- Remove user_id from deployments table
ALTER TABLE deploy_knot.deployments DROP COLUMN IF EXISTS user_id;

-- Drop users table
DROP TABLE IF EXISTS deploy_knot.users; 