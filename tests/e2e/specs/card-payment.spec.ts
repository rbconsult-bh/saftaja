import { test, expect } from '@playwright/test';
import { seedDb, SeedIds } from '../fixtures/db';

test.describe('Card Payment Flow', () => {
  let ids: SeedIds;
  test.beforeEach(async () => {
    ids = await seedDb();
  });

  test('shows checkout page for pending invoice', async ({ page }) => {
    await page.goto(`/checkout/${ids.INVOICE_PENDING}`);
    await expect(page.locator('text=Payment Method')).toBeVisible();
  });

  test('shows success page for paid invoice', async ({ page }) => {
    await page.goto(`/checkout/${ids.INVOICE_PAID}`);
    await expect(page.locator('text=Payment Successful')).toBeVisible();
  });

  test('show success page after paying for pending invoice', async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER: ${msg.text()}`));

    await page.goto(`/checkout/${ids.INVOICE_PENDING}`);

    await page.locator('button[data-type="card"]').click();
    await page.waitForSelector('#card-form-container:not(.hidden)', { timeout: 15000 });

    await page.frameLocator('iframe[id="#card-number"]').getByRole('textbox').fill('5123450000000008');
    await page.frameLocator('iframe[id="#expiry-month"]').getByRole('textbox').fill('01');
    await page.frameLocator('iframe[id="#expiry-year"]').getByRole('textbox').fill('39');
    await page.frameLocator('iframe[id="#security-code"]').getByRole('textbox').fill('100');
    await page.frameLocator('iframe[id="#cardholder-name"]').getByRole('textbox').fill('HUMAN BEING');

    await page.getByRole('button', { name: /Pay.*BHD/ }).click();

    await page.waitForSelector('#challenge-overlay:not(.hidden)', { timeout: 30000 });

    const challengeFrame = page.frameLocator('#challenge-iframe-container iframe');
    await challengeFrame.getByText(/ACS Emulator/i).waitFor({ state: 'visible', timeout: 30000 });

    const submitBtn = challengeFrame.getByRole('button', { name: 'Submit' });
    await submitBtn.waitFor({ state: 'visible' });
    await submitBtn.click();

    await expect(page.getByRole('heading', { name: 'Payment Successful!' })).toBeVisible({ timeout: 30000 });
  });
});

