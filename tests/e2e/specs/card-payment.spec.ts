import { test, expect } from '@playwright/test';
import { seedDb } from '../fixtures/db';
import { TEST_IDS } from '../fixtures/config';

test.beforeEach(async () => {
  await seedDb();
});

test('shows checkout page for pending invoice', async ({ page }) => {
  await page.goto(`/checkout/${TEST_IDS.INVOICE_PENDING}`);
  await expect(page.locator('text=Payment Method')).toBeVisible();
});

test('shows success page for paid invoice', async ({ page }) => {
  await page.goto(`/checkout/${TEST_IDS.INVOICE_PAID}`);
  await expect(page.locator('text=Payment Successful')).toBeVisible();
});
