import { execSync } from 'child_process';
import crypto from 'crypto';

export const DB_CONFIG = {
  host: 'db',
  port: 5432,
  database: 'paydb_test',
  user: 'test',
  password: 'test',
} as const;

export const TUNNEL_TIMEOUT_MS = 30000;
export const HEALTH_CHECK_TIMEOUT_MS = 60000;

export function generateTestEncryptionKey(): string {
  const key = crypto.randomBytes(32);
  return key.toString('base64');
}

export function generateTestJWTKey(): string {
  const key = execSync('openssl genrsa 2048 2>/dev/null', { encoding: 'utf-8' });
  return Buffer.from(key).toString('base64');
}

export function generateTestSecret(): string {
  return crypto.randomBytes(32).toString('hex');
}
