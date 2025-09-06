import { test, expect } from '@playwright/test';

// Helper function to login
async function loginUser(page: any, email: string = 'test@example.com', password: string = 'Password123!') {
  await page.goto('/login');
  await page.getByTestId('email-input').fill(email);
  await page.getByTestId('password-input').fill(password);
  await page.getByTestId('email-login-button').click();
  
  // Wait for navigation to dashboard
  await page.waitForURL('/dashboard', { timeout: 5000 }).catch(() => {
    // If login fails, we're still on login page
  });
}

test.describe('Dashboard', () => {
  test('should redirect to login when not authenticated', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Should redirect to login
    await expect(page).toHaveURL('/login');
    await expect(page.getByTestId('login-card')).toBeVisible();
  });

  test('should display loading state', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Should show loading spinner briefly
    const spinner = page.locator('.animate-spin');
    
    // Either we see the spinner or we're redirected quickly
    const spinnerExists = await spinner.count() > 0;
    if (spinnerExists) {
      await expect(spinner).toBeVisible();
    }
  });
});

test.describe('Dashboard - Authenticated', () => {
  // These tests require a valid user account
  // You would need to either:
  // 1. Create a test user before running tests
  // 2. Mock the authentication
  // 3. Use test database with seed data
  
  test.skip('should display user information', async ({ page }) => {
    await loginUser(page);
    
    // Check user info section
    await expect(page.locator('text=Account Information')).toBeVisible();
    await expect(page.locator('text=test@example.com')).toBeVisible();
  });

  test.skip('should display organizations section', async ({ page }) => {
    await loginUser(page);
    
    await expect(page.locator('text=Your Organizations')).toBeVisible();
    await expect(page.locator('text=Create Organization')).toBeVisible();
  });

  test.skip('should display statistics', async ({ page }) => {
    await loginUser(page);
    
    await expect(page.locator('text=Statistics')).toBeVisible();
    await expect(page.locator('text=Organizations')).toBeVisible();
    await expect(page.locator('text=Public')).toBeVisible();
    await expect(page.locator('text=Private')).toBeVisible();
  });

  test.skip('should open create organization modal', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Create Organization');
    
    // Modal should appear
    await expect(page.locator('text=Organization Name *')).toBeVisible();
    await expect(page.locator('input[placeholder="Acme Corp"]')).toBeVisible();
    await expect(page.locator('input[placeholder="acme-corp"]')).toBeVisible();
  });

  test.skip('should close modal on cancel', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Create Organization');
    await expect(page.locator('text=Organization Name *')).toBeVisible();
    
    await page.click('button:has-text("Cancel")');
    
    // Modal should close
    await expect(page.locator('text=Organization Name *')).not.toBeVisible();
  });

  test.skip('should auto-generate slug from name', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Create Organization');
    
    const nameInput = page.locator('input[placeholder="Acme Corp"]');
    const slugInput = page.locator('input[placeholder="acme-corp"]');
    
    await nameInput.fill('My Test Organization');
    
    // Slug should auto-generate
    await expect(slugInput).toHaveValue('my-test-organization');
  });

  test.skip('should validate organization form', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Create Organization');
    
    // Try to submit empty form
    await page.click('button:has-text("Create")');
    
    // Should show validation errors
    await expect(page.locator('text=Organization Name *')).toBeVisible();
  });

  test.skip('should toggle public visibility checkbox', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Create Organization');
    
    const publicCheckbox = page.locator('input[type="checkbox"]#isPublic');
    
    // Initially unchecked
    await expect(publicCheckbox).not.toBeChecked();
    
    // Check it
    await publicCheckbox.click();
    await expect(publicCheckbox).toBeChecked();
  });

  test.skip('should navigate to browse organizations', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Browse Organizations');
    await expect(page).toHaveURL('/organizations');
  });

  test.skip('should sign out', async ({ page }) => {
    await loginUser(page);
    
    await page.click('text=Sign Out');
    
    // Should redirect to login
    await expect(page).toHaveURL('/login');
    await expect(page.getByTestId('login-card')).toBeVisible();
  });
});

test.describe('Dashboard - Organization Cards', () => {
  test.skip('should display organization cards', async ({ page }) => {
    await loginUser(page);
    
    // If user has organizations, they should be displayed
    const orgCards = page.locator('.border.rounded-lg.p-4');
    const cardCount = await orgCards.count();
    
    if (cardCount > 0) {
      await expect(orgCards.first()).toBeVisible();
      
      // Card should have actions
      await expect(page.locator('text=View Profile').first()).toBeVisible();
      await expect(page.locator('text=Edit').first()).toBeVisible();
      await expect(page.locator('text=Delete').first()).toBeVisible();
    } else {
      // Should show empty state
      await expect(page.locator('text=No organizations yet')).toBeVisible();
    }
  });

  test.skip('should toggle organization visibility', async ({ page }) => {
    await loginUser(page);
    
    const toggles = page.locator('[role="switch"]');
    const toggleCount = await toggles.count();
    
    if (toggleCount > 0) {
      const firstToggle = toggles.first();
      
      // Click toggle
      await firstToggle.click();
      
      // Wait for update
      await page.waitForTimeout(1000);
      
      // Text should update
      const visibilityText = await page.locator('text=Public, text=Private').first().textContent();
      expect(visibilityText).toBeTruthy();
    }
  });

  test.skip('should confirm before deleting organization', async ({ page }) => {
    await loginUser(page);
    
    const deleteButtons = page.locator('button:has-text("Delete")');
    const buttonCount = await deleteButtons.count();
    
    if (buttonCount > 0) {
      // Mock window.confirm
      await page.evaluate(() => {
        window.confirm = () => false; // Cancel deletion
      });
      
      await deleteButtons.first().click();
      
      // Organization should still be there
      await expect(deleteButtons.first()).toBeVisible();
    }
  });
});