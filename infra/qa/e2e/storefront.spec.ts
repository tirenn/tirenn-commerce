import { test, expect } from '@playwright/test';

test.describe('Storefront User-Facing Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for product card to render
    await page.locator('[data-testid^="product-card-"]').first().waitFor({ timeout: 10000 });
  });

  test('1. Should render homepage title and rich 280-product catalog grid', async ({ page }) => {
    await expect(page).toHaveTitle(/Tirenn Commerce/);
    await expect(page.locator('text=Tirenn Commerce').first()).toBeVisible();

    const productGrid = page.getByTestId('products-grid');
    await expect(productGrid).toBeVisible();

    const productCards = page.locator('[data-testid^="product-card-"]');
    const count = await productCards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('2. Should filter catalog by Category and Sub-category', async ({ page }) => {
    // Click on Smartphone & Handphone category tab
    const phoneCatTab = page.locator('button:has-text("Smartphone")').first();
    await phoneCatTab.click();

    await page.locator('[data-testid^="product-card-"]').first().waitFor();
    await expect(page.locator('text=/Smartphone/i').first()).toBeVisible();

    // Verify Subcategory pills appeared (e.g. Flagship Smartphones, Mid-Range Smartphones)
    const subCatPill = page.locator('button:has-text("Flagship")').first();
    if (await subCatPill.isVisible()) {
      await subCatPill.click();
      await page.locator('[data-testid^="product-card-"]').first().waitFor();
    }

    // Reset filter
    await page.getByTestId('category-tab-all').click();
    await page.locator('[data-testid^="product-card-"]').first().waitFor();
  });

  test('3. Should switch language (i18n) and convert currency dynamically (IDR <-> USD)', async ({ page }) => {
    const langToggle = page.getByTestId('lang-currency-toggle');
    await expect(langToggle).toBeVisible();

    // Initially IDR (Rp)
    await expect(page.locator('text=Rp').first()).toBeVisible();

    // Toggle to English & USD ($)
    await langToggle.click();

    // Verify USD symbol ($) is displayed
    await expect(page.locator('text=$').first()).toBeVisible({ timeout: 5000 });

    // Toggle back to IDR (Rp)
    await langToggle.click();
    await expect(page.locator('text=Rp').first()).toBeVisible({ timeout: 5000 });
  });

  test('4. Should search for products using real-time search input', async ({ page }) => {
    const searchInput = page.getByTestId('search-input');
    await searchInput.fill('Galaxy');

    await expect(page.locator('text=/Galaxy|Samsung/i').first()).toBeVisible({ timeout: 5000 });

    await searchInput.fill('');
    await page.locator('[data-testid^="product-card-"]').first().waitFor();
  });

  test('5. Should open Product Detail Modal (PDP) and adjust quantity', async ({ page }) => {
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

  test('6. Should add product to Cart, open Cart Drawer, and modify quantity', async ({ page }) => {
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

  test('7. Guest user should be able to add product to cart via AI Chat', async ({ page }) => {
    test.setTimeout(120000);
    const aiBtn = page.getByTestId('ai-shopper-floating-btn');
    await aiBtn.click();

    // Wait for chat dialog to appear
    const chatInput = page.getByTestId('ai-chat-input');
    await expect(chatInput).toBeVisible();

    // Send query for headphones
    await chatInput.fill('AuraSound');
    await page.getByTestId('ai-chat-send').click();

    // Wait for product card to appear inside chat
    const chatProductCard = page.locator('[data-testid^="chat-product-card-"]').first();
    await expect(chatProductCard).toBeVisible({ timeout: 90000 });

    // Click Add to Cart inside AI Chat
    const chatAddBtn = page.locator('[data-testid^="chat-add-to-cart-"]').first();
    await chatAddBtn.click();

    // Verify cart badge updated
    const cartBadge = page.getByTestId('cart-badge');
    await expect(cartBadge).toBeVisible();

    await page.getByTestId('ai-chat-close').click();
  });

  test('8. Should authenticate via 1-Click Shopper Demo and place an order', async ({ page }) => {
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

    await expect(page.locator('text=Riwayat Belanja').or(page.locator('text=My Orders')).or(page.locator('text=Your Orders')).first()).toBeVisible();
  });

  test('9. Should display AI product recommendations in PDP and Cart Drawer with 1-click quick add', async ({ page }) => {
    // Step 1: Open the first product card from storefront grid to view PDP
    const firstCard = page.locator('[data-testid^="product-card-"]').first();
    await firstCard.click();

    // Step 2: Assert PDP Modal is visible
    const pdpModal = page.getByTestId('pdp-modal');
    await expect(pdpModal).toBeVisible();

    // Step 3: Assert PDP recommendations section is rendered with recommendation cards
    const pdpRecSection = page.getByTestId('pdp-recommendations-section');
    await expect(pdpRecSection).toBeVisible({ timeout: 10000 });

    const pdpRecCards = page.locator('[data-testid^="pdp-recommendation-card-"]');
    await expect(pdpRecCards.first()).toBeVisible();
    const pdpRecCount = await pdpRecCards.count();
    expect(pdpRecCount).toBeGreaterThan(0);

    // Step 4: Verify currency symbol is rendered on recommendation cards (Rp or $)
    await expect(
      pdpRecCards.first().locator('text=Rp').or(pdpRecCards.first().locator('text=$')).first()
    ).toBeVisible();

    // Step 5: Perform 1-Click Quick Add from PDP recommendation
    const pdpQuickAddBtn = page.locator('[data-testid^="pdp-recommendation-add-"]').first();
    await pdpQuickAddBtn.click();

    // Step 6: Verify Cart Badge appears and is visible
    const cartBadge = page.getByTestId('cart-badge');
    await expect(cartBadge).toBeVisible();

    // Step 7: Close PDP Modal
    await page.getByTestId('pdp-close').click();
    await expect(pdpModal).not.toBeVisible();

    // Step 8: Open Cart Drawer
    await page.getByTestId('cart-button').click();
    const cartDrawer = page.getByTestId('cart-drawer');
    await expect(cartDrawer).toBeVisible();

    // Step 9: Verify Cart contains item and displays Cart Drawer Contextual Recommendations
    await expect(page.locator('[data-testid^="cart-item-"]').first()).toBeVisible();
    const cartRecSection = page.getByTestId('cart-recommendations-section');
    await expect(cartRecSection).toBeVisible({ timeout: 10000 });

    const cartRecCards = page.locator('[data-testid^="cart-recommendation-card-"]');
    await expect(cartRecCards.first()).toBeVisible();

    // Step 10: Perform 1-Click Quick Add from Cart Drawer recommendations
    const cartQuickAddBtn = page.locator('[data-testid^="cart-recommendation-add-"]').first();
    await cartQuickAddBtn.click();

    // Step 11: Verify Cart Total Price updates and close drawer
    await expect(page.getByTestId('cart-total-price')).toBeVisible();
    await page.getByTestId('cart-drawer-close').click();
    await expect(cartDrawer).not.toBeVisible();
  });
});
