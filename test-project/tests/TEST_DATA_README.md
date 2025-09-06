# Test Data Setup Guide

This guide explains how to set up and use test data for Playwright tests.

## Quick Start

1. **Setup test environment with seeded data:**
```bash
bun run test:setup
```

2. **Run tests:**
```bash
# Run all tests
bun test

# Run with UI mode for debugging
bun test:ui

# Run specific test file
bun test tests/auth-with-seed.spec.ts
```

## Test Credentials

The following test users are created by the seed script:

| Email | Username | Password | Description |
|-------|----------|----------|-------------|
| test@example.com | testuser | Password123! | Regular user with basic access |
| john@example.com | johndoe | Password123! | User with multiple organizations |
| admin@example.com | admin | Password123! | Admin user with elevated privileges |

## Test Organizations

The seed script creates 15 test organizations:

### Featured Organizations
- **Acme Corporation** (slug: `acme-corp`) - Public, Featured, Verified
- **Green Energy Solutions** (slug: `green-energy`) - Public, Featured

### Regular Organizations
- **TechStart Inc** (slug: `techstart`) - Public, Verified
- **Healthcare Innovations** (slug: `healthcare-innovations`) - Public, Verified
- **Private Ventures** (slug: `private-ventures`) - Private (not visible in public showcase)

### Test Organizations
- 10 additional organizations (`test-org-6` through `test-org-15`) for pagination testing

## Database Management

### Reset and reseed database
```bash
bun run db:reset  # Warning: This will delete all data
bun run db:seed   # Repopulate with test data
```

### View database contents
```bash
bun run db:studio  # Opens Prisma Studio in browser
```

### Update schema without losing data
```bash
bun prisma db push
```

## Running Specific Test Suites

### Authenticated Tests
These tests use pre-saved authentication states:

```bash
# Run as regular user (test@example.com)
bun run test:auth

# Run as admin (admin@example.com)
bun run test:admin

# Run as user with organizations (john@example.com)
bun run test:john
```

### Manual Login Tests
Tests that manually perform login:
```bash
bun test tests/auth.spec.ts
bun test tests/login-example.spec.ts
```

### Organization Tests
Tests for organization features:
```bash
bun test tests/organizations.spec.ts
bun test tests/dashboard.spec.ts
```

## Troubleshooting

### Database Issues

If you encounter database errors:

1. **Database out of sync:**
```bash
bun prisma db push --skip-generate
bun run db:seed
```

2. **Complete reset:**
```bash
rm prisma/dev.db
bun run test:setup
```

### Authentication Issues

If authenticated tests fail:

1. Make sure the dev server is running:
```bash
bun dev  # In another terminal
```

2. Clear authentication states:
```bash
rm -rf tests/.auth/*.json
bun test  # Will recreate auth states
```

### Test Data Already Exists

The seed script cleans existing data before inserting new data. If you see errors about duplicate data:

```bash
bun run db:reset  # Complete reset
bun run db:seed   # Fresh seed
```

## Test Data Structure

### User Relationships
- `test@example.com` - Member of Healthcare Innovations
- `john@example.com` - Owner of Acme Corp, TechStart, Private Ventures
- `admin@example.com` - Admin of Acme Corp, Member of Green Energy

### Organization Visibility
- **Public Organizations**: Visible on `/organizations` page
- **Private Organizations**: Only visible to members
- **Featured Organizations**: Highlighted in showcase
- **Verified Organizations**: Display verification badge

## Writing New Tests

When writing tests that require authenticated users:

```typescript
// Use pre-authenticated state
test.describe('Authenticated tests', () => {
  test.use({ storageState: 'tests/.auth/john.json' });
  
  test('should see dashboard', async ({ page }) => {
    await page.goto('/dashboard');
    // Already logged in as john@example.com
  });
});

// Or login manually
test('manual login', async ({ page }) => {
  await page.goto('/login');
  await page.getByTestId('email-input').fill('test@example.com');
  await page.getByTestId('password-input').fill('Password123!');
  await page.getByTestId('email-login-button').click();
  await expect(page).toHaveURL('/dashboard');
});
```

## CI/CD Integration

For CI environments, the global setup automatically:
1. Creates database if it doesn't exist
2. Seeds with test data
3. Creates authentication states (if server is running)

No manual setup required - just run `bun test`!