import { test, expect } from '@playwright/test';

test.describe('Adversarial Storefront & Cart Challenge Suite', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.locator('[data-testid^="product-card-"]').first().waitFor({ timeout: 10000 });
  });

  test('ADV-01: Modal switching vs Quick-add stopPropagation check', async ({ page }) => {
    // 1. Open first product card (ID 1) to open PDP modal
    const firstCard = page.locator('[data-testid^="product-card-"]').first();
    await firstCard.click();

    const pdpModal = page.getByTestId('pdp-modal');
    await expect(pdpModal).toBeVisible();

    // 2. Wait for recommendations in PDP
    const pdpRecCards = page.locator('[data-testid^="pdp-recommendation-card-"]');
    await expect(pdpRecCards.first()).toBeVisible({ timeout: 10000 });

    // Grab first recommendation product name/heading before quick-add
    const firstRecCard = pdpRecCards.first();
    const firstRecTitle = await firstRecCard.locator('h4').textContent();

    // Grab current main product heading in modal
    const currentHeading = await pdpModal.locator('h2').textContent();

    // 3. Click the Quick-Add (+) button inside the recommendation card
    const quickAddBtn = firstRecCard.locator('[data-testid^="pdp-recommendation-add-"]');
    await quickAddBtn.click();

    // 4. Assert: Main product modal did NOT change to the recommendation product (stopPropagation prevented modal switch)
    const headingAfterQuickAdd = await pdpModal.locator('h2').textContent();
    expect(headingAfterQuickAdd).toBe(currentHeading);

    // 5. Assert: Cart badge updated to at least 1
    const cartBadge = page.getByTestId('cart-badge');
    await expect(cartBadge).toBeVisible();

    // 6. Now click on the recommendation card body (NOT the quick add button)
    await firstRecCard.click();

    // 7. Assert: Main product modal DOES switch to the recommended product
    await expect(pdpModal.locator('h2')).toHaveText(firstRecTitle!.trim());

    // 8. Close modal
    await page.getByTestId('pdp-close').click();
    await expect(pdpModal).not.toBeVisible();
  });

  test('ADV-02: CartDrawer empty vs non-empty recommendation behavior and filtering', async ({ page }) => {
    // 1. Open empty cart drawer
    await page.getByTestId('cart-button').click();
    const cartDrawer = page.getByTestId('cart-drawer');
    await expect(cartDrawer).toBeVisible();

    // 2. Verify empty cart message is displayed
    await expect(cartDrawer.locator('text=🛒')).toBeVisible();

    // 3. Verify recommendations section is NOT rendered when cart is empty
    const cartRecSection = page.getByTestId('cart-recommendations-section');
    await expect(cartRecSection).not.toBeVisible();

    // 4. Close cart drawer
    await page.getByTestId('cart-drawer-close').click();
    await expect(cartDrawer).not.toBeVisible();

    // 5. Add product 1 to cart
    const firstAddBtn = page.locator('[data-testid^="add-to-cart-"]').first();
    await firstAddBtn.click();

    // 6. Reopen cart drawer
    await page.getByTestId('cart-button').click();
    await expect(cartDrawer).toBeVisible();

    // 7. Verify recommendations section is NOW rendered
    await expect(cartRecSection).toBeVisible({ timeout: 10000 });
    const recCards = page.locator('[data-testid^="cart-recommendation-card-"]');
    await expect(recCards.first()).toBeVisible();

    // 8. Grab the ID of the first recommendation card and click quick add
    const firstRecCard = recCards.first();
    const recTestId = await firstRecCard.getAttribute('data-testid');
    const recId = recTestId?.replace('cart-recommendation-card-', '');
    expect(recId).toBeTruthy();

    const recAddBtn = firstRecCard.locator('[data-testid^="cart-recommendation-add-"]');
    await recAddBtn.click();

    // 9. Assert: The added recommendation product is now in the cart items list
    await expect(page.getByTestId(`cart-item-${recId}`)).toBeVisible();

    // 10. Assert: The added recommendation product is filtered out from recommendations list
    await expect(page.getByTestId(`cart-recommendation-card-${recId}`)).not.toBeVisible();

    // 11. Close drawer
    await page.getByTestId('cart-drawer-close').click();
  });

  test('ADV-03: Rapid multi-click quick-add race condition and cart total consistency', async ({ page }) => {
    // 1. Rapidly click quick-add button 5 times in quick succession
    const addBtn = page.locator('[data-testid^="add-to-cart-"]').first();
    
    // Multi-click 5 times rapidly
    await Promise.all([
      addBtn.click(),
      addBtn.click(),
      addBtn.click(),
      addBtn.click(),
      addBtn.click(),
    ]);

    // 2. Check cart badge
    const cartBadge = page.getByTestId('cart-badge');
    await expect(cartBadge).toBeVisible();
    const badgeText = await cartBadge.textContent();
    // Badge count should be >= 5 (or max stock)
    expect(Number(badgeText)).toBeGreaterThanOrEqual(5);

    // 3. Open cart drawer
    await page.getByTestId('cart-button').click();
    const cartDrawer = page.getByTestId('cart-drawer');
    await expect(cartDrawer).toBeVisible();

    // 4. Verify item quantity in drawer matches badge count
    const itemQty = page.locator('[data-testid^="cart-quantity-"]').first();
    await expect(itemQty).toHaveText(badgeText!.trim());

    // 5. Verify cart total is visible and formatted properly
    const totalPrice = page.getByTestId('cart-total-price');
    await expect(totalPrice).toBeVisible();

    await page.getByTestId('cart-drawer-close').click();
  });

  test('ADV-04: Dynamic currency conversion across PDP and Cart Drawer recommendation cards', async ({ page }) => {
    // 1. Verify default currency is IDR (Rp)
    const langToggle = page.getByTestId('lang-currency-toggle');
    await expect(langToggle).toBeVisible();
    await expect(page.locator('text=Rp').first()).toBeVisible();

    // 2. Open PDP Modal
    const firstCard = page.locator('[data-testid^="product-card-"]').first();
    await firstCard.click();
    const pdpModal = page.getByTestId('pdp-modal');
    await expect(pdpModal).toBeVisible();

    // Wait for PDP recommendations
    const pdpRecCards = page.locator('[data-testid^="pdp-recommendation-card-"]');
    await expect(pdpRecCards.first()).toBeVisible({ timeout: 10000 });

    // Assert recommendation card has Rp price
    await expect(pdpRecCards.first().locator('text=Rp')).toBeVisible();

    // Close PDP Modal
    await page.getByTestId('pdp-close').click();

    // 3. Switch currency to USD ($)
    await langToggle.click();
    await expect(page.locator('text=$').first()).toBeVisible({ timeout: 5000 });

    // 4. Re-open PDP Modal and verify recommendation card price is now in USD ($)
    await firstCard.click();
    await expect(pdpModal).toBeVisible();
    await expect(pdpRecCards.first()).toBeVisible({ timeout: 10000 });
    await expect(pdpRecCards.first().locator('text=$')).toBeVisible();

    // Add recommendation to cart
    const pdpQuickAddBtn = pdpRecCards.first().locator('[data-testid^="pdp-recommendation-add-"]');
    await pdpQuickAddBtn.click();
    await page.getByTestId('pdp-close').click();

    // 5. Open Cart Drawer and verify recommendation card and total are in USD ($)
    await page.getByTestId('cart-button').click();
    const cartDrawer = page.getByTestId('cart-drawer');
    await expect(cartDrawer).toBeVisible();

    const cartTotal = page.getByTestId('cart-total-price');
    await expect(cartTotal).toContainText('$');

    const cartRecCards = page.locator('[data-testid^="cart-recommendation-card-"]');
    if (await cartRecCards.count() > 0) {
      await expect(cartRecCards.first().locator('text=$')).toBeVisible();
    }

    // 6. Toggle back to IDR (Rp) while cart drawer is open (if toggle accessible) or close and toggle
    await page.getByTestId('cart-drawer-close').click();
    await langToggle.click();
    await expect(page.locator('text=Rp').first()).toBeVisible({ timeout: 5000 });

    // Reopen cart drawer and verify total is back to Rp
    await page.getByTestId('cart-button').click();
    await expect(page.getByTestId('cart-total-price')).toContainText('Rp');
    await page.getByTestId('cart-drawer-close').click();
  });
});
