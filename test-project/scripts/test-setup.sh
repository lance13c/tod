#!/bin/bash

echo "🚀 Setting up test environment..."

# Ensure database exists and is up to date
echo "📦 Updating database schema..."
bun prisma db push --skip-generate

# Seed the database
echo "🌱 Seeding database with test data..."
bun run db:seed

echo "✅ Test environment ready!"
echo ""
echo "📝 Test Credentials:"
echo "───────────────────────────────────────"
echo "Email: test@example.com  | Password: Password123!"
echo "Email: john@example.com  | Password: Password123!"
echo "Email: admin@example.com | Password: Password123!"
echo "───────────────────────────────────────"
echo ""
echo "Now run: bun test"