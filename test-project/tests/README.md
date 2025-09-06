# Playwright Tests

## Running Tests

```bash
# Run all tests
bun run test

# Run tests in UI mode (interactive)
bun run test:ui

# Run tests in debug mode
bun run test:debug

# Run specific test file
bunx playwright test login-example

# Run tests in headed mode (see browser)
bunx playwright test --headed

# Run only chromium tests
bunx playwright test --project=chromium

# Show test report after run
bun run test:report
```

## Test Structure

### Available Test Files

- `auth.spec.ts` - Basic authentication flow tests
- `navigation.spec.ts` - Navigation and routing tests  
- `login-example.spec.ts` - Comprehensive login test examples

### Writing New Tests

1. Create a new file in `/tests` with `.spec.ts` extension
2. Import Playwright test utilities:
   ```typescript
   import { test, expect } from '@playwright/test';
   ```
3. Group related tests with `test.describe()`
4. Use `test()` for individual test cases

## Common Test Patterns

### Basic Page Test
```typescript
test('should load home page', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle(/Your App/);
});
```

### Form Interaction
```typescript
test('submit form', async ({ page }) => {
  await page.goto('/login');
  await page.fill('input[name="email"]', 'test@example.com');
  await page.fill('input[name="password"]', 'password123');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL('/dashboard');
});
```

### Waiting for Elements
```typescript
// Wait for element to be visible
await expect(page.locator('.success-message')).toBeVisible();

// Wait for navigation
await page.waitForURL('/dashboard');

// Wait for network request
await page.waitForResponse(resp => resp.url().includes('/api/auth'));
```

### Testing API Responses
```typescript
test('api returns user data', async ({ page }) => {
  // Intercept API calls
  const responsePromise = page.waitForResponse('/api/auth/session');
  await page.goto('/dashboard');
  const response = await responsePromise;
  expect(response.status()).toBe(200);
});
```

## Test Data Setup

### Using Test Users

For tests requiring authenticated users, you'll need to either:

1. **Create test users in your database:**
   ```bash
   # Add a test user to your database
   # You might want to create a seed script
   ```

2. **Mock authentication in tests:**
   ```typescript
   // Override auth endpoint
   await page.route('/api/auth/sign-in/email', route => {
     route.fulfill({
       status: 200,
       body: JSON.stringify({ 
         user: { email: 'test@example.com' },
         session: { token: 'test-token' }
       })
     });
   });
   ```

3. **Use authentication state:**
   ```bash
   # Generate auth state file
   bunx playwright codegen --save-storage=auth.json
   ```

## Debugging Tests

### Visual Debugging
```bash
# Run with UI mode for step-by-step debugging
bun run test:ui

# Use debug mode with breakpoints
bun run test:debug
```

### In-Test Debugging
```typescript
test('debug example', async ({ page }) => {
  // Pause test execution
  await page.pause();
  
  // Take screenshot
  await page.screenshot({ path: 'debug.png' });
  
  // Log page content
  console.log(await page.content());
});
```

## Best Practices

1. **Use data-testid attributes** for reliable element selection:
   ```tsx
   <button data-testid="submit-button">Submit</button>
   ```
   ```typescript
   await page.click('[data-testid="submit-button"]');
   ```

2. **Avoid hard-coded waits:**
   ```typescript
   // Bad
   await page.waitForTimeout(5000);
   
   // Good
   await expect(page.locator('.loading')).toBeHidden();
   ```

3. **Group related tests:**
   ```typescript
   test.describe('Feature Name', () => {
     test.beforeEach(async ({ page }) => {
       // Common setup
     });
     
     test('scenario 1', async ({ page }) => {});
     test('scenario 2', async ({ page }) => {});
   });
   ```

4. **Use Page Object Model for complex tests:**
   ```typescript
   class LoginPage {
     constructor(private page: Page) {}
     
     async login(email: string, password: string) {
       await this.page.fill('input[name="email"]', email);
       await this.page.fill('input[name="password"]', password);
       await this.page.click('button[type="submit"]');
     }
   }
   ```

## CI/CD Integration

Add to your CI pipeline:

```yaml
# GitHub Actions example
- name: Install dependencies
  run: bun install
  
- name: Install Playwright browsers
  run: bunx playwright install --with-deps
  
- name: Run tests
  run: bun run test
  
- name: Upload test results
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: playwright-report/
```

## Troubleshooting

### Tests failing locally
1. Ensure dev server is running: `bun run dev`
2. Check database is seeded with test data
3. Clear browser cache: `bunx playwright clean`

### Tests timing out
- Increase timeout in config or specific test:
  ```typescript
  test.setTimeout(60000); // 60 seconds
  ```

### Flaky tests
- Use `test.retry()` for unreliable tests
- Add proper wait conditions
- Check for race conditions in your app

## Resources

- [Playwright Documentation](https://playwright.dev/docs/intro)
- [Best Practices](https://playwright.dev/docs/best-practices)
- [API Reference](https://playwright.dev/docs/api/class-test)
- [Debugging Guide](https://playwright.dev/docs/debug)