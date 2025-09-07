import { PrismaClient } from '@prisma/client';
import { hashPassword } from 'better-auth/crypto';

const prisma = new PrismaClient();

async function main() {
  console.log('🌱 Starting database seed...');

  // Clean existing data (optional - comment out if you want to preserve data)
  await prisma.groupFile.deleteMany();
  await prisma.groupMember.deleteMany();
  await prisma.group.deleteMany();
  await prisma.organizationMember.deleteMany();
  await prisma.organization.deleteMany();
  await prisma.verification.deleteMany();
  await prisma.session.deleteMany();
  await prisma.account.deleteMany();
  await prisma.user.deleteMany();

  console.log('🧹 Cleaned existing data');

  // Create test users
  const hashedPassword = await hashPassword('Password123!');

  // Test User 1 - Regular user with email/password
  const testUser1 = await prisma.user.create({
    data: {
      id: 'test-user-1',
      email: 'test@example.com',
      emailVerified: true,
      name: 'Test User',
      username: 'testuser',
      password: hashedPassword,
      image: null,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  // Create account for test user 1
  await prisma.account.create({
    data: {
      id: 'test-account-1',
      accountId: 'test@example.com',
      providerId: 'credential',
      userId: testUser1.id,
      accessToken: null,
      refreshToken: null,
      idToken: null,
      accessTokenExpiresAt: null,
      refreshTokenExpiresAt: null,
      scope: null,
      password: hashedPassword,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  console.log('✅ Created test user 1:', testUser1.email);

  // Test User 2 - User with organizations
  const testUser2 = await prisma.user.create({
    data: {
      id: 'test-user-2',
      email: 'john@example.com',
      emailVerified: true,
      name: 'John Doe',
      username: 'johndoe',
      password: hashedPassword,
      image: 'https://api.dicebear.com/7.x/avataaars/svg?seed=johndoe',
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.account.create({
    data: {
      id: 'test-account-2',
      accountId: 'john@example.com',
      providerId: 'credential',
      userId: testUser2.id,
      accessToken: null,
      refreshToken: null,
      idToken: null,
      accessTokenExpiresAt: null,
      refreshTokenExpiresAt: null,
      scope: null,
      password: hashedPassword,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  console.log('✅ Created test user 2:', testUser2.email);

  // Test User 3 - Admin user
  const adminUser = await prisma.user.create({
    data: {
      id: 'admin-user',
      email: 'admin@example.com',
      emailVerified: true,
      name: 'Admin User',
      username: 'admin',
      password: hashedPassword,
      image: 'https://api.dicebear.com/7.x/avataaars/svg?seed=admin',
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.account.create({
    data: {
      id: 'admin-account',
      accountId: 'admin@example.com',
      providerId: 'credential',
      userId: adminUser.id,
      accessToken: null,
      refreshToken: null,
      idToken: null,
      accessTokenExpiresAt: null,
      refreshTokenExpiresAt: null,
      scope: null,
      password: hashedPassword,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  console.log('✅ Created admin user:', adminUser.email);

  // Create test organizations with branding
  const org1 = await prisma.organization.create({
    data: {
      id: 'org-1',
      name: 'Acme Corporation',
      slug: 'acme-corp',
      description: 'Leading provider of innovative solutions for modern businesses.',
      isPublic: true,
      featured: true,
      verified: true,
      industry: 'Technology',
      size: '100-500',
      location: 'San Francisco, CA',
      website: 'https://acme.example.com',
      email: 'contact@acme.example.com',
      github: 'acme-corp',
      twitter: 'acmecorp',
      linkedin: 'acme-corporation',
      tags: 'saas,b2b,enterprise,cloud',
      logo: 'https://api.dicebear.com/7.x/identicon/svg?seed=acme',
      logoUrl: 'https://api.dicebear.com/7.x/identicon/svg?seed=acme',
      brandColor: '#FF6B6B',
      founded: '2015',
      viewCount: 1250,
      ownerId: testUser2.id,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  // Add members to org1
  await prisma.organizationMember.create({
    data: {
      id: 'member-1',
      userId: testUser2.id,
      organizationId: org1.id,
      role: 'owner',
      title: 'CEO & Founder',
      joinedAt: new Date(),
    }
  });

  await prisma.organizationMember.create({
    data: {
      id: 'member-2',
      userId: adminUser.id,
      organizationId: org1.id,
      role: 'admin',
      title: 'CTO',
      joinedAt: new Date(),
    }
  });

  console.log('✅ Created organization 1:', org1.name);

  const org2 = await prisma.organization.create({
    data: {
      id: 'org-2',
      name: 'TechStart Inc',
      slug: 'techstart',
      description: 'Empowering startups with cutting-edge technology solutions.',
      isPublic: true,
      featured: false,
      verified: true,
      industry: 'Technology',
      size: '10-50',
      location: 'Austin, TX',
      website: 'https://techstart.example.com',
      email: 'hello@techstart.example.com',
      github: 'techstart',
      tags: 'startup,innovation,ai,ml',
      logo: 'https://api.dicebear.com/7.x/identicon/svg?seed=techstart',
      founded: '2020',
      viewCount: 450,
      ownerId: testUser2.id,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.organizationMember.create({
    data: {
      id: 'member-3',
      userId: testUser2.id,
      organizationId: org2.id,
      role: 'admin',
      title: 'Co-Founder',
      joinedAt: new Date(),
    }
  });

  console.log('✅ Created organization 2:', org2.name);

  const org3 = await prisma.organization.create({
    data: {
      id: 'org-3',
      name: 'Private Ventures',
      slug: 'private-ventures',
      description: 'Exclusive investment opportunities for qualified investors.',
      isPublic: false, // Private organization
      featured: false,
      verified: false,
      industry: 'Finance',
      size: '1-10',
      location: 'New York, NY',
      tags: 'finance,investment,private',
      viewCount: 0,
      ownerId: testUser2.id,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.organizationMember.create({
    data: {
      id: 'member-4',
      userId: testUser2.id,
      organizationId: org3.id,
      role: 'owner',
      title: 'Managing Partner',
      joinedAt: new Date(),
    }
  });

  console.log('✅ Created organization 3 (private):', org3.name);

  const org4 = await prisma.organization.create({
    data: {
      id: 'org-4',
      name: 'Green Energy Solutions',
      slug: 'green-energy',
      description: 'Sustainable energy solutions for a better tomorrow.',
      isPublic: true,
      featured: true,
      verified: false,
      industry: 'Energy',
      size: '50-100',
      location: 'Seattle, WA',
      website: 'https://greenenergy.example.com',
      email: 'info@greenenergy.example.com',
      twitter: 'greenenergysol',
      tags: 'sustainability,renewable,solar,wind',
      logo: 'https://api.dicebear.com/7.x/identicon/svg?seed=greenenergy',
      founded: '2018',
      viewCount: 780,
      ownerId: adminUser.id,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.organizationMember.create({
    data: {
      id: 'member-5',
      userId: adminUser.id,
      organizationId: org4.id,
      role: 'member',
      title: 'Sustainability Advisor',
      joinedAt: new Date(),
    }
  });

  console.log('✅ Created organization 4:', org4.name);

  const org5 = await prisma.organization.create({
    data: {
      id: 'org-5',
      name: 'Healthcare Innovations',
      slug: 'healthcare-innovations',
      description: 'Revolutionizing healthcare through technology and innovation.',
      isPublic: true,
      featured: false,
      verified: true,
      industry: 'Healthcare',
      size: '500-1000',
      location: 'Boston, MA',
      website: 'https://healthinnovate.example.com',
      github: 'healthinnovate',
      linkedin: 'healthcare-innovations',
      tags: 'healthcare,medtech,biotech,research',
      logo: 'https://api.dicebear.com/7.x/identicon/svg?seed=healthcare',
      founded: '2010',
      viewCount: 2100,
      ownerId: testUser1.id,
      createdAt: new Date(),
      updatedAt: new Date(),
    }
  });

  await prisma.organizationMember.create({
    data: {
      id: 'member-6',
      userId: testUser1.id,
      organizationId: org5.id,
      role: 'member',
      title: 'Research Scientist',
      joinedAt: new Date(),
    }
  });

  console.log('✅ Created organization 5:', org5.name);

  // Create some additional organizations for pagination testing
  const industries = ['Technology', 'Finance', 'Healthcare', 'Energy', 'Education', 'Retail'];
  const sizes = ['1-10', '10-50', '50-100', '100-500', '500-1000', '1000+'];
  
  for (let i = 6; i <= 15; i++) {
    const org = await prisma.organization.create({
      data: {
        id: `org-${i}`,
        name: `Test Organization ${i}`,
        slug: `test-org-${i}`,
        description: `This is test organization number ${i} for testing pagination and filtering.`,
        isPublic: true,
        featured: i % 3 === 0, // Every 3rd org is featured
        verified: i % 2 === 0, // Every even org is verified
        industry: industries[i % industries.length],
        size: sizes[i % sizes.length],
        location: `City ${i}, State`,
        tags: `test,org${i},sample`,
        viewCount: Math.floor(Math.random() * 1000),
        ownerId: testUser1.id,
        createdAt: new Date(),
        updatedAt: new Date(),
      }
    });
    console.log(`✅ Created test organization ${i}:`, org.name);
  }

  // Create test sessions for authenticated users
  const session1 = await prisma.session.create({
    data: {
      id: 'test-session-1',
      expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days from now
      token: 'test-token-1',
      createdAt: new Date(),
      updatedAt: new Date(),
      ipAddress: '127.0.0.1',
      userAgent: 'Playwright Test Browser',
      userId: testUser1.id,
    }
  });

  console.log('✅ Created test session for user 1');

  const session2 = await prisma.session.create({
    data: {
      id: 'test-session-2',
      expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days from now
      token: 'test-token-2',
      createdAt: new Date(),
      updatedAt: new Date(),
      ipAddress: '127.0.0.1',
      userAgent: 'Playwright Test Browser',
      userId: testUser2.id,
    }
  });

  console.log('✅ Created test session for user 2');

  // Create test groups with organization branding
  const group1 = await prisma.group.create({
    data: {
      id: '550e8400-e29b-41d4-a716-446655440000',
      name: 'Team Standup - Q1 Planning',
      description: 'Quarterly planning session for the engineering team',
      organizationId: org1.id,
      creatorId: testUser2.id,
      latitude: 37.7749,
      longitude: -122.4194,
      radius: 100,
      expiresAt: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours from now (active)
      extendedCount: 0,
      storageFolder: '550e8400-e29b-41d4-a716-446655440000',
      isActive: true,
      createdAt: new Date(Date.now() - 60 * 60 * 1000), // Created 1 hour ago
    }
  });

  // Add members to group1
  await prisma.groupMember.create({
    data: {
      groupId: group1.id,
      userId: testUser2.id,
      role: 'creator',
      joinedLatitude: 37.7749,
      joinedLongitude: -122.4194,
    }
  });

  await prisma.groupMember.create({
    data: {
      groupId: group1.id,
      userId: adminUser.id,
      role: 'participant',
      joinedLatitude: 37.7750,
      joinedLongitude: -122.4195,
    }
  });

  await prisma.groupMember.create({
    data: {
      groupId: group1.id,
      userId: testUser1.id,
      role: 'participant',
      joinedLatitude: 37.7748,
      joinedLongitude: -122.4193,
    }
  });

  console.log('✅ Created active group 1:', group1.name);

  const group2 = await prisma.group.create({
    data: {
      id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
      name: 'Design Review Session',
      description: 'Review of new UI designs for the mobile app',
      organizationId: org2.id,
      creatorId: adminUser.id,
      latitude: 30.2672,
      longitude: -97.7431,
      radius: 100,
      expiresAt: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago (expired)
      extendedCount: 2,
      storageFolder: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
      isActive: false,
      isArchived: true,
      createdAt: new Date(Date.now() - 8 * 60 * 60 * 1000), // Created 8 hours ago
    }
  });

  // Add members to group2
  await prisma.groupMember.create({
    data: {
      groupId: group2.id,
      userId: adminUser.id,
      role: 'creator',
      joinedLatitude: 30.2672,
      joinedLongitude: -97.7431,
    }
  });

  await prisma.groupMember.create({
    data: {
      groupId: group2.id,
      userId: testUser2.id,
      role: 'participant',
      joinedLatitude: 30.2673,
      joinedLongitude: -97.7432,
    }
  });

  console.log('✅ Created expired group 2:', group2.name);

  const group3 = await prisma.group.create({
    data: {
      id: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
      name: 'Coffee Break Meetup',
      description: 'Informal team gathering',
      organizationId: null, // No organization
      creatorId: testUser1.id,
      latitude: 40.7128,
      longitude: -74.0060,
      radius: 50,
      expiresAt: new Date(Date.now() + 3 * 60 * 60 * 1000), // 3 hours from now
      extendedCount: 0,
      storageFolder: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
      isActive: true,
      createdAt: new Date(Date.now() - 30 * 60 * 1000), // Created 30 minutes ago
    }
  });

  // Add members to group3
  await prisma.groupMember.create({
    data: {
      groupId: group3.id,
      userId: testUser1.id,
      role: 'creator',
      joinedLatitude: 40.7128,
      joinedLongitude: -74.0060,
    }
  });

  console.log('✅ Created group 3 (no org):', group3.name);

  // Create sample files for group1
  await prisma.groupFile.create({
    data: {
      filename: 'q1-planning.pdf',
      originalName: 'Q1 Planning Document.pdf',
      mimetype: 'application/pdf',
      size: 2048576, // 2MB
      path: '550e8400-e29b-41d4-a716-446655440000/q1-planning.pdf',
      uploaderId: testUser2.id,
      groupId: group1.id,
      isFromCreator: true,
    }
  });

  await prisma.groupFile.create({
    data: {
      filename: 'roadmap.png',
      originalName: 'Product Roadmap 2025.png',
      mimetype: 'image/png',
      size: 1024768,
      path: '550e8400-e29b-41d4-a716-446655440000/roadmap.png',
      uploaderId: adminUser.id,
      groupId: group1.id,
      isFromCreator: false,
      thumbnailPath: '550e8400-e29b-41d4-a716-446655440000/thumbnails/roadmap_thumb.png',
    }
  });

  console.log('✅ Created sample files for group 1');

  console.log('\n🎉 Database seeding completed successfully!');
  console.log('\n📝 Test Credentials:');
  console.log('───────────────────────────────────────');
  console.log('Email: test@example.com | Username: testuser | Password: Password123!');
  console.log('Email: john@example.com | Username: johndoe | Password: Password123!');
  console.log('Email: admin@example.com | Username: admin | Password: Password123!');
  console.log('───────────────────────────────────────');
  console.log('\n🏢 Test Organizations:');
  console.log('- Acme Corporation (public, featured, verified, brand: #FF6B6B)');
  console.log('- TechStart Inc (public, verified)');
  console.log('- Private Ventures (private)');
  console.log('- Green Energy Solutions (public, featured)');
  console.log('- Healthcare Innovations (public, verified)');
  console.log('- Plus 10 more test organizations for pagination');
  console.log('\n👥 Test Groups:');
  console.log('- Team Standup - Q1 Planning (Active, Acme Corp, expires in 2h)');
  console.log('- Design Review Session (Expired/Archived, TechStart Inc)');
  console.log('- Coffee Break Meetup (Active, No org, expires in 3h)');
}

main()
  .then(async () => {
    await prisma.$disconnect();
  })
  .catch(async (e) => {
    console.error('❌ Error seeding database:', e);
    await prisma.$disconnect();
    process.exit(1);
  });