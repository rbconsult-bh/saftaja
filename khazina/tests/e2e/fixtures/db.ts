import { Client } from 'pg';
import crypto from 'crypto';

export interface SeedIds {
  ORGANIZATION: string;
  PROJECT: string;
  GATEWAY_ACCOUNT: string;
  INVOICE_PENDING: string;
  INVOICE_PAID: string;
}

// TODO: Switch to ConnectRPC dashboard API for seeding once CreateInvoice endpoint exists.
// SQL seeding is fine for now since invoices have no API yet.

function encryptAES256GCM(plaintext: string, keyBase64: string): Buffer {
  const key = Buffer.from(keyBase64, 'base64');
  const nonce = crypto.randomBytes(12);
  const cipher = crypto.createCipheriv('aes-256-gcm', key, nonce);
  const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()]);
  const authTag = cipher.getAuthTag();
  // Go layout: nonce(12) || ciphertext || authTag(16)
  return Buffer.concat([nonce, encrypted, authTag]);
}

export async function seedDb(): Promise<SeedIds> {
  const connectionString = process.env.KHAZINA_TEST_DATABASE_URL;
  if (!connectionString) throw new Error('KHAZINA_TEST_DATABASE_URL not set (global-setup must run first)');

  const publicUrl = process.env.KHAZINA_TEST_BASE_URL;
  if (!publicUrl) throw new Error('KHAZINA_TEST_BASE_URL not set (global-setup must run first)');

  const domain = new URL(publicUrl).hostname;

  const mpgsMerchantId = process.env.KHAZINA_TEST_MPGS_MERCHANT_ID;
  const mpgsApiPassword = process.env.KHAZINA_TEST_MPGS_API_PASSWORD;
  const mpgsBaseUrl = process.env.KHAZINA_TEST_MPGS_BASE_URL;

  if (!mpgsMerchantId || !mpgsApiPassword || !mpgsBaseUrl) {
    throw new Error('TEST_MPGS_MERCHANT_ID, TEST_MPGS_API_PASSWORD, TEST_MPGS_BASE_URL must be set in tests/e2e/.env');
  }

  const encryptionKey = process.env.KHAZINA_TEST_ENCRYPTION_KEY;
  if (!encryptionKey) throw new Error('KHAZINA_TEST_ENCRYPTION_KEY not set (global-setup must run first)');

  const configJSON = JSON.stringify({
    merchant_id: mpgsMerchantId,
    base_url: mpgsBaseUrl,
  });
  const secretJSON = JSON.stringify({
    api_password: mpgsApiPassword,
  });
  const encryptedSecret = encryptAES256GCM(secretJSON, encryptionKey);

  const client = new Client({ connectionString });
  await client.connect();

  try {
    await client.query(`
      TRUNCATE gateway_accounts, invoices, invoice_items, transactions, payment_intents, organizations, projects CASCADE
    `);

    const orgResult = await client.query(`
      INSERT INTO organizations (name) VALUES ('Test Org') RETURNING id
    `);
    const orgId = orgResult.rows[0].id;

    const projectResult = await client.query(`
      INSERT INTO projects (organization_id, name, environment, custom_domain) VALUES ($1, 'Sandbox', 'sandbox', $2) RETURNING id
    `, [orgId, domain]);
    const projectId = projectResult.rows[0].id;

    const gwResult = await client.query(`
      INSERT INTO gateway_accounts (project_id, connector_type, account_name, secret, config, is_active)
      VALUES ($1, 'mpgs', 'Test MPGS', $2, $3, true) RETURNING id
    `, [projectId, encryptedSecret, configJSON]);
    const gatewayAccountId = gwResult.rows[0].id;

    const pendingResult = await client.query(`
      INSERT INTO invoices (project_id, amount, currency, customer_email, customer_name, description, status)
      VALUES ($1, '15.000', 'BHD', 'test@example.com', 'Test User', 'Test Invoice', 'pending') RETURNING id
    `, [projectId]);
    const invoicePendingId = pendingResult.rows[0].id;

    await client.query(`
      INSERT INTO invoice_items (invoice_id, name, description, quantity, unit_price, amount)
      VALUES ($1, 'Test Item', 'A test line item', 1, '15.000', '15.000')
    `, [invoicePendingId]);

    const paidResult = await client.query(`
      INSERT INTO invoices (project_id, amount, currency, customer_email, customer_name, description, status, paid_at)
      VALUES ($1, '25.000', 'BHD', 'test@example.com', 'Test User', 'Paid Invoice', 'paid', NOW()) RETURNING id
    `, [projectId]);
    const invoicePaidId = paidResult.rows[0].id;

    await client.query(`
      INSERT INTO invoice_items (invoice_id, name, description, quantity, unit_price, amount)
      VALUES ($1, 'Paid Item', 'A paid line item', 1, '25.000', '25.000')
    `, [invoicePaidId]);

    return {
      ORGANIZATION: orgId,
      PROJECT: projectId,
      GATEWAY_ACCOUNT: gatewayAccountId,
      INVOICE_PENDING: invoicePendingId,
      INVOICE_PAID: invoicePaidId,
    };
  } finally {
    await client.end();
  }
}

export async function resetDb(): Promise<SeedIds> {
  return seedDb();
}

export async function dumpDb(): Promise<void> {
  const connectionString = process.env.KHAZINA_TEST_DATABASE_URL;
  if (!connectionString) return;

  const client = new Client({ connectionString });
  await client.connect();

  try {
    const tables = ['organizations', 'projects', 'gateway_accounts', 'invoices', 'payment_intents', 'transactions'];
    for (const table of tables) {
      const result = await client.query(`SELECT * FROM ${table}`);
      if (result.rows.length > 0) {
        console.log(`\n📋 ${table} (${result.rows.length} rows):`);
        for (const row of result.rows) {
          const safe = { ...row };
          if ('secret' in safe) safe.secret = '[encrypted]';
          console.log(JSON.stringify(safe, null, 2));
        }
      }
    }
  } finally {
    await client.end();
  }
}
