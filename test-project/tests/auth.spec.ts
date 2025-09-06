import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Clear any existing auth state
    await page.context().clearCookies();
  });

  test('should display login page with NextUI components', async ({ page }) => {
    await page.goto('/login');
    
    // Check for NextUI Card structure
    await expect(page.getByTestId('login-card')).toBeVisible();
    await expect(page.getByTestId('login-title')).toContainText('Welcome Back');
    await expect(page.getByTestId('login-subtitle')).toContainText('Sign in to access your account');
    
    // Check for tabs
    await expect(page.getByTestId('login-method-tabs')).toBeVisible();
    await expect(page.getByTestId('email-tab')).toBeVisible();
    await expect(page.getByTestId('username-tab')).toBeVisible();
  });

  test('should switch between email and username login methods', async ({ page }) => {
    await page.goto('/login');
    
    // Default should be email tab
    await expect(page.getByTestId('email-input')).toBeVisible();
    
    // Switch to username tab
    await page.getByRole('tab', { name: 'Username' }).click();
    await expect(page.getByTestId('username-input')).toBeVisible();
    
    // Switch back to email tab
    await page.getByRole('tab', { name: 'Email' }).click();
    await expect(page.getByTestId('email-input')).toBeVisible();
  });

  test('should toggle password visibility', async ({ page }) => {
    await page.goto('/login');
    
    const passwordInput = page.getByTestId('password-input');
    const toggleButton = page.getByTestId('toggle-password-visibility');
    
    // Initially password should be hidden
    await expect(passwordInput).toHaveAttribute('type', 'password');
    
    // Click toggle to show password
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    
    // Click toggle to hide password again
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('should show error for invalid email login', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByTestId('email-input').fill('invalid@example.com');
    await page.getByTestId('password-input').fill('wrongpassword');
    await page.getByTestId('email-login-button').click();
    
    await expect(page.getByTestId('error-message')).toBeVisible();
    await expect(page.getByTestId('error-message')).toContainText(/Failed to sign in/);
  });

  test('should show error for invalid username login', async ({ page }) => {
    await page.goto('/login');
    
    // Switch to username tab
    await page.getByRole('tab', { name: 'Username' }).click();
    
    await page.getByTestId('username-input').fill('invaliduser');
    await page.getByTestId('password-input-username').fill('wrongpassword');
    await page.getByTestId('username-login-button').click();
    
    await expect(page.getByTestId('error-message-username')).toBeVisible();
    await expect(page.getByTestId('error-message-username')).toContainText(/Failed to sign in/);
  });

  test('should handle magic link request', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByTestId('magic-link-email-input').fill('test@example.com');
    await page.getByTestId('magic-link-button').click();
    
    // Should show success message
    await expect(page.getByTestId('magic-link-success')).toBeVisible();
    await expect(page.getByTestId('magic-link-success')).toContainText('Magic link sent!');
    await expect(page.getByTestId('magic-link-success')).toContainText('test@example.com');
  });

  test('should navigate to signup page', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByTestId('signup-link').click();
    await expect(page).toHaveURL('/signup');
    await expect(page.getByTestId('signup-card')).toBeVisible();
  });

  test('should navigate to forgot password', async ({ page }) => {
    await page.goto('/login');
    
    await page.getByTestId('forgot-password-link').click();
    await expect(page).toHaveURL('/forgot-password');
  });

  test('should remember me checkbox work', async ({ page }) => {
    await page.goto('/login');
    
    const checkbox = page.getByTestId('remember-me-checkbox');
    
    // Initially unchecked
    await expect(checkbox).not.toBeChecked();
    
    // Check it
    await checkbox.click();
    await expect(checkbox).toBeChecked();
    
    // Uncheck it
    await checkbox.click();
    await expect(checkbox).not.toBeChecked();
  });
});

test.describe('Signup', () => {
  test('should display signup page with NextUI components', async ({ page }) => {
    await page.goto('/signup');
    
    await expect(page.getByTestId('signup-card')).toBeVisible();
    await expect(page.getByTestId('signup-title')).toContainText('Create Account');
    await expect(page.getByTestId('signup-subtitle')).toContainText('Join us to get started');
  });

  test('should show password strength indicator', async ({ page }) => {
    await page.goto('/signup');
    
    const passwordInput = page.getByTestId('password-input');
    
    // Weak password
    await passwordInput.fill('weak');
    await expect(page.getByTestId('password-strength-indicator')).toBeVisible();
    await expect(page.locator('text=Weak')).toBeVisible();
    
    // Strong password
    await passwordInput.fill('StrongP@ssw0rd123!');
    await expect(page.locator('text=Strong')).toBeVisible();
  });

  test('should validate password match', async ({ page }) => {
    await page.goto('/signup');
    
    await page.getByTestId('password-input').fill('Password123!');
    await page.getByTestId('confirm-password-input').fill('DifferentPassword');
    
    // Should show error message
    await expect(page.locator('text=Passwords do not match')).toBeVisible();
  });

  test('should require terms acceptance', async ({ page }) => {
    await page.goto('/signup');
    
    // Fill all fields
    await page.getByTestId('name-input').fill('John Doe');
    await page.getByTestId('username-input').fill('johndoe');
    await page.getByTestId('email-input').fill('john@example.com');
    await page.getByTestId('password-input').fill('Password123!');
    await page.getByTestId('confirm-password-input').fill('Password123!');
    
    // Button should be disabled without terms acceptance
    await expect(page.getByTestId('signup-button')).toBeDisabled();
    
    // Accept terms
    await page.getByTestId('terms-checkbox').click();
    await expect(page.getByTestId('signup-button')).toBeEnabled();
    
    // Try to submit
    await page.getByTestId('signup-button').click();
  });

  test('should navigate to login page', async ({ page }) => {
    await page.goto('/signup');
    
    await page.getByTestId('login-link').click();
    await expect(page).toHaveURL('/login');
    await expect(page.getByTestId('login-card')).toBeVisible();
  });

  test('should toggle password visibility for both fields', async ({ page }) => {
    await page.goto('/signup');
    
    const passwordInput = page.getByTestId('password-input');
    const confirmPasswordInput = page.getByTestId('confirm-password-input');
    const togglePassword = page.getByTestId('toggle-password-visibility');
    const toggleConfirm = page.getByTestId('toggle-confirm-password-visibility');
    
    // Initially both should be hidden
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(confirmPasswordInput).toHaveAttribute('type', 'password');
    
    // Toggle main password
    await togglePassword.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    
    // Toggle confirm password
    await toggleConfirm.click();
    await expect(confirmPasswordInput).toHaveAttribute('type', 'text');
  });
});