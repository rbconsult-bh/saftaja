import { Client } from 'pg';
import { randomUUID } from 'crypto';
import { encryptJSON } from './crypto';
import { TEST_ENCRYPTION_KEY } from './config';

export interface SeedIds {
  ORGANIZATION: string;
  PROJECT: string;
  GATEWAY_ACCOUNT: string;
  INVOICE_PENDING: string;
  INVOICE_PAID: string;
}

export async function seedDb(): Promise<SeedIds> {
  const connectionString = process.env.DATABASE_URL;
  if (!connectionString) {
    throw new Error('DATABASE_URL not set');
  }
  const publicUrl = process.env.BASE_URL;
  if (!publicUrl) throw new Error('BASE_URL not set');
  const domain = new URL(publicUrl).hostname;

  const ids: SeedIds = {
    ORGANIZATION: randomUUID(),
    PROJECT: randomUUID(),
    GATEWAY_ACCOUNT: randomUUID(),
    INVOICE_PENDING: randomUUID(),
    INVOICE_PAID: randomUUID(),
  };

  const client = new Client({ connectionString });
  await client.connect();

  try {
    await client.query(`
      TRUNCATE gateway_accounts, invoices, invoice_items, transactions, payment_sessions, organizations, projects CASCADE
    `);

    await client.query(`
INSERT INTO organizations (id, name)
VALUES ($1, 'Default');`,
      [
        ids.ORGANIZATION
      ]);

    await client.query(`
INSERT INTO projects (id, organization_id, name, environment, custom_domain)
VALUES (
    $1,
    $2,
    'Sandbox',
    'sandbox',
    $3
);`, [ids.PROJECT, ids.ORGANIZATION, domain]);

    const encryptedCredentials = encryptJSON(
      {
        merchant_id: process.env.TEST_MPGS_MERCHANT_ID,
        api_password: process.env.TEST_MPGS_API_PASSWORD,
        base_url: process.env.TEST_MPGS_BASE_URL,
      },
      TEST_ENCRYPTION_KEY
    );

    await client.query(
      `INSERT INTO gateway_accounts (
        id, project_id, connector_type, account_name, credentials, settings, payment_methods, is_active
      ) VALUES ($1, $2, 'mpgs', 'Test MPGS Account', $3, '{"display_name": "Credit Card"}', '["card"]', true)`,
      [ids.GATEWAY_ACCOUNT, ids.PROJECT, encryptedCredentials]
    );

    await client.query(
      `INSERT INTO invoices (id, project_id, amount, currency, status, customer_name, customer_email, description)
       VALUES ($1, $2, 15.000, 'BHD', 'pending', 'Test User', 'test@example.com', 'Test Invoice')`,
      [ids.INVOICE_PENDING, ids.PROJECT]
    );

    await client.query(
      `INSERT INTO invoices (id, project_id, amount, currency, status, customer_name, customer_email, description)
       VALUES ($1, $2, 25.000, 'BHD', 'paid', 'Test User', 'test@example.com', 'Paid Invoice')`,
      [ids.INVOICE_PAID, ids.PROJECT]
    );
  } finally {
    await client.end();
  }

  return ids;
}

export async function resetDb(): Promise<SeedIds> {
  return seedDb();
}
