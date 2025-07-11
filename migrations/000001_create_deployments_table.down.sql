-- Drop deployments table and indexes
DROP INDEX IF EXISTS deploy_knot.idx_deployments_created_at;
DROP INDEX IF EXISTS deploy_knot.idx_deployments_status;
DROP TABLE IF EXISTS deploy_knot.deployments;
