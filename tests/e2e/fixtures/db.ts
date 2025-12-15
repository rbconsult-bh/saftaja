import { Client } from 'pg';
import { TEST_IDS } from './config';

export async function seedDb() {
  const connectionString = process.env.DATABASE_URL;
  if (!connectionString) {
    throw new Error('DATABASE_URL not set');
  }

  const client = new Client({ connectionString });
  await client.connect();

  await client.query(`TRUNCATE gateway_accounts, invoices, invoice_items, transactions, payment_sessions CASCADE`);

  await client.query(`
    INSERT INTO gateway_accounts (
      id, project_id, connector_type, account_name, credentials, settings, payment_methods, is_active
    ) VALUES (
      $1, $2, 'mpgs', 'Test MPGS Account',
      $3, '{"display_name": "Credit Card"}', '["card"]', true
    )
  `, [
    TEST_IDS.GATEWAY_ACCOUNT,
    TEST_IDS.PROJECT,
    JSON.stringify({
      merchant_id: process.env.MPGS_MERCHANT_ID,
      api_password: process.env.MPGS_API_PASSWORD,
      base_url: process.env.MPGS_BASE_URL,
    })
  ]);

  await client.query(`
    INSERT INTO invoices (id, project_id, amount, currency, status, customer_name, customer_email, description)
    VALUES ($1, $2, 15.000, 'BHD', 'pending', 'Test User', 'test@example.com', 'Test Invoice')
  `, [TEST_IDS.INVOICE_PENDING, TEST_IDS.PROJECT]);

  await client.query(`
    INSERT INTO invoices (id, project_id, amount, currency, status, customer_name, customer_email, description)
    VALUES ($1, $2, 25.000, 'BHD', 'paid', 'Test User', 'test@example.com', 'Paid Invoice')
  `, [TEST_IDS.INVOICE_PAID, TEST_IDS.PROJECT]);

  await client.end();
}

export async function resetDb() {
  await seedDb();
}
