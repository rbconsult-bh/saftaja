CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ORGANIZATIONS
-- ============================================================================

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(100) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================================================
-- PROJECTS
-- ============================================================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),

    name VARCHAR(100) NOT NULL,
    environment VARCHAR(20) NOT NULL DEFAULT 'sandbox', -- sandbox, production
    custom_domain VARCHAR(255) UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(organization_id, environment)
);

-- ============================================================================
-- GATEWAY ACCOUNTS
-- ============================================================================

CREATE TABLE gateway_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id),

    connector_type VARCHAR(50) NOT NULL, -- "mpgs"

    account_name VARCHAR(100) NOT NULL,

    -- 🔒 SECRETS (Password, Merchant ID, Keys) - Never send to frontend
    credentials JSONB NOT NULL DEFAULT '{}',

    -- 🌍 SETTINGS (Display Name, Test Mode, Logos) - Safe for frontend
    settings JSONB NOT NULL DEFAULT '{}',

    -- 🧭 ROUTING (e.g. ['card', 'apple_pay'])
    payment_methods JSONB NOT NULL DEFAULT '[]',

    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================================================
-- INVOICES
-- ============================================================================

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id),

    amount DECIMAL(12, 3) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'BHD',

    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- pending, processing, paid, failed

    external_id VARCHAR(100),
    customer_email VARCHAR(255),
    customer_name VARCHAR(100),
    description TEXT,

    paid_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(project_id, external_id)
);

CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),

    name VARCHAR(255) NOT NULL,
    description TEXT,

    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price DECIMAL(12, 3) NOT NULL,

    amount DECIMAL(12, 3) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- PAYMENT SESSIONS
-- ============================================================================

CREATE TABLE payment_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    gateway_account_id UUID NOT NULL REFERENCES gateway_accounts(id),

    gateway_session_id VARCHAR(100) NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'created', -- created, ready_to_start_challenge, awaiting_challenge_completion, ready_to_capture, capturing_payment, succeeded, failed

    payment_method VARCHAR(30) NOT NULL DEFAULT 'card', -- card, apple_pay

    payer_ip VARCHAR(50),
    payer_user_agent TEXT,

    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 minutes'),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================================================
-- TRANSACTIONS
-- ============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_session_id UUID NOT NULL REFERENCES payment_sessions(id),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    project_id UUID NOT NULL REFERENCES projects(id),

    transaction_type VARCHAR(30) NOT NULL, -- initiate_authentication, authenticate_payer, pay
    gateway_transaction_id VARCHAR(100) NOT NULL,

    amount DECIMAL(12, 3) NOT NULL,
    currency VARCHAR(3) NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- pending, success, failed

    raw_request JSONB,
    raw_response JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- ============================================================================
-- UPDATED_AT TRIGGER
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_gateway_accounts_updated_at BEFORE UPDATE ON gateway_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoices_updated_at BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payment_sessions_updated_at BEFORE UPDATE ON payment_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
