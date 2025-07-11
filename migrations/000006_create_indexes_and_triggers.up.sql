-- Create function for updating updated_at column
CREATE OR REPLACE FUNCTION deploy_knot.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_deployments_updated_at 
    BEFORE UPDATE ON deploy_knot.deployments
    FOR EACH ROW EXECUTE FUNCTION deploy_knot.update_updated_at_column();

CREATE TRIGGER update_projects_updated_at 
    BEFORE UPDATE ON deploy_knot.projects
    FOR EACH ROW EXECUTE FUNCTION deploy_knot.update_updated_at_column();

CREATE TRIGGER update_deployment_templates_updated_at 
    BEFORE UPDATE ON deploy_knot.deployment_templates
    FOR EACH ROW EXECUTE FUNCTION deploy_knot.update_updated_at_column();
