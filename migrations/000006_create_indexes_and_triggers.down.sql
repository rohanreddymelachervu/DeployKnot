-- Drop triggers
DROP TRIGGER IF EXISTS update_deployment_templates_updated_at ON deploy_knot.deployment_templates;
DROP TRIGGER IF EXISTS update_projects_updated_at ON deploy_knot.projects;
DROP TRIGGER IF EXISTS update_deployments_updated_at ON deploy_knot.deployments;

-- Drop function
DROP FUNCTION IF EXISTS deploy_knot.update_updated_at_column();
