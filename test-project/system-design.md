GroupUp MVP - System Design Document
1. System Overview
GroupUp MVP is a simplified proximity-based onboarding utility that allows users to quickly connect with those around them and share document/photos/links/credentials 

Core Features (MVP)

Email/password magic link sign in. Show the last auth method a user used (based on local storage flag)
Create/join geo-locked sharing sessions
Have a guest/no login needed page that just asks for geolocation, checks a sqlite db for what building polygon (based on https://github.com/microsoft/USBuildingFootprints?tab=readme-ov-file) geojson files, which building the user is currently it, then setup a webrtc connection with all other participants.
Upload and share documents to the webrtc. Source phone/computer has to stay awake/on (the files aren't really uploaded, they are transmitted directly)
Grid-based document/photo gallery (like immich)
Mobile-responsive design
Persistent user sessions and documents (have the user be presented an option for logging in)
Easy to understand, keep it simple - stupid, frontpage that describes how this is a 

2. Architecture Diagram
┌─────────────────────────────────────────────────────────────┐
│                     Client (Browser/PWA)                     │
│                   Next.js + TailwindCSS                      │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    Next.js Application                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   App Router                         │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │   │
│  │  │   Auth   │  │   API    │  │  Static Assets   │ │   │
│  │  │  Routes  │  │  Routes  │  │   (Documents)    │ │   │
│  │  └──────────┘  └──────────┘  └──────────────────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
│                               │                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Better Auth                         │   │
│  │         (Email/Password + Google/GitHub SSO)        │   │
│  └─────────────────────────────────────────────────────┘   │
│                               │                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Prisma ORM                          │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                     SQLite Database                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Users   │  │ Sessions │  │Documents │  │  Shares  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                   File System Storage                        │
│                  /app/uploads (Docker Volume)                │
└─────────────────────────────────────────────────────────────┘
3. Technology Stack
Frontend

Framework: Next.js 15 (App Router)
Styling: TailwindCSS 3.4
UI Components: Custom components + Radix UI primitives
Icons: Lucide React
Forms: React Hook Form + Zod validation
Grid Layout: CSS Grid with Masonry fallback
trpc

Backend

Runtime: Node.js 20 LTS
Framework: Next.js API Routes
Authentication: Better Auth
Database: SQLite with Prisma ORM
File Storage: Local filesystem (Docker volume)
Session Management: JWT tokens

Testing

E2E Testing: Playwright
Unit Testing: Jest + React Testing Library
API Testing: Supertest
Test Coverage: NYC/C8

DevOps

Container: Docker + Docker Compose
Development: Hot reload with mounted volumes
Production: Multi-stage Docker build
CI/CD: GitHub Actions with Playwright tests

4. Database Schema
prisma// schema.prisma

model User {
  id            String    @id @default(cuid())
  email         String    @unique
  name          String?
  image         String?
  emailVerified DateTime?
  createdAt     DateTime  @default(now())
  updatedAt     DateTime  @updatedAt
  
  sessions      Session[]
  documents     Document[]
  participants  Participant[]
  accounts      Account[]
}

model Account {
  id                String  @id @default(cuid())
  userId            String
  type              String
  provider          String
  providerAccountId String
  refresh_token     String?
  access_token      String?
  expires_at        Int?
  token_type        String?
  scope             String?
  id_token          String?
  session_state     String?
  
  user User @relation(fields: [userId], references: [id], onDelete: Cascade)
  
  @@unique([provider, providerAccountId])
}

model Session {
  id              String   @id @default(cuid())
  code            String   @unique // 6-character code
  name            String
  description     String?
  creatorId       String
  latitude        Float
  longitude       Float
  radius          Int      @default(100) // meters
  expiresAt       DateTime
  createdAt       DateTime @default(now())
  
  creator         User     @relation(fields: [creatorId], references: [id])
  documents       Document[]
  participants    Participant[]
}

model Document {
  id          String   @id @default(cuid())
  filename    String
  mimetype    String
  size        Int
  path        String   // filesystem path
  uploaderId  String
  sessionId   String
  createdAt   DateTime @default(now())
  
  uploader    User     @relation(fields: [uploaderId], references: [id])
  session     Session  @relation(fields: [sessionId], references: [id], onDelete: Cascade)
}

model Participant {
  id        String   @id @default(cuid())
  userId    String
  sessionId String
  joinedAt  DateTime @default(now())
  latitude  Float?
  longitude Float?
  
  user      User     @relation(fields: [userId], references: [id])
  session   Session  @relation(fields: [sessionId], references: [id], onDelete: Cascade)
  
  @@unique([userId, sessionId])
}
5. API Design
Authentication Endpoints (Better Auth)
typescript// Handled by Better Auth
POST   /api/auth/signin/email     // Email/password login
POST   /api/auth/signup/email     // Email/password registration
GET    /api/auth/signin/google    // Google SSO
GET    /api/auth/signin/github    // GitHub SSO
POST   /api/auth/signout          // Logout
GET    /api/auth/session          // Get current session
Application Endpoints
typescript// Sessions
POST   /api/sessions              // Create new session
GET    /api/sessions              // List user's sessions
GET    /api/sessions/:code        // Get session by code
POST   /api/sessions/:code/join   // Join session (with location)
DELETE /api/sessions/:id          // Delete session

// Documents
POST   /api/documents             // Upload document
GET    /api/documents             // List user's documents
GET    /api/documents/:id         // Download document
DELETE /api/documents/:id         // Delete document

// Location
POST   /api/sessions/:code/verify-location  // Verify user location
6. Page Structure
app/
├── (auth)/
│   ├── login/
│   │   └── page.tsx          // Login with email/SSO
│   ├── register/
│   │   └── page.tsx          // Registration page
│   └── layout.tsx            // Auth layout
│
├── (app)/
│   ├── layout.tsx            // Authenticated layout
│   ├── page.tsx              // Dashboard/Home
│   ├── sessions/
│   │   ├── page.tsx          // My sessions list
│   │   ├── new/
│   │   │   └── page.tsx      // Create session
│   │   └── [code]/
│   │       └── page.tsx      // Session detail/documents
│   ├── documents/
│   │   └── page.tsx          // Document gallery
│   └── join/
│       └── page.tsx          // Join session by code
│
├── api/
│   ├── auth/
│   │   └── [...all]/route.ts // Better Auth handler
│   ├── sessions/
│   │   └── route.ts
│   └── documents/
│       └── route.ts
│
└── layout.tsx                // Root layout
7. Component Architecture
typescript// Key Components

components/
├── auth/
│   ├── LoginForm.tsx         // Email/password form
│   ├── SSOButtons.tsx        // Google/GitHub buttons
│   └── AuthGuard.tsx         // Route protection
│
├── sessions/
│   ├── SessionCard.tsx       // Session preview card
│   ├── SessionList.tsx       // List of sessions
│   ├── CreateSessionForm.tsx // New session form
│   └── JoinSessionForm.tsx   // Join with code + location
│
├── documents/
│   ├── DocumentGrid.tsx      // Masonry grid layout
│   ├── DocumentCard.tsx      // Document preview
│   ├── UploadDropzone.tsx    // Drag & drop upload
│   └── DocumentViewer.tsx    // Preview modal
│
└── shared/
    ├── Layout.tsx            // App shell
    ├── Navigation.tsx        // Mobile-friendly nav
    ├── LocationVerifier.tsx  // Geolocation component
    └── QRCode.tsx           // QR code generator
8. Mobile-First Design
Responsive Breakpoints
css/* TailwindCSS default breakpoints */
sm: 640px   /* Tablets */
md: 768px   /* Small laptops */
lg: 1024px  /* Desktop */
xl: 1280px  /* Large screens */
Mobile UI Patterns

Bottom Navigation: Fixed bottom nav for mobile
Swipe Gestures: Swipe to delete/archive
Touch Targets: Minimum 44x44px touch areas
Progressive Disclosure: Collapsible sections
Thumb-Friendly: Primary actions in bottom 60% of screen

9. Docker Configuration
Dockerfile
dockerfile# Multi-stage build
FROM node:20-alpine AS base

# Dependencies
FROM base AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci

# Builder
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Generate Prisma client
RUN npx prisma generate
RUN npm run build

# Runner
FROM base AS runner
WORKDIR /app

ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/prisma ./prisma

# Create upload directory
RUN mkdir -p /app/uploads && chown -R nextjs:nodejs /app/uploads

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
docker-compose.yml
yamlversion: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"
    environment:
      - DATABASE_URL=file:/app/data/database.db
      - BETTER_AUTH_SECRET=${BETTER_AUTH_SECRET}
      - BETTER_AUTH_URL=http://localhost:3000
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
      - GITHUB_CLIENT_ID=${GITHUB_CLIENT_ID}
      - GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
    volumes:
      - ./data:/app/data          # SQLite database
      - ./uploads:/app/uploads    # Document storage
    restart: unless-stopped

  # Development only
  dev:
    image: node:20-alpine
    working_dir: /app
    command: npm run dev
    ports:
      - "3000:3000"
    environment:
      - DATABASE_URL=file:/app/data/database.db
      - BETTER_AUTH_SECRET=dev-secret
    volumes:
      - .:/app
      - /app/node_modules
      - ./data:/app/data
      - ./uploads:/app/uploads
    profiles:
      - development
10. Security Considerations
Authentication

Password Requirements: Minimum 8 characters, complexity rules
Session Duration: 7-day refresh tokens, 1-hour access tokens
Rate Limiting: 5 login attempts per 15 minutes
CSRF Protection: Built into Better Auth

Data Protection

File Validation: MIME type checking, size limits (10MB)
Path Traversal: Sanitize file paths, use UUIDs
SQL Injection: Prisma ORM parameterized queries
XSS Prevention: React automatic escaping

Location Privacy

Fuzzing: Round coordinates to 3 decimal places
Opt-in: Explicit permission for location access
No Storage: Don't persist exact user locations

11. Performance Optimization
Frontend

Code Splitting: Dynamic imports for routes
Image Optimization: Next.js Image component
Lazy Loading: Intersection Observer for document grid
Caching: SWR for data fetching

Backend

Database Indexes: On frequently queried fields
Connection Pooling: Prisma connection management
Static Generation: Pre-render marketing pages
API Response Caching: Cache-Control headers

12. Deployment Strategy
Development
bash# Local development
docker-compose --profile development up

# Database migrations
docker exec -it groupup-dev npx prisma migrate dev
Production
bash# Build and run
docker-compose up -d

# Initialize database
docker exec -it groupup npx prisma migrate deploy
Environment Variables
env# .env.production
DATABASE_URL=file:/app/data/database.db
BETTER_AUTH_SECRET=<generated-secret>
BETTER_AUTH_URL=https://groupup.app
GOOGLE_CLIENT_ID=<oauth-client-id>
GOOGLE_CLIENT_SECRET=<oauth-secret>
GITHUB_CLIENT_ID=<oauth-client-id>
GITHUB_CLIENT_SECRET=<oauth-secret>
UPLOAD_DIR=/app/uploads
MAX_FILE_SIZE=10485760  # 10MB
13. MVP Feature Scope
Phase 1 (Week 1-2)

✅ Basic authentication (email/password)
✅ Create/join sessions with codes
✅ Upload documents
✅ SQLite database setup
✅ Docker configuration

Phase 2 (Week 3-4)

✅ SSO integration (Google/GitHub)
✅ Document grid gallery
✅ Mobile responsive design
✅ Location verification
✅ Session expiration

Post-MVP

⏳ Real-time updates (WebSockets)
⏳ File previews
⏳ Share via QR codes
⏳ Advanced search/filters
⏳ Admin dashboard

14. Testing Strategy
End-to-End Testing with Playwright
javascriptplaywright.config.ts
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
    { name: 'mobile', use: { ...devices['iPhone 13'] } },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
Key Test Scenarios
typescripte2e/auth.spec.ts
- User registration flow
- Login with email/password
- SSO authentication (Google/GitHub)
- Password reset flow
- Session persistence
- Logout functionality

e2e/sessions.spec.ts
- Create new session
- Join session with code
- Location verification
- Session expiration
- View session list
- Delete session

e2e/documents.spec.ts
- Upload single document
- Bulk upload documents
- Download document
- Delete document
- View document gallery
- File validation errors

e2e/mobile.spec.ts
- Mobile navigation
- Touch interactions
- Responsive layouts
- Gesture support
Unit Testing
javascript// Example unit test structure
tests/
├── api/
│   ├── auth.test.ts
│   ├── sessions.test.ts
│   └── documents.test.ts
├── components/
│   ├── SessionCard.test.tsx
│   ├── DocumentGrid.test.tsx
│   └── LocationVerifier.test.tsx
└── utils/
    ├── validation.test.ts
    └── fileHelpers.test.ts
CI/CD Integration
yaml.github/workflows/test.yml
name: Test Suite

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'npm'
      
      - name: Install dependencies
        run: npm ci
      
      - name: Run unit tests
        run: npm run test:unit
      
      - name: Install Playwright Browsers
        run: npx playwright install --with-deps
      
      - name: Run Playwright tests
        run: npm run test:e2e
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: playwright-report/
          retention-days: 30

This system design provides a solid foundation for the GroupUp MVP with a focus on simplicity, mobile-friendliness, comprehensive testing with Playwright, and quick deployment using Docker and SQLite.