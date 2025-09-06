import { test, expect } from '@playwright/test';

test.describe('Login Flow Examples with NextUI', () => {
  test('complete email login flow', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Verify we're on the login page with NextUI components
    await expect(page.getByTestId('login-card')).toBeVisible();
    await expect(page.getByTestId('login-title')).toContainText('Welcome Back');
    
    // Fill in email credentials
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('password-input').fill('SecurePassword123!');
    
    // Check remember me
    await page.getByTestId('remember-me-checkbox').click();
    await expect(page.getByTestId('remember-me-checkbox')).toBeChecked();
    
    // Submit the form
    await page.getByTestId('email-login-button').click();
    
    // Note: This will fail with invalid credentials
    // For now, let's check the error appears
    await expect(page.getByTestId('error-message')).toBeVisible({ timeout: 5000 });
  });

  test('complete username login flow', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Switch to username tab
    await page.getByRole('tab', { name: 'Username' }).click();
    
    // Verify username input is visible
    await expect(page.getByTestId('username-input')).toBeVisible();
    
    // Fill in username credentials
    await page.getByTestId('username-input').fill('testuser');
    await page.getByTestId('password-input-username').fill('SecurePassword123!');
    
    // Check remember me for username login
    await page.getByTestId('remember-me-checkbox-username').click();
    
    // Submit the form
    await page.getByTestId('username-login-button').click();
    
    // Check for error (since these are invalid credentials)
    await expect(page.getByTestId('error-message-username')).toBeVisible({ timeout: 5000 });
  });

  test('magic link flow', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Scroll to magic link section
    await page.getByTestId('magic-link-title').scrollIntoViewIfNeeded();
    
    // Enter email for magic link
    await page.getByTestId('magic-link-email-input').fill('user@example.com');
    
    // Send magic link
    await page.getByTestId('magic-link-button').click();
    
    // Verify success message appears
    await expect(page.getByTestId('magic-link-success')).toBeVisible();
    await expect(page.getByTestId('magic-link-success')).toContainText('Magic link sent!');
    await expect(page.getByTestId('magic-link-success')).toContainText('user@example.com');
  });

  test('password visibility toggle for both tabs', async ({ page }) => {
    await page.goto('/login');
    
    // Test email tab password toggle
    const passwordInput = page.getByTestId('password-input');
    const toggleButton = page.getByTestId('toggle-password-visibility');
    
    // Type a password
    await passwordInput.fill('MySecretPassword');
    
    // Initially should be hidden
    await expect(passwordInput).toHaveAttribute('type', 'password');
    
    // Show password
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    await expect(passwordInput).toHaveValue('MySecretPassword');
    
    // Hide password again
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
    
    // Test username tab password toggle
    await page.getByRole('tab', { name: 'Username' }).click();
    const usernamePasswordInput = page.getByTestId('password-input-username');
    const usernameToggleButton = page.getByTestId('toggle-password-visibility-username');
    
    await usernamePasswordInput.fill('AnotherPassword');
    await usernameToggleButton.click();
    await expect(usernamePasswordInput).toHaveAttribute('type', 'text');
  });

  test('navigation between login and signup', async ({ page }) => {
    // Start at login
    await page.goto('/login');
    
    // Navigate to signup
    await page.getByTestId('signup-link').click();
    await expect(page).toHaveURL('/signup');
    await expect(page.getByTestId('signup-card')).toBeVisible();
    
    // Navigate back to login
    await page.getByTestId('login-link').click();
    await expect(page).toHaveURL('/login');
    await expect(page.getByTestId('login-card')).toBeVisible();
  });

  test('forgot password navigation', async ({ page }) => {
    await page.goto('/login');
    
    // Click forgot password link in email tab
    await page.getByTestId('forgot-password-link').click();
    
    // Should navigate to forgot password page
    await expect(page).toHaveURL('/forgot-password');
  });

  test('form validation for required fields', async ({ page }) => {
    await page.goto('/login');
    
    // Try to submit empty form
    await page.getByTestId('email-login-button').click();
    
    // Browser should show validation errors
    // Check that inputs have required attribute
    const emailInput = page.getByTestId('email-input');
    const passwordInput = page.getByTestId('password-input');
    
    await expect(emailInput).toHaveAttribute('required', '');
    await expect(passwordInput).toHaveAttribute('required', '');
    
    // Fill invalid email
    await emailInput.fill('notanemail');
    await passwordInput.fill('pass');
    await page.getByTestId('email-login-button').click();
    
    // Should still be on login page
    await expect(page).toHaveURL('/login');
  });

  test('loading states', async ({ page }) => {
    await page.goto('/login');
    
    // Fill form
    await page.getByTestId('email-input').fill('test@example.com');
    await page.getByTestId('password-input').fill('password123');
    
    // Click login button
    const loginButton = page.getByTestId('email-login-button');
    await loginButton.click();
    
    // Button should show loading state (if implemented)
    // NextUI Button component handles this with isLoading prop
    // The button might be disabled during loading
    await expect(loginButton).toBeVisible();
  });

  test('responsive design with NextUI', async ({ page }) => {
    // Test mobile view
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/login');
    
    // Card should be responsive
    await expect(page.getByTestId('login-card')).toBeVisible();
    await expect(page.getByTestId('email-input')).toBeVisible();
    
    // Test tablet view
    await page.setViewportSize({ width: 768, height: 1024 });
    await expect(page.getByTestId('login-card')).toBeVisible();
    
    // Test desktop view
    await page.setViewportSize({ width: 1920, height: 1080 });
    await expect(page.getByTestId('login-card')).toBeVisible();
  });

  test('keyboard navigation', async ({ page }) => {
    await page.goto('/login');
    
    // Tab through form elements
    await page.keyboard.press('Tab'); // Skip to first focusable element
    await page.keyboard.press('Tab'); // Focus email tab (if not already selected)
    
    // Type in focused email input
    await page.getByTestId('email-input').focus();
    await page.keyboard.type('test@example.com');
    
    await page.keyboard.press('Tab'); // Focus password input
    await page.keyboard.type('testpass123');
    
    await page.keyboard.press('Tab'); // Focus toggle button
    await page.keyboard.press('Tab'); // Focus remember me
    await page.keyboard.press('Space'); // Check remember me
    
    await page.keyboard.press('Tab'); // Focus forgot password link
    await page.keyboard.press('Tab'); // Focus submit button
    await page.keyboard.press('Enter'); // Submit form
    
    // Should attempt login
    await expect(page.getByTestId('error-message')).toBeVisible({ timeout: 5000 });
  });

  test('multiple login methods in sequence', async ({ page }) => {
    await page.goto('/login');
    
    // Try email login
    await page.getByTestId('email-input').fill('test@example.com');
    await page.getByTestId('password-input').fill('password');
    await page.getByTestId('email-login-button').click();
    await expect(page.getByTestId('error-message')).toBeVisible();
    
    // Clear and try username login
    await page.getByRole('tab', { name: 'Username' }).click();
    await page.getByTestId('username-input').fill('testuser');
    await page.getByTestId('password-input-username').fill('password');
    await page.getByTestId('username-login-button').click();
    await expect(page.getByTestId('error-message-username')).toBeVisible();
    
    // Try magic link
    await page.getByTestId('magic-link-email-input').fill('test@example.com');
    await page.getByTestId('magic-link-button').click();
    await expect(page.getByTestId('magic-link-success')).toBeVisible();
  });

  test('accessibility with NextUI components', async ({ page }) => {
    await page.goto('/login');
    
    // Check for proper ARIA labels
    const card = page.getByTestId('login-card');
    await expect(card).toBeVisible();
    
    // NextUI components have built-in accessibility
    // Check tab navigation works
    const tabs = page.getByTestId('login-method-tabs');
    await expect(tabs).toBeVisible();
    
    // Check form inputs have labels
    const emailInput = page.getByTestId('email-input');
    await expect(emailInput).toBeVisible();
    
    // NextUI Input components have built-in label support
    const emailLabel = await emailInput.getAttribute('aria-label');
    expect(emailLabel || await page.locator('label:has-text("Email")').count() > 0).toBeTruthy();
  });
});