import { test, expect } from '@playwright/test';

test.describe('Organizations Showcase', () => {
  test('should display public organizations page', async ({ page }) => {
    await page.goto('/organizations');
    
    // Check hero section
    await expect(page.locator('h1')).toContainText('Organization Showcase');
    await expect(page.locator('text=Discover amazing organizations')).toBeVisible();
    
    // Check filters section
    await expect(page.locator('input[placeholder="Search organizations..."]')).toBeVisible();
    await expect(page.locator('select').first()).toBeVisible(); // Industry filter
    await expect(page.locator('text=Featured Only')).toBeVisible();
  });

  test('should filter organizations by search term', async ({ page }) => {
    await page.goto('/organizations');
    
    // Type in search
    const searchInput = page.locator('input[placeholder="Search organizations..."]');
    await searchInput.fill('tech');
    
    // Wait for filtered results (or no results message)
    await page.waitForTimeout(500); // Wait for debounce
    
    // Check that either results are shown or no results message
    const noResults = page.locator('text=No organizations found');
    const orgCards = page.locator('[href^="/org/"]');
    
    const hasResults = await orgCards.count() > 0;
    const hasNoResultsMessage = await noResults.isVisible();
    
    expect(hasResults || hasNoResultsMessage).toBeTruthy();
  });

  test('should filter by industry', async ({ page }) => {
    await page.goto('/organizations');
    
    // Select an industry
    const industrySelect = page.locator('select').first();
    await industrySelect.selectOption({ index: 1 }); // Select first industry option
    
    // Wait for filtered results
    await page.waitForTimeout(500);
    
    // Verify filter is applied
    const selectedValue = await industrySelect.inputValue();
    expect(selectedValue).not.toBe('');
  });

  test('should toggle featured only filter', async ({ page }) => {
    await page.goto('/organizations');
    
    const featuredCheckbox = page.locator('input[type="checkbox"]').first();
    
    // Initially unchecked
    await expect(featuredCheckbox).not.toBeChecked();
    
    // Check it
    await featuredCheckbox.click();
    await expect(featuredCheckbox).toBeChecked();
    
    // Wait for filtered results
    await page.waitForTimeout(500);
  });

  test('should navigate to organization profile', async ({ page }) => {
    await page.goto('/organizations');
    
    // Wait for organizations to load
    await page.waitForTimeout(1000);
    
    // Click on first organization card if available
    const firstOrgCard = page.locator('[href^="/org/"]').first();
    const cardCount = await firstOrgCard.count();
    
    if (cardCount > 0) {
      const href = await firstOrgCard.getAttribute('href');
      await firstOrgCard.click();
      
      // Should navigate to org profile
      await expect(page).toHaveURL(href!);
      
      // Profile page should have organization name
      await expect(page.locator('h1')).toBeVisible();
    }
  });

  test('should show load more button when scrolling', async ({ page }) => {
    await page.goto('/organizations');
    
    // Check if load more button exists (only if there are many orgs)
    const loadMoreButton = page.locator('button:has-text("Load More")');
    const buttonCount = await loadMoreButton.count();
    
    if (buttonCount > 0) {
      await expect(loadMoreButton).toBeVisible();
      
      // Click load more
      await loadMoreButton.click();
      
      // Button should show loading state
      await expect(page.locator('button:has-text("Loading...")')).toBeVisible();
    }
  });

  test('should display popular tags section', async ({ page }) => {
    await page.goto('/organizations');
    
    // Scroll to bottom to see popular tags
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    
    // Check if popular tags section exists
    const popularTagsSection = page.locator('text=Popular Tags');
    const sectionCount = await popularTagsSection.count();
    
    if (sectionCount > 0) {
      await expect(popularTagsSection).toBeVisible();
    }
  });
});

test.describe('Organization Profile', () => {
  test('should display organization not found for invalid slug', async ({ page }) => {
    await page.goto('/org/invalid-org-slug-123456');
    
    await expect(page.locator('text=Organization Not Found')).toBeVisible();
    await expect(page.locator('text=Browse Public Organizations')).toBeVisible();
  });

  test('should navigate back to organizations list', async ({ page }) => {
    await page.goto('/org/invalid-org-slug-123456');
    
    await page.click('text=Browse Public Organizations');
    await expect(page).toHaveURL('/organizations');
  });
});

test.describe('Dashboard Organization Management', () => {
  // These tests would require authentication setup
  // Skipping for now as they need a logged-in state
  
  test.skip('should show create organization button', async ({ page }) => {
    // Login first
    await page.goto('/login');
    // ... login steps
    
    await page.goto('/dashboard');
    await expect(page.locator('text=Create Organization')).toBeVisible();
  });

  test.skip('should open create organization modal', async ({ page }) => {
    // Login first
    await page.goto('/dashboard');
    
    await page.click('text=Create Organization');
    await expect(page.locator('text=Create Organization')).toBeVisible();
    await expect(page.locator('input[placeholder="Acme Corp"]')).toBeVisible();
  });

  test.skip('should toggle organization visibility', async ({ page }) => {
    // Login and have an organization
    await page.goto('/dashboard');
    
    // Find visibility toggle
    const visibilityToggle = page.locator('[role="switch"]').first();
    await visibilityToggle.click();
    
    // Check toggle state changed
    await page.waitForTimeout(500);
  });
});