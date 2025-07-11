-- Drop deployment_logs table and indexes
DROP INDEX IF EXISTS deploy_knot.idx_deployment_logs_created_at;
DROP INDEX IF EXISTS deploy_knot.idx_deployment_logs_deployment_id;
DROP TABLE IF EXISTS deploy_knot.deployment_logs;
