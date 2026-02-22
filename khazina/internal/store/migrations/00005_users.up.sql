-- ============================================================================
-- USERS
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- ORGANIZATION MEMBERS
-- ============================================================================

CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id         UUID NOT NULL REFERENCES users(id),

    role VARCHAR(20) NOT NULL DEFAULT 'member', -- owner, admin, member, viewer

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(organization_id, user_id)
);

-- ============================================================================
-- API KEYS
-- ============================================================================

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_id UUID NOT NULL REFERENCES organizations(id),
    project_id      UUID REFERENCES projects(id), -- NULL = all projects in org
    user_id         UUID NOT NULL REFERENCES users(id),

    name       VARCHAR(100) NOT NULL,
    key_hash   VARCHAR(255) NOT NULL UNIQUE, -- sha256 hex of actual key
    key_prefix VARCHAR(12)  NOT NULL,        -- first 12 chars for display

    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- INVITATIONS
-- ============================================================================

CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_id UUID NOT NULL REFERENCES organizations(id),

    email      VARCHAR(255) NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'member',
    invited_by UUID         NOT NULL REFERENCES users(id),
    token      VARCHAR(100) NOT NULL UNIQUE,

    accepted_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- PROJECTS: add slug + theme columns
-- ============================================================================

ALTER TABLE projects
    ADD COLUMN slug  VARCHAR(50) UNIQUE,
    ADD COLUMN theme JSONB NOT NULL DEFAULT '{}';
