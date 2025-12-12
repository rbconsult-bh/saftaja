DROP TRIGGER IF EXISTS update_transactions_updated_at ON transactions;
DROP TRIGGER IF EXISTS update_payment_sessions_updated_at ON payment_sessions;
DROP TRIGGER IF EXISTS update_invoices_updated_at ON invoices;
DROP TRIGGER IF EXISTS update_gateway_configs_updated_at ON gateway_configs;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS payment_sessions;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS gateway_configs;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organizations;

DROP EXTENSION IF EXISTS "pgcrypto";
