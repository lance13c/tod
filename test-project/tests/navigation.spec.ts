import { test, expect } from '@playwright/test';

test.describe('Navigation', () => {
  test('should navigate between auth pages', async ({ page }) => {
    // Start from home
    await page.goto('/');
    
    // Go to login
    await page.click('text=Sign In with Password');
    await expect(page).toHaveURL('/login');
    
    // Go to signup from login
    await page.click('text=create a new account');
    await expect(page).toHaveURL('/signup');
    
    // Go back to login from signup
    await page.click('text=sign in to existing account');
    await expect(page).toHaveURL('/login');
    
    // Go to magic link from login
    await page.click('text=Sign in with Magic Link');
    await expect(page).toHaveURL('/magic-link');
    
    // Go back to login from magic link
    await page.click('text=Password');
    await expect(page).toHaveURL('/login');
  });

  test('should protect dashboard route', async ({ page }) => {
    // Try to access dashboard without authentication
    await page.goto('/dashboard');
    
    // Should redirect to login
    await expect(page).toHaveURL('/login');
  });

  test('should have working links on all pages', async ({ page }) => {
    const pages = [
      { url: '/', title: 'Welcome to Better Auth' },
      { url: '/login', title: 'Sign in to your account' },
      { url: '/signup', title: 'Create your account' },
      { url: '/magic-link', title: 'Sign in with magic link' },
      { url: '/forgot-password', title: 'Reset your password' },
    ];

    for (const pageInfo of pages) {
      await page.goto(pageInfo.url);
      await expect(page.locator('h1, h2').first()).toContainText(pageInfo.title);
      
      // Check that the page loads without console errors
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });
      
      // Wait a moment for any async errors
      await page.waitForTimeout(1000);
      
      // No critical errors should occur
      const criticalErrors = consoleErrors.filter(error => 
        !error.includes('Failed to load resource') && 
        !error.includes('favicon')
      );
      expect(criticalErrors).toHaveLength(0);
    }
  });
});