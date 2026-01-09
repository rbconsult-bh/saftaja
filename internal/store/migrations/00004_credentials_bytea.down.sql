ALTER TABLE gateway_accounts
DROP COLUMN credentials;

ALTER TABLE gateway_accounts
ADD COLUMN credentials JSONB NOT NULL DEFAULT '{}';
