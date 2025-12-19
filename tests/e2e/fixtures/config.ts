export const DB_CONFIG = {
  host: 'db',
  port: 5432,
  database: 'paydb_test',
  user: 'test',
  password: 'test',
} as const;

export const TUNNEL_TIMEOUT_MS = 30000;
export const HEALTH_CHECK_TIMEOUT_MS = 60000;
