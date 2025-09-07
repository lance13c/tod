# GroupUp MVP Sprint Plan - Updated Priority

## Sprint Goal
Build a group-based file sharing platform with organization branding, time-limited groups (4-hour expiration), and a comprehensive dashboard showing group history with differentiation between started and participated groups.

## Sprint Duration: 3 Weeks

---

## Week 1: Authentication & Core Group Infrastructure

### Day 1-2: Authentication Enhancement
**Priority: CRITICAL**
- [x] Magic link authentication as primary method
- [x] Email/password as secondary option
- [x] Auto-redirect to dashboard after sign-in
- [x] Remove username functionality completely
- [ ] Session management with Better Auth

**Acceptance Criteria:**
- Magic link is the first option users see
- Clean, spacious UI matching design mockup
- Automatic dashboard redirect after authentication
- No username fields anywhere in the UI

### Day 3-4: Dashboard & Group Foundation
**Priority: CRITICAL**
- [ ] Create dashboard page with group list
- [ ] Implement group model with UUID identification
- [ ] Setup UUID-based folder structure for file storage
- [ ] Create groups table with expiration tracking
- [ ] Differentiate "started by me" vs "participated" groups

**Acceptance Criteria:**
- Dashboard shows comprehensive group history
- Groups display with role badges (Creator/Participant)
- UUID folders created for each group
- Clear visual distinction between active and expired groups

### Day 5: Organization Management
**Priority: HIGH**
- [ ] Create organization model and API
- [ ] Build organization creation flow
- [ ] Implement logo upload (2MB limit)
- [ ] Add brand color selection
- [ ] Create organization selector component

**Acceptance Criteria:**
- Users can create and manage organizations
- Logo upload with proper validation
- Brand colors properly stored and displayed
- Organizations selectable during group creation

---

## Week 2: Group Features & File Management

### Day 6-7: Group Creation & Branding
**Priority: CRITICAL**
- [ ] Create group creation form with organization selection
- [ ] Implement 4-hour expiration timer
- [ ] Add organization branding display
- [ ] Build group card component with branding
- [ ] Location-based group creation

**Acceptance Criteria:**
- Groups show organization branding prominently
- 4-hour countdown timer visible
- Location captured at group creation
- Organization logo and colors applied to UI

### Day 8-9: File Storage System
**Priority: HIGH**
- [ ] Implement UUID-based file storage
- [ ] Create file upload to group folders
- [ ] Add creator badge for files
- [ ] Build file grid with metadata
- [ ] Implement file download system

**Acceptance Criteria:**
- Files stored in /app/groups/{uuid}/ structure
- Creator files marked with special icon
- File grid shows upload metadata
- Secure file access with group membership check

### Day 10: Group Expiration & Extensions
**Priority: HIGH**
- [ ] Implement 4-hour expiration logic
- [ ] Add extension capability (creator only)
- [ ] Create countdown timer component
- [ ] Build expiration warning notifications
- [ ] Archive expired groups (read-only)

**Acceptance Criteria:**
- Groups expire exactly after 4 hours
- Creators can extend up to 3 times
- Clear countdown display with warnings
- Expired groups remain accessible as archives

---

## Week 3: Enhanced Features & Polish

### Day 11-12: Group Participation
**Priority: MEDIUM**
- [ ] Build join group flow with location verification
- [ ] Create participant list view
- [ ] Add real-time member count
- [ ] Implement leave group functionality
- [ ] Build group discovery for nearby groups

**Acceptance Criteria:**
- Location verified when joining groups
- Participant list shows all members
- Real-time updates for member changes
- Clean leave group experience

### Day 13: Dashboard Enhancements
**Priority: MEDIUM**
- [ ] Add group filtering (Active/Expired/Started/Participated)
- [ ] Implement search by organization
- [ ] Create group statistics view
- [ ] Add quick action buttons
- [ ] Build group activity timeline

**Acceptance Criteria:**
- Multiple filter options work correctly
- Search finds groups by org name
- Statistics show meaningful metrics
- Quick actions for common tasks

### Day 14-15: Mobile Optimization & Testing
**Priority: HIGH**
- [ ] Optimize dashboard for mobile screens
- [ ] Ensure touch-friendly group cards
- [ ] Test file upload on mobile devices
- [ ] Verify organization branding on small screens
- [ ] Add PWA manifest for mobile install

**Acceptance Criteria:**
- Dashboard fully responsive
- All features work on mobile Safari/Chrome
- File upload smooth on mobile
- Organization branding scales properly

---

## Technical Implementation Details

### Database Schema (Priority Updates)
```prisma
model Organization {
  id          String   @id @default(cuid())
  name        String
  slug        String   @unique
  logoUrl     String?
  brandColor  String?
  groups      Group[]
}

model Group {
  id             String   @id @default(uuid())
  name           String
  organizationId String?
  creatorId      String
  expiresAt      DateTime
  extendedCount  Int      @default(0)
  storageFolder  String   @unique
  isActive       Boolean  @default(true)
  members        GroupMember[]
  files          GroupFile[]
}

model GroupMember {
  groupId   String
  userId    String
  role      String   // "creator" or "participant"
  joinedAt  DateTime @default(now())
}
```

### Priority API Endpoints
```typescript
// Groups - CRITICAL
POST   /api/groups                 // Create with org branding
GET    /api/groups                 // List with role filter
POST   /api/groups/:id/extend      // Extend expiration
POST   /api/groups/:id/join        // Join with location

// Organizations - HIGH
POST   /api/organizations          // Create organization
POST   /api/organizations/:id/logo // Upload logo
GET    /api/organizations          // List user's orgs

// Files - HIGH
POST   /api/groups/:id/files       // Upload to UUID folder
GET    /api/groups/:id/files       // List with creator badges
```

### Component Priority Structure
```
components/
├── dashboard/              // CRITICAL
│   ├── GroupList.tsx      // Started vs Participated
│   ├── GroupFilters.tsx   // Active/Expired filter
│   └── QuickActions.tsx   // Start new group button
├── groups/                 // CRITICAL
│   ├── GroupCard.tsx      // With org branding
│   ├── GroupTimer.tsx     // 4-hour countdown
│   ├── CreateGroupForm.tsx // With org selector
│   └── ExtendButton.tsx   // Creator-only extend
├── organizations/          // HIGH
│   ├── OrgSelector.tsx    // For group creation
│   ├── OrgBranding.tsx    // Logo and color display
│   └── OrgLogoUpload.tsx  // 2MB limit, PNG/JPG/SVG
└── files/                  // HIGH
    ├── FileGrid.tsx       // With creator badges
    └── FileUpload.tsx     // To UUID folders
```

---

## Updated Definition of Done
- [ ] Dashboard shows all groups with role differentiation
- [ ] Organization branding visible on groups
- [ ] 4-hour expiration with extension working
- [ ] UUID-based file storage implemented
- [ ] Mobile responsive dashboard
- [ ] All authentication flows redirect to dashboard
- [ ] Playwright tests for critical paths
- [ ] No console errors in production

## Success Metrics
- Users can create branded groups in < 30 seconds
- Dashboard loads group history in < 2 seconds
- 100% of files stored in correct UUID folders
- Organization logos display correctly at all sizes
- Group expiration accurate within 1 minute
- Clear visual distinction between roles

## Risk Mitigation
- **UUID Collisions**: Use crypto.randomUUID() for guaranteed uniqueness
- **Storage Scaling**: Implement folder cleanup for old groups
- **Timer Accuracy**: Use server-side expiration checks
- **Brand Image Size**: Resize logos on upload, store multiple sizes
- **Permission Issues**: Strict role checking at API level

## Next Sprint Preview
- WebRTC peer-to-peer file transfer
- Real-time updates with WebSockets
- QR codes for group sharing
- Advanced file previews
- Group templates for organizations
- Analytics dashboard for group creators
- Scheduled/recurring groups
- API for third-party integrations