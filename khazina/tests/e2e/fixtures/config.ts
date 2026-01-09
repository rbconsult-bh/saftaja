export const DB_CONFIG = {
  host: 'db',
  port: 5432,
  database: 'paydb_test',
  user: 'test',
  password: 'test',
} as const;

export const TUNNEL_TIMEOUT_MS = 30000;
export const HEALTH_CHECK_TIMEOUT_MS = 60000;

export const TEST_ENCRYPTION_KEY_BASE64 = 'YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=';
export const TEST_VERIFY_DOMAIN_SECRET = 'test-verify-secret';
export const TEST_ADMIN_API_KEY = 'test-admin-key';
