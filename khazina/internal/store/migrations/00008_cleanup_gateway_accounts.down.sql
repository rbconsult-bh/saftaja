ALTER TABLE gateway_accounts ADD COLUMN payment_methods JSONB NOT NULL DEFAULT '[]';

ALTER TABLE gateway_accounts RENAME COLUMN config TO settings;
ALTER TABLE gateway_accounts RENAME COLUMN secret TO credentials;
