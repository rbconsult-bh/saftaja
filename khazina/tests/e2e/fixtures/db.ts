import { Client } from 'pg';
import { TEST_ADMIN_API_KEY } from './config';

export interface SeedIds {
  ORGANIZATION: string;
  PROJECT: string;
  GATEWAY_ACCOUNT: string;
  INVOICE_PENDING: string;
  INVOICE_PAID: string;
}

async function adminFetch(path: string, body: object): Promise<any> {
  const baseUrl = process.env.BASE_URL;
  if (!baseUrl) throw new Error('BASE_URL not set');

  const resp = await fetch(`${baseUrl}/admin${path}`, {
    method: 'POST',
    headers: {
      'X-Admin-Key': TEST_ADMIN_API_KEY,
      'Content-Type': 'application/json',
      'bypass-tunnel-reminder': 'yes'
    },
    body: JSON.stringify(body)
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Admin API ${path} failed: ${resp.status} ${text}`);
  }

  return resp.json();
}

export async function seedDb(): Promise<SeedIds> {
  const connectionString = process.env.DATABASE_URL;
  if (!connectionString) throw new Error('DATABASE_URL not set');

  const publicUrl = process.env.BASE_URL;
  if (!publicUrl) throw new Error('BASE_URL not set');
  const domain = new URL(publicUrl).hostname;

  const client = new Client({ connectionString });
  await client.connect();

  try {
    await client.query(`
      TRUNCATE gateway_accounts, invoices, invoice_items, transactions, payment_sessions, organizations, projects CASCADE
    `);

    const org = await adminFetch('/organizations', { name: 'Test Org' });

    const project = await adminFetch('/projects', {
      organization_id: org.id,
      name: 'Sandbox',
      environment: 'sandbox',
      custom_domain: domain
    });

    const gateway = await adminFetch('/gateway-accounts', {
      project_id: project.id,
      connector_type: 'mpgs',
      account_name: 'Test MPGS',
      credentials: {
        merchant_id: process.env.TEST_MPGS_MERCHANT_ID,
        api_password: process.env.TEST_MPGS_API_PASSWORD,
        base_url: process.env.TEST_MPGS_BASE_URL
      },
      payment_methods: ['card']
    });

    const pendingInvoice = await adminFetch('/invoices', {
      project_id: project.id,
      amount: '15.000',
      currency: 'BHD',
      customer_email: 'test@example.com',
      customer_name: 'Test User',
      description: 'Test Invoice'
    });

    const paidInvoice = await adminFetch('/invoices', {
      project_id: project.id,
      amount: '25.000',
      currency: 'BHD',
      customer_email: 'test@example.com',
      customer_name: 'Test User',
      description: 'Paid Invoice'
    });

    await client.query(`UPDATE invoices SET status = 'paid' WHERE id = $1`, [paidInvoice.id]);

    return {
      ORGANIZATION: org.id,
      PROJECT: project.id,
      GATEWAY_ACCOUNT: gateway.id,
      INVOICE_PENDING: pendingInvoice.id,
      INVOICE_PAID: paidInvoice.id,
    };
  } finally {
    await client.end();
  }
}

export async function resetDb(): Promise<SeedIds> {
  return seedDb();
}
