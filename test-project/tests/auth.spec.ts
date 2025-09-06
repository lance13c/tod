import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('should display home page', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/Better Auth/);
    await expect(page.locator('h1')).toContainText('Welcome to Better Auth');
  });

  test('should navigate to login page', async ({ page }) => {
    await page.goto('/');
    await page.click('text=Sign In with Password');
    await expect(page).toHaveURL('/login');
    await expect(page.locator('h2')).toContainText('Sign in to your account');
  });

  test('should navigate to signup page', async ({ page }) => {
    await page.goto('/');
    await page.click('text=Create Account');
    await expect(page).toHaveURL('/signup');
    await expect(page.locator('h2')).toContainText('Create your account');
  });

  test('should navigate to magic link page', async ({ page }) => {
    await page.goto('/');
    await page.click('text=Sign In with Magic Link');
    await expect(page).toHaveURL('/magic-link');
    await expect(page.locator('h2')).toContainText('Sign in with magic link');
  });

  test('should show error for invalid login', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[name="email"]', 'invalid@example.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.bg-red-50')).toBeVisible();
    await expect(page.locator('.text-red-800')).toContainText('Invalid email or password');
  });

  test('should validate signup form', async ({ page }) => {
    await page.goto('/signup');
    
    // Test password mismatch
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'password123');
    await page.fill('input[name="confirmPassword"]', 'different123');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.bg-red-50')).toBeVisible();
    await expect(page.locator('.text-red-800')).toContainText('Passwords do not match');
    
    // Test short password
    await page.fill('input[name="password"]', 'short');
    await page.fill('input[name="confirmPassword"]', 'short');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.text-red-800')).toContainText('Password must be at least 8 characters');
  });

  test('should navigate to forgot password', async ({ page }) => {
    await page.goto('/login');
    await page.click('text=Forgot your password?');
    await expect(page).toHaveURL('/forgot-password');
    await expect(page.locator('h2')).toContainText('Reset your password');
  });

  test('should show success message for magic link request', async ({ page }) => {
    await page.goto('/magic-link');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.click('button[type="submit"]');
    
    // Wait for the success state
    await expect(page.locator('.bg-green-50')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Check your email')).toBeVisible();
  });
});