-- Drop deployment_templates table and indexes
DROP INDEX IF EXISTS deploy_knot.idx_deployment_templates_is_active;
DROP TABLE IF EXISTS deploy_knot.deployment_templates;
