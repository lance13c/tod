import { z } from 'zod';
import { 
  createTRPCRouter, 
  publicProcedure, 
  protectedProcedure 
} from '@/server/trpc';
import { TRPCError } from '@trpc/server';

const createOrgSchema = z.object({
  name: z.string().min(2).max(100),
  slug: z.string().min(2).max(50).regex(/^[a-z0-9-]+$/),
  description: z.string().max(500).optional(),
  website: z.string().url().optional().or(z.literal('')),
  industry: z.string().optional(),
  size: z.enum(['1-10', '11-50', '51-200', '201-500', '500+']).optional(),
  location: z.string().optional(),
  email: z.string().email().optional().or(z.literal('')),
  twitter: z.string().optional(),
  linkedin: z.string().optional(),
  github: z.string().optional(),
  tags: z.string().optional(),
  isPublic: z.boolean().default(false),
});

const updateOrgSchema = createOrgSchema.partial().extend({
  id: z.string(),
});

export const organizationRouter = createTRPCRouter({
  // Public procedures
  getPublicOrganizations: publicProcedure
    .input(z.object({
      limit: z.number().min(1).max(100).default(20),
      cursor: z.string().optional(),
      featured: z.boolean().optional(),
      search: z.string().optional(),
      industry: z.string().optional(),
      tags: z.string().optional(),
    }))
    .query(async ({ ctx, input }) => {
      const where: any = {
        isPublic: true,
      };

      if (input.featured) {
        where.featured = true;
      }

      if (input.search) {
        where.OR = [
          { name: { contains: input.search, mode: 'insensitive' } },
          { description: { contains: input.search, mode: 'insensitive' } },
          { tags: { contains: input.search, mode: 'insensitive' } },
        ];
      }

      if (input.industry) {
        where.industry = input.industry;
      }

      if (input.tags) {
        where.tags = { contains: input.tags, mode: 'insensitive' };
      }

      const organizations = await ctx.prisma.organization.findMany({
        where,
        take: input.limit + 1,
        cursor: input.cursor ? { id: input.cursor } : undefined,
        orderBy: [
          { featured: 'desc' },
          { verified: 'desc' },
          { viewCount: 'desc' },
        ],
        include: {
          owner: {
            select: {
              id: true,
              name: true,
              image: true,
            },
          },
          _count: {
            select: {
              members: true,
            },
          },
        },
      });

      let nextCursor: typeof input.cursor | undefined = undefined;
      if (organizations.length > input.limit) {
        const nextItem = organizations.pop();
        nextCursor = nextItem!.id;
      }

      return {
        organizations,
        nextCursor,
      };
    }),

  getOrganizationBySlug: publicProcedure
    .input(z.object({
      slug: z.string(),
    }))
    .query(async ({ ctx, input }) => {
      const organization = await ctx.prisma.organization.findUnique({
        where: { slug: input.slug },
        include: {
          owner: {
            select: {
              id: true,
              name: true,
              email: true,
              image: true,
            },
          },
          members: {
            include: {
              user: {
                select: {
                  id: true,
                  name: true,
                  email: true,
                  image: true,
                },
              },
            },
            orderBy: [
              { role: 'asc' },
              { joinedAt: 'asc' },
            ],
          },
          _count: {
            select: {
              members: true,
            },
          },
        },
      });

      if (!organization) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Organization not found',
        });
      }

      // Increment view count if public
      if (organization.isPublic) {
        await ctx.prisma.organization.update({
          where: { id: organization.id },
          data: { viewCount: { increment: 1 } },
        });
      }

      // Check if user has access to private org
      if (!organization.isPublic) {
        if (!ctx.user) {
          throw new TRPCError({
            code: 'UNAUTHORIZED',
            message: 'This organization is private',
          });
        }

        const isMember = organization.members.some(
          member => member.userId === ctx.user?.id
        );

        if (!isMember && organization.ownerId !== ctx.user.id) {
          throw new TRPCError({
            code: 'FORBIDDEN',
            message: 'You do not have access to this organization',
          });
        }
      }

      return organization;
    }),

  getIndustries: publicProcedure.query(async ({ ctx }) => {
    const industries = await ctx.prisma.organization.findMany({
      where: { 
        isPublic: true,
        industry: { not: null },
      },
      select: { industry: true },
      distinct: ['industry'],
    });

    return industries.map(i => i.industry).filter(Boolean);
  }),

  getPopularTags: publicProcedure.query(async ({ ctx }) => {
    const orgs = await ctx.prisma.organization.findMany({
      where: { 
        isPublic: true,
        tags: { not: null },
      },
      select: { tags: true },
    });

    // Parse and count tags
    const tagCounts = new Map<string, number>();
    orgs.forEach(org => {
      if (org.tags) {
        org.tags.split(',').forEach(tag => {
          const trimmed = tag.trim().toLowerCase();
          tagCounts.set(trimmed, (tagCounts.get(trimmed) || 0) + 1);
        });
      }
    });

    // Sort and return top tags
    return Array.from(tagCounts.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 20)
      .map(([tag, count]) => ({ tag, count }));
  }),

  // Protected procedures
  getMyOrganizations: protectedProcedure.query(async ({ ctx }) => {
    const organizations = await ctx.prisma.organization.findMany({
      where: {
        OR: [
          { ownerId: ctx.user.id },
          {
            members: {
              some: {
                userId: ctx.user.id,
              },
            },
          },
        ],
      },
      include: {
        _count: {
          select: {
            members: true,
          },
        },
      },
      orderBy: {
        createdAt: 'desc',
      },
    });

    return organizations;
  }),

  createOrganization: protectedProcedure
    .input(createOrgSchema)
    .mutation(async ({ ctx, input }) => {
      // Check if slug is unique
      const existing = await ctx.prisma.organization.findUnique({
        where: { slug: input.slug },
      });

      if (existing) {
        throw new TRPCError({
          code: 'CONFLICT',
          message: 'An organization with this slug already exists',
        });
      }

      // Create organization
      const organization = await ctx.prisma.organization.create({
        data: {
          ...input,
          ownerId: ctx.user.id,
          members: {
            create: {
              userId: ctx.user.id,
              role: 'owner',
            },
          },
        },
        include: {
          owner: {
            select: {
              id: true,
              name: true,
              email: true,
            },
          },
          _count: {
            select: {
              members: true,
            },
          },
        },
      });

      return organization;
    }),

  updateOrganization: protectedProcedure
    .input(updateOrgSchema)
    .mutation(async ({ ctx, input }) => {
      const { id, ...data } = input;

      // Check ownership
      const org = await ctx.prisma.organization.findUnique({
        where: { id },
        select: { ownerId: true },
      });

      if (!org) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Organization not found',
        });
      }

      if (org.ownerId !== ctx.user.id) {
        // Check if user is admin
        const member = await ctx.prisma.organizationMember.findUnique({
          where: {
            userId_organizationId: {
              userId: ctx.user.id,
              organizationId: id,
            },
          },
        });

        if (!member || member.role !== 'admin') {
          throw new TRPCError({
            code: 'FORBIDDEN',
            message: 'You do not have permission to update this organization',
          });
        }
      }

      // Check slug uniqueness if changing
      if (data.slug) {
        const existing = await ctx.prisma.organization.findFirst({
          where: {
            slug: data.slug,
            id: { not: id },
          },
        });

        if (existing) {
          throw new TRPCError({
            code: 'CONFLICT',
            message: 'An organization with this slug already exists',
          });
        }
      }

      return await ctx.prisma.organization.update({
        where: { id },
        data,
      });
    }),

  togglePublicVisibility: protectedProcedure
    .input(z.object({
      id: z.string(),
      isPublic: z.boolean(),
    }))
    .mutation(async ({ ctx, input }) => {
      // Check ownership
      const org = await ctx.prisma.organization.findUnique({
        where: { id: input.id },
        select: { ownerId: true },
      });

      if (!org) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Organization not found',
        });
      }

      if (org.ownerId !== ctx.user.id) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'Only the owner can change visibility settings',
        });
      }

      return await ctx.prisma.organization.update({
        where: { id: input.id },
        data: { isPublic: input.isPublic },
      });
    }),

  deleteOrganization: protectedProcedure
    .input(z.object({
      id: z.string(),
    }))
    .mutation(async ({ ctx, input }) => {
      // Check ownership
      const org = await ctx.prisma.organization.findUnique({
        where: { id: input.id },
        select: { ownerId: true },
      });

      if (!org) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Organization not found',
        });
      }

      if (org.ownerId !== ctx.user.id) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'Only the owner can delete the organization',
        });
      }

      await ctx.prisma.organization.delete({
        where: { id: input.id },
      });

      return { success: true };
    }),

  // Member management
  inviteMember: protectedProcedure
    .input(z.object({
      organizationId: z.string(),
      email: z.string().email(),
      role: z.enum(['admin', 'member']).default('member'),
    }))
    .mutation(async ({ ctx, input }) => {
      // Check if user is owner or admin
      const member = await ctx.prisma.organizationMember.findUnique({
        where: {
          userId_organizationId: {
            userId: ctx.user.id,
            organizationId: input.organizationId,
          },
        },
      });

      if (!member || (member.role !== 'owner' && member.role !== 'admin')) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'You do not have permission to invite members',
        });
      }

      // Find user by email
      const invitedUser = await ctx.prisma.user.findUnique({
        where: { email: input.email },
      });

      if (!invitedUser) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'User not found. They must have an account first.',
        });
      }

      // Check if already a member
      const existingMember = await ctx.prisma.organizationMember.findUnique({
        where: {
          userId_organizationId: {
            userId: invitedUser.id,
            organizationId: input.organizationId,
          },
        },
      });

      if (existingMember) {
        throw new TRPCError({
          code: 'CONFLICT',
          message: 'User is already a member of this organization',
        });
      }

      // Add member
      return await ctx.prisma.organizationMember.create({
        data: {
          userId: invitedUser.id,
          organizationId: input.organizationId,
          role: input.role,
        },
        include: {
          user: {
            select: {
              id: true,
              name: true,
              email: true,
            },
          },
        },
      });
    }),

  removeMember: protectedProcedure
    .input(z.object({
      organizationId: z.string(),
      userId: z.string(),
    }))
    .mutation(async ({ ctx, input }) => {
      // Check if user is owner or admin
      const member = await ctx.prisma.organizationMember.findUnique({
        where: {
          userId_organizationId: {
            userId: ctx.user.id,
            organizationId: input.organizationId,
          },
        },
      });

      if (!member || (member.role !== 'owner' && member.role !== 'admin')) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'You do not have permission to remove members',
        });
      }

      // Can't remove the owner
      const org = await ctx.prisma.organization.findUnique({
        where: { id: input.organizationId },
        select: { ownerId: true },
      });

      if (org?.ownerId === input.userId) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'Cannot remove the organization owner',
        });
      }

      await ctx.prisma.organizationMember.delete({
        where: {
          userId_organizationId: {
            userId: input.userId,
            organizationId: input.organizationId,
          },
        },
      });

      return { success: true };
    }),
});