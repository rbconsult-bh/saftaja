export const DB_CONFIG = {
  host: 'db',
  port: 5432,
  database: 'paydb_test',
  user: 'test',
  password: 'test',
} as const;

export const TEST_IDS = {
  PROJECT: '00000000-0000-0000-0000-000000000002',
  GATEWAY_ACCOUNT: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  INVOICE_PENDING: '11111111-1111-1111-1111-111111111111',
  INVOICE_PAID: '22222222-2222-2222-2222-222222222222',
} as const;
