import { test, expect } from '@playwright/test';

test.describe('Storefront User-Facing Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for product card to render
    await page.locator('[data-testid^="product-card-"]').first().waitFor({ timeout: 10000 });
  });

  test('1. Should render homepage title and product catalog grid', async ({ page }) => {
    await expect(page).toHaveTitle(/Tirenn Commerce/);
    await expect(page.locator('text=Tirenn Commerce').first()).toBeVisible();

    const productGrid = page.getByTestId('products-grid');
    await expect(productGrid).toBeVisible();

    const productCards = page.locator('[data-testid^="product-card-"]');
    const count = await productCards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('2. Should filter catalog by department category', async ({ page }) => {
    const electronicsTab = page.locator('button:has-text("Electronics")').first();
    await electronicsTab.click();

    await page.locator('[data-testid^="product-card-"]').first().waitFor();
    await expect(page.locator('text=Electronics').first()).toBeVisible();

    // Reset filter
    await page.getByTestId('category-tab-all').click();
    await page.locator('[data-testid^="product-card-"]').first().waitFor();
  });

  test('3. Should search for products using real-time search input', async ({ page }) => {
    const searchInput = page.getByTestId('search-input');
    await searchInput.fill('Headphones');

    await expect(page.locator('text=AuraPro').first()).toBeVisible({ timeout: 5000 });

    await searchInput.fill('');
    await page.locator('[data-testid^="product-card-"]').first().waitFor();
  });

  test('4. Should open Product Detail Modal (PDP) and adjust quantity', async ({ page }) => {
    const firstCard = page.locator('[data-testid^="product-card-"]').first();
    await firstCard.click();

    const modal = page.getByTestId('pdp-modal');
    await expect(modal).toBeVisible();

    // Increment quantity inside modal
    await modal.getByRole('button', { name: '+', exact: true }).click();

    // Close modal
    await page.getByTestId('pdp-close').click();
    await expect(modal).not.toBeVisible();
  });

  test('5. Should add product to Cart, open Cart Drawer, and modify quantity', async ({ page }) => {
    const addBtn = page.locator('[data-testid^="add-to-cart-"]').first();
    await addBtn.click();

    const cartBadge = page.getByTestId('cart-badge');
    await expect(cartBadge).toBeVisible();

    await page.getByTestId('cart-button').click();
    const cartDrawer = page.getByTestId('cart-drawer');
    await expect(cartDrawer).toBeVisible();

    await expect(page.locator('[data-testid^="cart-item-"]').first()).toBeVisible();
    await expect(page.getByTestId('cart-total-price')).toBeVisible();

    const incBtn = page.locator('[data-testid^="cart-increment-"]').first();
    await incBtn.click();

    await page.getByTestId('cart-drawer-close').click();
    await expect(cartDrawer).not.toBeVisible();
  });

  test('6. Should authenticate via 1-Click Shopper Demo and place an order', async ({ page }) => {
    await page.getByTestId('login-button').click();
    const authModal = page.getByTestId('auth-modal');
    await expect(authModal).toBeVisible();

    await page.getByTestId('demo-shopper-login').click();
    await expect(authModal).not.toBeVisible();
    await expect(page.getByTestId('logout-button')).toBeVisible();

    await page.locator('[data-testid^="add-to-cart-"]').first().click();

    await page.getByTestId('cart-button').click();
    await page.getByTestId('cart-checkout-button').click();

    const checkoutModal = page.getByTestId('checkout-modal');
    await expect(checkoutModal).toBeVisible();

    await page.getByTestId('checkout-submit').click();
    await expect(checkoutModal).not.toBeVisible({ timeout: 10000 });

    await expect(page.locator('text=Your Orders')).toBeVisible();
    await expect(page.locator('text=Order #').first()).toBeVisible();
  });
});
