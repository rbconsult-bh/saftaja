-- Rename columns for clarity:
--   config   = public info (base_url, merchant_id)
--   secret   = encrypted private info (api_password)
ALTER TABLE gateway_accounts RENAME COLUMN settings TO config;
ALTER TABLE gateway_accounts RENAME COLUMN credentials TO secret;

-- Drop hardcoded payment_methods.
-- Payment methods are discovered dynamically by connector type:
--   mpgs     → MPGSA Payment Options Inquiry API
--   future   → connector-specific discovery
ALTER TABLE gateway_accounts DROP COLUMN payment_methods;
