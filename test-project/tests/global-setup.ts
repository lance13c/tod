import { spawn } from 'child_process';
import { chromium, FullConfig } from '@playwright/test';
import fs from 'fs';
import path from 'path';

async function runCommand(command: string, args: string[]): Promise<void> {
  return new Promise((resolve, reject) => {
    console.log(`Running: ${command} ${args.join(' ')}`);
    const child = spawn(command, args, { 
      stdio: 'inherit',
      shell: true 
    });
    
    child.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(`Command failed with code ${code}`));
      } else {
        resolve();
      }
    });
    
    child.on('error', (err) => {
      reject(err);
    });
  });
}

async function globalSetup(config: FullConfig) {
  console.log('\n🚀 Starting Playwright global setup...\n');

  try {
    // Check if database exists
    const dbPath = path.join(process.cwd(), 'prisma', 'dev.db');
    const dbExists = fs.existsSync(dbPath);

    if (!dbExists) {
      console.log('📦 Database does not exist, creating and migrating...');
      
      // Create the database directory if it doesn't exist
      const prismaDir = path.join(process.cwd(), 'prisma');
      if (!fs.existsSync(prismaDir)) {
        fs.mkdirSync(prismaDir, { recursive: true });
      }

      // Push schema to create database
      await runCommand('bun', ['prisma', 'db', 'push']);
      console.log('✅ Database created\n');
    } else {
      console.log('✅ Database already exists\n');
      
      // Just push the schema to ensure it's up to date (won't lose data)
      console.log('📦 Ensuring database schema is up to date...');
      try {
        await runCommand('bun', ['prisma', 'db', 'push', '--skip-generate']);
        console.log('✅ Database schema updated\n');
      } catch (e) {
        console.log('⚠️  Could not update schema, continuing with existing database\n');
      }
    }

    // Seed the database with test data
    console.log('🌱 Seeding database with test data...');
    try {
      await runCommand('bun', ['prisma', 'db', 'seed']);
      console.log('✅ Database seeding completed\n');
    } catch (e) {
      console.log('⚠️  Seeding failed, database might already have data\n');
    }

    // Optional: Create authenticated browser state for reuse in tests
    console.log('🔐 Creating authenticated browser states...');
    
    // Check if the dev server is running
    try {
      const response = await fetch('http://localhost:3001');
      if (!response.ok) {
        throw new Error('Dev server not responding');
      }
    } catch (e) {
      console.log('⚠️  Dev server not running, skipping authenticated state creation');
      console.log('   Run "bun dev" in another terminal to enable authenticated tests\n');
      return;
    }

    const browser = await chromium.launch();

    // Create .auth directory if it doesn't exist
    const authDir = path.join(process.cwd(), 'tests', '.auth');
    if (!fs.existsSync(authDir)) {
      fs.mkdirSync(authDir, { recursive: true });
    }

    // Login as test user 1
    try {
      const context = await browser.newContext();
      const page = await context.newPage();
      
      await page.goto('http://localhost:3001/login');
      await page.getByTestId('email-input').fill('test@example.com');
      await page.getByTestId('password-input').fill('Password123!');
      await page.getByTestId('email-login-button').click();
      
      // Wait for navigation to dashboard (with timeout)
      await page.waitForURL('**/dashboard', { timeout: 5000 });
      // Save authentication state
      await context.storageState({ path: 'tests/.auth/user.json' });
      console.log('✅ Saved authenticated state for test@example.com');
      await context.close();
    } catch (e) {
      console.log('⚠️  Could not save authenticated state for test user');
    }

    // Login as admin user
    try {
      const adminContext = await browser.newContext();
      const adminPage = await adminContext.newPage();
      
      await adminPage.goto('http://localhost:3001/login');
      await adminPage.getByTestId('email-input').fill('admin@example.com');
      await adminPage.getByTestId('password-input').fill('Password123!');
      await adminPage.getByTestId('email-login-button').click();
      
      await adminPage.waitForURL('**/dashboard', { timeout: 5000 });
      await adminContext.storageState({ path: 'tests/.auth/admin.json' });
      console.log('✅ Saved authenticated state for admin@example.com');
      await adminContext.close();
    } catch (e) {
      console.log('⚠️  Could not save authenticated state for admin user');
    }

    // Login as John (user with organizations)
    try {
      const johnContext = await browser.newContext();
      const johnPage = await johnContext.newPage();
      
      await johnPage.goto('http://localhost:3001/login');
      await johnPage.getByTestId('email-input').fill('john@example.com');
      await johnPage.getByTestId('password-input').fill('Password123!');
      await johnPage.getByTestId('email-login-button').click();
      
      await johnPage.waitForURL('**/dashboard', { timeout: 5000 });
      await johnContext.storageState({ path: 'tests/.auth/john.json' });
      console.log('✅ Saved authenticated state for john@example.com');
      await johnContext.close();
    } catch (e) {
      console.log('⚠️  Could not save authenticated state for john user');
    }

    await browser.close();

    console.log('\n✨ Global setup completed successfully!\n');
  } catch (error) {
    console.error('\n❌ Global setup failed:', error);
    
    // Don't fail completely, allow tests to continue
    console.log('\n⚠️  Setup had issues but tests will continue');
    console.log('   Some features might not work properly\n');
  }
}

export default globalSetup;