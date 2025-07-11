-- Drop deployment_steps table and indexes
DROP INDEX IF EXISTS deploy_knot.idx_deployment_steps_status;
DROP INDEX IF EXISTS deploy_knot.idx_deployment_steps_deployment_id;
DROP TABLE IF EXISTS deploy_knot.deployment_steps;
