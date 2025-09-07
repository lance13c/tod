GroupUp MVP - System Design Document
1. System Overview
GroupUp MVP is a simplified proximity-based sharing platform that allows users and organizations to create time-limited sharing groups at specific locations.

Core Features (MVP)

Email/magic link authentication (prioritizing magic link)
Create location-based sharing groups with organization branding
View history of groups (started vs participated)
4-hour group expiration with extension capability
Organization branding (name, logo) visible to group participants
UUID-based file storage folders per group
WebRTC direct file sharing (peer-to-peer)
Building detection using Microsoft Building Footprints GeoJSON
Grid-based document/photo gallery
Mobile-responsive design
Guest access for quick sharing without login 

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
  groups        Group[]        // Groups created by user
  groupMembers  GroupMember[]  // Groups participated in
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

model Organization {
  id              String   @id @default(cuid())
  name            String
  slug            String   @unique
  logoUrl         String?  // URL to organization logo
  brandColor      String?  // Hex color for branding
  description     String?
  website         String?
  createdAt       DateTime @default(now())
  updatedAt       DateTime @updatedAt
  
  groups          Group[]
}

model Group {
  id              String   @id @default(uuid()) // UUID for folder naming
  name            String
  description     String?
  organizationId  String?
  creatorId       String
  latitude        Float
  longitude       Float
  radius          Int      @default(100) // meters
  expiresAt       DateTime // 4 hours from creation by default
  extendedCount   Int      @default(0) // Track number of extensions
  storageFolder   String   @unique // UUID folder path
  isActive        Boolean  @default(true)
  createdAt       DateTime @default(now())
  updatedAt       DateTime @updatedAt
  
  creator         User          @relation(fields: [creatorId], references: [id])
  organization    Organization? @relation(fields: [organizationId], references: [id])
  members         GroupMember[]
  files           GroupFile[]
}

model GroupMember {
  id              String   @id @default(cuid())
  groupId         String
  userId          String
  role            String   @default("participant") // "creator" or "participant"
  joinedAt        DateTime @default(now())
  latitude        Float?   // Location when joined
  longitude       Float?
  
  group           Group    @relation(fields: [groupId], references: [id], onDelete: Cascade)
  user            User     @relation(fields: [userId], references: [id])
  
  @@unique([groupId, userId])
}

model GroupFile {
  id              String   @id @default(cuid())
  filename        String
  originalName    String
  mimetype        String
  size            Int
  path            String   // Path within UUID folder
  uploaderId      String
  groupId         String
  isFromCreator   Boolean  // Mark files from group creator
  createdAt       DateTime @default(now())
  
  group           Group    @relation(fields: [groupId], references: [id], onDelete: Cascade)
}

// Legacy Session model (kept for compatibility)
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
typescript// Groups
POST   /api/groups                 // Create new group
GET    /api/groups                 // List user's groups (created & participated)
GET    /api/groups/:id             // Get group details
POST   /api/groups/:id/join        // Join group (with location verification)
POST   /api/groups/:id/extend      // Extend group expiration (creator only)
DELETE /api/groups/:id             // Delete group (creator only)

// Organizations
POST   /api/organizations          // Create organization
GET    /api/organizations          // List organizations
GET    /api/organizations/:slug    // Get organization by slug
PUT    /api/organizations/:id      // Update organization (branding, etc.)
POST   /api/organizations/:id/logo // Upload organization logo

// Group Files
POST   /api/groups/:id/files       // Upload file to group
GET    /api/groups/:id/files       // List group files
GET    /api/files/:id              // Download file
DELETE /api/files/:id              // Delete file (uploader only)

// Location
POST   /api/groups/:id/verify-location  // Verify user location for group

// Legacy Sessions (kept for compatibility)
POST   /api/sessions              // Create new session
GET    /api/sessions              // List user's sessions
GET    /api/sessions/:code        // Get session by code
POST   /api/sessions/:code/join   // Join session (with location)
DELETE /api/sessions/:id          // Delete session
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
│   ├── dashboard/
│   │   └── page.tsx          // Dashboard with groups list
│   ├── groups/
│   │   ├── page.tsx          // My groups list
│   │   ├── new/
│   │   │   └── page.tsx      // Create group with organization
│   │   └── [id]/
│   │       └── page.tsx      // Group detail/files
│   ├── organizations/
│   │   ├── page.tsx          // Organizations list
│   │   ├── new/
│   │   │   └── page.tsx      // Create organization
│   │   └── [slug]/
│   │       └── page.tsx      // Organization detail
│   ├── sessions/
│   │   ├── page.tsx          // Legacy sessions list
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
├── groups/
│   ├── GroupCard.tsx         // Group preview with org branding
│   ├── GroupList.tsx         // List with started/participated filter
│   ├── CreateGroupForm.tsx   // New group with organization select
│   ├── JoinGroupForm.tsx     // Join group with location
│   ├── GroupTimer.tsx        // 4-hour countdown with extend option
│   └── GroupBranding.tsx     // Organization branding display
│
├── organizations/
│   ├── OrgCard.tsx           // Organization card with logo
│   ├── OrgSelector.tsx       // Organization dropdown/selector
│   ├── OrgLogoUpload.tsx     // Logo upload component
│   └── OrgBrandingForm.tsx   // Edit org branding
│
├── files/
│   ├── FileGrid.tsx          // Grid layout with creator badges
│   ├── FileCard.tsx          // File preview with metadata
│   ├── FileUploadZone.tsx    // Drag & drop for group files
│   └── FileViewer.tsx        // Preview modal
│
├── sessions/
│   ├── SessionCard.tsx       // Legacy session card
│   ├── SessionList.tsx       // Legacy sessions list
│   ├── CreateSessionForm.tsx // Legacy session form
│   └── JoinSessionForm.tsx   // Legacy join form
│
├── documents/
│   ├── DocumentGrid.tsx      // Legacy document grid
│   ├── DocumentCard.tsx      // Legacy document card
│   ├── UploadDropzone.tsx    // Legacy upload
│   └── DocumentViewer.tsx    // Legacy preview
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
GROUP_STORAGE_DIR=/app/groups  # UUID-based group folders
MAX_FILE_SIZE=10485760  # 10MB
GROUP_EXPIRATION_HOURS=4  # Default group expiration
MAX_GROUP_EXTENSIONS=3  # Maximum times a group can be extended
ORG_LOGO_MAX_SIZE=2097152  # 2MB for organization logos
13. MVP Feature Scope
Phase 1 (Week 1-2)

✅ Magic link authentication (primary)
✅ Email/password authentication (secondary)
✅ Group creation with organization branding
✅ UUID-based file storage folders
✅ 4-hour group expiration
✅ Dashboard with group history

Phase 2 (Week 3-4)

✅ Organization management (logos, branding)
✅ Group extension capability
✅ File grid with creator badges
✅ Mobile responsive design
✅ Location verification
✅ Differentiate started vs participated groups

Phase 3 (Week 5-6)

⏳ WebRTC peer-to-peer file sharing
⏳ Real-time updates (WebSockets)
⏳ File previews and thumbnails
⏳ Share groups via QR codes
⏳ Advanced search/filters
⏳ Analytics dashboard

Post-MVP

⏳ Multiple organization membership
⏳ Group templates
⏳ Scheduled groups
⏳ Admin dashboard
⏳ API for third-party integrations

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

15. Group Features Implementation Details

Group Lifecycle

1. **Creation**: User selects organization (optional), sets location, creates group
2. **UUID Generation**: System generates UUID for group ID and storage folder
3. **Expiration Timer**: 4-hour countdown starts upon creation
4. **Extensions**: Creator can extend up to 3 times (4 hours each)
5. **Archival**: Expired groups become read-only, files remain accessible

Organization Branding

- **Logo Requirements**: Max 2MB, supports PNG/JPG/SVG
- **Brand Colors**: Hex color codes for consistent theming
- **Display**: Organization branding shown prominently in group UI
- **Verification**: Optional organization verification for trusted brands

File Storage Structure

bash/app/groups/
├── 550e8400-e29b-41d4-a716-446655440000/  # Group UUID folder
│   ├── metadata.json                       # Group metadata
│   ├── files/
│   │   ├── user1_file1.pdf
│   │   ├── user2_image.jpg
│   │   └── creator_document.docx
│   └── thumbnails/                        # Generated thumbnails
│       ├── user2_image_thumb.jpg
│       └── ...
├── 6ba7b810-9dad-11d1-80b4-00c04fd430c8/
│   └── ...

Dashboard Features

- **Group Filters**: Active, Expired, Started by Me, Participated
- **Search**: By organization, location, date range
- **Quick Actions**: Start new group, rejoin active group, view files
- **Statistics**: Total groups, files shared, active participants

Group Permissions

| Action | Creator | Participant | Guest |
|--------|---------|-------------|-------|
| View group | ✓ | ✓ | ✓ (with link) |
| Upload files | ✓ | ✓ | ✗ |
| Delete own files | ✓ | ✓ | ✗ |
| Delete any files | ✓ | ✗ | ✗ |
| Extend expiration | ✓ | ✗ | ✗ |
| Edit group info | ✓ | ✗ | ✗ |
| End group early | ✓ | ✗ | ✗ |

API Response Examples

typescript// GET /api/groups response
{
  "started": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Team Standup",
      "organization": {
        "name": "Acme Corp",
        "logoUrl": "/logos/acme.png",
        "brandColor": "#FF6B6B"
      },
      "expiresAt": "2025-01-06T16:00:00Z",
      "memberCount": 5,
      "fileCount": 12,
      "role": "creator"
    }
  ],
  "participated": [
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "name": "Design Review",
      "organization": null,
      "expiresAt": "2025-01-06T14:30:00Z",
      "memberCount": 3,
      "fileCount": 8,
      "role": "participant"
    }
  ]
}

WebRTC Integration (Future)

- **Signaling Server**: Coordinate peer connections
- **STUN/TURN**: Handle NAT traversal
- **Chunking**: Large file transfer optimization
- **Fallback**: Server relay for failed P2P connections

This system design provides a solid foundation for the GroupUp MVP with organization branding, time-limited groups, UUID-based storage, and clear differentiation between group creators and participants. The architecture supports future expansion with WebRTC peer-to-peer sharing while maintaining a simple, mobile-friendly user experience.