import { test, expect } from '@playwright/test';

test.describe('Admin Control Panel Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');

    // Sign in as Merchant Admin
    await page.getByTestId('login-button').click();
    await page.getByTestId('demo-admin-login').click();
    await expect(page.getByTestId('admin-tab-dashboard')).toBeVisible();
  });

  test('1. Should view Executive Dashboard Analytics and KPIs in IDR / Rp', async ({ page }) => {
    await expect(page.locator('text=Merchant Executive Dashboard')).toBeVisible();
    await expect(page.locator('text=TOTAL REVENUE')).toBeVisible();
    await expect(page.locator('text=TOTAL ORDERS')).toBeVisible();
    await expect(page.locator('text=7-Day Sales Velocity')).toBeVisible();
    // Verify Rupiah formatting
    await expect(page.locator('text=Rp').first()).toBeVisible();
  });

  test('2. Should navigate Products catalog and view Stock Management', async ({ page }) => {
    await page.getByTestId('admin-tab-products').click();
    await expect(page.locator('text=Product & Stock Management')).toBeVisible();

    const tableRows = page.locator('tbody tr');
    await expect(tableRows.first()).toBeVisible();
    const count = await tableRows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('3. Should view Customer Orders and CRM directory', async ({ page }) => {
    // Navigate to Orders
    await page.getByTestId('admin-tab-orders').click();
    await expect(page.locator('text=Order Fulfillment & Tracking')).toBeVisible();

    // Navigate to Customers CRM
    await page.getByTestId('admin-tab-customers').click();
    await expect(page.locator('text=Customer CRM Directory')).toBeVisible();
  });

  test('4. Should navigate to SOP & AI Knowledge Base and view indexed documents', async ({ page }) => {
    await page.getByTestId('admin-tab-knowledge').click();
    await expect(page.locator('text=/Knowledge Base|Basis Pengetahuan/i')).toBeVisible();
    await expect(page.locator('text=/Upload|Unggah/i').first()).toBeVisible();
    await expect(page.locator('text=/Playground|Uji Coba/i')).toBeVisible();
    await expect(page.locator('text=/Indexed|Terindeks/i').first()).toBeVisible();
  });
});
