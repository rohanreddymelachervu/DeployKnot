-- Create users table
CREATE TABLE deploy_knot.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for users
CREATE INDEX idx_users_username ON deploy_knot.users(username);
CREATE INDEX idx_users_email ON deploy_knot.users(email);
CREATE INDEX idx_users_is_active ON deploy_knot.users(is_active);

-- Add user_id to deployments table
ALTER TABLE deploy_knot.deployments ADD COLUMN user_id UUID REFERENCES deploy_knot.users(id) ON DELETE CASCADE;

-- Create index for user-specific deployments
CREATE INDEX idx_deployments_user_id ON deploy_knot.deployments(user_id); 