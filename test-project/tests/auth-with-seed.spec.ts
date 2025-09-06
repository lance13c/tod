import { test, expect } from '@playwright/test';

test.describe('Authentication with Seeded Data', () => {
  test('should login with test user credentials', async ({ page }) => {
    await page.goto('/login');
    
    // Login with seeded test user
    await page.getByTestId('email-input').fill('test@example.com');
    await page.getByTestId('password-input').fill('Password123!');
    await page.getByTestId('email-login-button').click();
    
    // Should redirect to dashboard
    await expect(page).toHaveURL('/dashboard', { timeout: 10000 });
    
    // Should see user information
    await expect(page.locator('text=test@example.com')).toBeVisible();
  });

  test('should login with username', async ({ page }) => {
    await page.goto('/login');
    
    // Switch to username tab
    await page.getByRole('tab', { name: 'Username' }).click();
    
    // Login with seeded username
    await page.getByTestId('username-input').fill('johndoe');
    await page.getByTestId('password-input-username').fill('Password123!');
    await page.getByTestId('username-login-button').click();
    
    // Should redirect to dashboard
    await expect(page).toHaveURL('/dashboard', { timeout: 10000 });
    
    // Should see user information
    await expect(page.locator('text=john@example.com')).toBeVisible();
  });

  test('should show organizations for john user', async ({ page }) => {
    await page.goto('/login');
    
    // Login as John (who has organizations)
    await page.getByTestId('email-input').fill('john@example.com');
    await page.getByTestId('password-input').fill('Password123!');
    await page.getByTestId('email-login-button').click();
    
    // Wait for dashboard
    await expect(page).toHaveURL('/dashboard', { timeout: 10000 });
    
    // Should see organizations section
    await expect(page.locator('text=Your Organizations')).toBeVisible();
    
    // Should see John's organizations
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=TechStart Inc')).toBeVisible();
    await expect(page.locator('text=Private Ventures')).toBeVisible();
  });

  test('should access public organizations page', async ({ page }) => {
    await page.goto('/organizations');
    
    // Should see public organizations
    await expect(page.locator('text=Organization Showcase')).toBeVisible();
    
    // Should see featured organizations
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=Green Energy Solutions')).toBeVisible();
    
    // Should NOT see private organizations
    await expect(page.locator('text=Private Ventures')).not.toBeVisible();
  });

  test('should view organization profile', async ({ page }) => {
    await page.goto('/org/acme-corp');
    
    // Should see organization details
    await expect(page.locator('h1:has-text("Acme Corporation")')).toBeVisible();
    await expect(page.locator('text=@acme-corp')).toBeVisible();
    await expect(page.locator('text=Leading provider of innovative solutions')).toBeVisible();
    
    // Should see team members
    await expect(page.locator('text=Team Members')).toBeVisible();
    await expect(page.locator('text=John Doe')).toBeVisible();
    await expect(page.locator('text=CEO & Founder')).toBeVisible();
  });

  test('should filter organizations by industry', async ({ page }) => {
    await page.goto('/organizations');
    
    // Select Technology industry
    const industrySelect = page.locator('select').first();
    await industrySelect.selectOption('Technology');
    
    // Wait for filtered results
    await page.waitForTimeout(500);
    
    // Should see technology companies
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=TechStart Inc')).toBeVisible();
    
    // Change to Healthcare
    await industrySelect.selectOption('Healthcare');
    await page.waitForTimeout(500);
    
    // Should see healthcare company
    await expect(page.locator('text=Healthcare Innovations')).toBeVisible();
  });
});

test.describe('Authenticated Dashboard Features', () => {
  test.use({ storageState: 'tests/.auth/john.json' });

  test.skip('should toggle organization visibility', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Find Acme Corporation's visibility toggle
    const acmeCard = page.locator('text=Acme Corporation').locator('..');
    const visibilityToggle = acmeCard.locator('[role="switch"]').first();
    
    // Toggle visibility
    await visibilityToggle.click();
    
    // Wait for update
    await page.waitForTimeout(1000);
    
    // The visibility text should update
    const visibilityText = acmeCard.locator('text=/Public|Private/').first();
    await expect(visibilityText).toBeVisible();
  });

  test.skip('should create new organization', async ({ page }) => {
    await page.goto('/dashboard');
    
    // Click create organization
    await page.click('text=Create Organization');
    
    // Fill form
    await page.fill('input[placeholder="Acme Corp"]', 'Test New Org');
    await page.fill('textarea', 'This is a test organization created by Playwright');
    await page.fill('input[placeholder="Technology"]', 'Testing');
    await page.fill('input[placeholder="10-50"]', '1-10');
    
    // Make it public
    await page.check('#isPublic');
    
    // Submit
    await page.click('button:has-text("Create")');
    
    // Should see new organization in list
    await expect(page.locator('text=Test New Org')).toBeVisible({ timeout: 5000 });
  });
});