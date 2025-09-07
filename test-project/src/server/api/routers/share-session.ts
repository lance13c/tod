import { z } from 'zod';
import { createTRPCRouter, publicProcedure, protectedProcedure } from '@/server/trpc';
import { TRPCError } from '@trpc/server';
import { 
  generateSessionCode, 
  isWithinGeoLock,
  findBuildingAtLocation,
  calculateBoundingBox,
  isPointInBoundingBox
} from '@/lib/geo/building-detector';
import { findNearestBuilding } from '@/lib/geo/duckdb-building-utils';
import { isWithinCoverage, getSampleLocation } from '@/lib/geo/coverage';
import crypto from 'crypto';

export const shareSessionRouter = createTRPCRouter({
  // Find building at user's location
  findBuildingAtLocation: publicProcedure
    .input(z.object({
      latitude: z.number().min(-90).max(90),
      longitude: z.number().min(-180).max(180),
    }))
    .query(async ({ ctx, input }) => {
      try {
        // First check if location is within dataset coverage
        const coverage = await isWithinCoverage(input.latitude, input.longitude);
        
        if (!coverage.isWithin) {
          const sampleLocation = await getSampleLocation();
          return {
            building: null,
            activeSession: null,
            coverage: {
              isWithinDataset: false,
              message: coverage.suggestion,
              sampleLocation: sampleLocation ? {
                latitude: sampleLocation.latitude,
                longitude: sampleLocation.longitude,
                description: 'Use these coordinates to test the system'
              } : null
            }
          };
        }
        
        // Use DuckDB to find the nearest building
        const building = await findNearestBuilding(
          input.latitude,
          input.longitude,
          50 // 50 meters buffer
        );

        if (building && building.isInside) {
          // Check if there's already an active session for this building
          // First, check if this building exists in Prisma
          let prismaBuilding = await ctx.prisma.building.findUnique({
            where: { id: building.id }
          });

          // If not, create it in Prisma for session tracking
          if (!prismaBuilding) {
            prismaBuilding = await ctx.prisma.building.create({
              data: {
                id: building.id,
                name: building.name,
                address: building.address,
                polygon: JSON.stringify(building.geometry),
                bbox: JSON.stringify({
                  minLat: building.centroid[1] - 0.001,
                  maxLat: building.centroid[1] + 0.001,
                  minLng: building.centroid[0] - 0.001,
                  maxLng: building.centroid[0] + 0.001,
                }),
              }
            });
          }

          const existingSession = await ctx.prisma.shareSession.findFirst({
            where: {
              buildingId: building.id,
              expiresAt: { gt: new Date() },
            },
            include: {
              _count: {
                select: { participants: true }
              }
            },
            orderBy: {
              createdAt: 'desc'
            }
          });

          return {
            building: {
              id: building.id,
              name: building.name || 'Unknown Building',
              distance: building.distance,
              isInside: building.isInside,
              centroid: building.centroid,
              geometry: building.geometry,
            },
            existingSession: existingSession ? {
              id: existingSession.id,
              code: existingSession.code,
              name: existingSession.name,
              participantCount: existingSession._count.participants,
              expiresAt: existingSession.expiresAt,
            } : null,
            coverage: {
              isWithinDataset: true,
              message: 'Location is within dataset coverage'
            }
          };
        }

        return {
          building: building ? {
            id: building.id,
            name: building.name || 'Unknown Building',
            distance: building.distance,
            isInside: building.isInside,
            centroid: building.centroid,
            geometry: building.geometry,
          } : null,
          existingSession: null,
          coverage: {
            isWithinDataset: true,
            message: building ? 'Building found nearby' : 'No building found at this location'
          }
        };
      } catch (error) {
        console.error('Error finding building at location:', error);
        return {
          building: null,
          existingSession: null,
          coverage: {
            isWithinDataset: false,
            message: 'Error checking location',
            error: error instanceof Error ? error.message : 'Unknown error'
          }
        };
      }
    }),
  // Create a new share session
  createSession: publicProcedure
    .input(z.object({
      name: z.string().min(1).max(100),
      description: z.string().optional(),
      latitude: z.number().min(-90).max(90),
      longitude: z.number().min(-180).max(180),
      geoLockRadius: z.number().min(10).max(1000).default(100),
      maxParticipants: z.number().min(2).max(50).default(10),
      expiresInHours: z.number().min(1).max(24).default(4),
      requiresAuth: z.boolean().default(false),
      isGuest: z.boolean().default(false),
      guestFingerprint: z.string().optional(),
    }))
    .mutation(async ({ ctx, input }) => {
      // Generate unique session code
      let code: string;
      let codeExists = true;
      let attempts = 0;
      
      while (codeExists && attempts < 10) {
        code = generateSessionCode();
        const existing = await ctx.prisma.shareSession.findUnique({
          where: { code }
        });
        codeExists = !!existing;
        attempts++;
      }
      
      if (codeExists) {
        throw new TRPCError({
          code: 'INTERNAL_SERVER_ERROR',
          message: 'Could not generate unique session code'
        });
      }

      // Find building at location using DuckDB
      const buildingInfo = await findNearestBuilding(
        input.latitude,
        input.longitude,
        50 // 50 meters buffer
      );

      let buildingId: string | null = null;
      
      if (buildingInfo && buildingInfo.isInside) {
        // Check if building exists in Prisma, create if not
        let prismaBuilding = await ctx.prisma.building.findUnique({
          where: { id: buildingInfo.id }
        });

        if (!prismaBuilding) {
          prismaBuilding = await ctx.prisma.building.create({
            data: {
              id: buildingInfo.id,
              name: buildingInfo.name,
              address: buildingInfo.address,
              polygon: JSON.stringify(buildingInfo.geometry),
              bbox: JSON.stringify({
                minLat: buildingInfo.centroid[1] - 0.001,
                maxLat: buildingInfo.centroid[1] + 0.001,
                minLng: buildingInfo.centroid[0] - 0.001,
                maxLng: buildingInfo.centroid[0] + 0.001,
              }),
            }
          });
        }
        
        buildingId = buildingInfo.id;
      }

      // Calculate expiration time
      const expiresAt = new Date();
      expiresAt.setHours(expiresAt.getHours() + input.expiresInHours);

      // Handle guest session creation
      let guestSession = null;
      if (input.isGuest && input.guestFingerprint) {
        // Find or create guest session
        guestSession = await ctx.prisma.guestSession.findUnique({
          where: { fingerprint: input.guestFingerprint }
        });

        if (!guestSession) {
          const guestExpiresAt = new Date();
          guestExpiresAt.setHours(guestExpiresAt.getHours() + 24); // Guest sessions last 24 hours
          
          guestSession = await ctx.prisma.guestSession.create({
            data: {
              fingerprint: input.guestFingerprint,
              buildingId: buildingId,
              expiresAt: guestExpiresAt,
            }
          });
        }
      }

      // Create the session
      const session = await ctx.prisma.shareSession.create({
        data: {
          code: code!,
          name: input.name,
          description: input.description,
          latitude: input.latitude,
          longitude: input.longitude,
          geoLockRadius: input.geoLockRadius,
          maxParticipants: input.maxParticipants,
          requiresAuth: input.requiresAuth,
          expiresAt,
          creatorId: ctx.user?.id || null,
          buildingId: buildingId,
          // Initialize ICE servers for WebRTC
          iceServers: JSON.stringify([
            { urls: 'stun:stun.l.google.com:19302' },
            { urls: 'stun:stun1.l.google.com:19302' },
          ]),
        },
        include: {
          building: true,
          creator: true,
        }
      });

      // Auto-join creator as participant
      if (ctx.user?.id || guestSession) {
        await ctx.prisma.shareSessionParticipant.create({
          data: {
            sessionId: session.id,
            userId: ctx.user?.id || null,
            guestId: guestSession?.id || null,
            peerId: crypto.randomUUID(),
            latitude: input.latitude,
            longitude: input.longitude,
            isConnected: true,
          }
        });
      }

      return session;
    }),

  // Join a session by code
  joinSession: publicProcedure
    .input(z.object({
      code: z.string().length(6),
      latitude: z.number().min(-90).max(90),
      longitude: z.number().min(-180).max(180),
      nickname: z.string().optional(),
      isGuest: z.boolean().default(false),
      guestFingerprint: z.string().optional(),
    }))
    .mutation(async ({ ctx, input }) => {
      // Find the session
      const session = await ctx.prisma.shareSession.findUnique({
        where: { code: input.code.toUpperCase() },
        include: {
          participants: true,
          building: true,
        }
      });

      if (!session) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Session not found',
        });
      }

      // Check if session expired
      if (session.expiresAt < new Date()) {
        throw new TRPCError({
          code: 'BAD_REQUEST',
          message: 'This session has expired',
        });
      }

      // Check if requires auth
      if (session.requiresAuth && !ctx.user) {
        throw new TRPCError({
          code: 'UNAUTHORIZED',
          message: 'This session requires authentication',
        });
      }

      // Check geo-lock
      if (!isWithinGeoLock(
        input.latitude,
        input.longitude,
        session.latitude,
        session.longitude,
        session.geoLockRadius
      )) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: `You must be within ${session.geoLockRadius} meters of the session location`,
        });
      }

      // Check max participants
      const activeParticipants = session.participants.filter(p => !p.leftAt);
      if (activeParticipants.length >= session.maxParticipants) {
        throw new TRPCError({
          code: 'BAD_REQUEST',
          message: 'Session is full',
        });
      }

      // Handle guest joining
      let guestSession = null;
      if (input.isGuest && input.guestFingerprint) {
        guestSession = await ctx.prisma.guestSession.findUnique({
          where: { fingerprint: input.guestFingerprint }
        });

        if (!guestSession) {
          const guestExpiresAt = new Date();
          guestExpiresAt.setHours(guestExpiresAt.getHours() + 24);
          
          guestSession = await ctx.prisma.guestSession.create({
            data: {
              fingerprint: input.guestFingerprint,
              nickname: input.nickname,
              buildingId: session.buildingId,
              expiresAt: guestExpiresAt,
            }
          });
        }
      }

      // Check if already a participant
      const existingParticipant = await ctx.prisma.shareSessionParticipant.findFirst({
        where: {
          sessionId: session.id,
          OR: [
            { userId: ctx.user?.id || undefined },
            { guestId: guestSession?.id || undefined }
          ]
        }
      });

      if (existingParticipant && !existingParticipant.leftAt) {
        throw new TRPCError({
          code: 'BAD_REQUEST',
          message: 'You are already in this session',
        });
      }

      // Create or update participant
      const participant = await ctx.prisma.shareSessionParticipant.upsert({
        where: existingParticipant ? { id: existingParticipant.id } : { id: 'new' },
        create: {
          sessionId: session.id,
          userId: ctx.user?.id || null,
          guestId: guestSession?.id || null,
          nickname: input.nickname,
          peerId: crypto.randomUUID(),
          latitude: input.latitude,
          longitude: input.longitude,
          isConnected: true,
        },
        update: {
          leftAt: null,
          isConnected: true,
          latitude: input.latitude,
          longitude: input.longitude,
          peerId: crypto.randomUUID(),
        }
      });

      return {
        session,
        participant,
        peerId: participant.peerId,
      };
    }),

  // Get session by code
  getSession: publicProcedure
    .input(z.object({
      code: z.string().length(6),
    }))
    .query(async ({ ctx, input }) => {
      const session = await ctx.prisma.shareSession.findUnique({
        where: { code: input.code.toUpperCase() },
        include: {
          building: true,
          participants: {
            where: { leftAt: null },
            include: {
              user: {
                select: {
                  id: true,
                  name: true,
                  image: true,
                }
              },
              guest: {
                select: {
                  id: true,
                  nickname: true,
                }
              }
            }
          },
          documents: {
            orderBy: { createdAt: 'desc' }
          },
          _count: {
            select: {
              participants: true,
              documents: true,
            }
          }
        }
      });

      if (!session) {
        throw new TRPCError({
          code: 'NOT_FOUND',
          message: 'Session not found',
        });
      }

      return session;
    }),

  // Leave a session
  leaveSession: publicProcedure
    .input(z.object({
      sessionId: z.string(),
      participantId: z.string(),
    }))
    .mutation(async ({ ctx, input }) => {
      const participant = await ctx.prisma.shareSessionParticipant.update({
        where: { id: input.participantId },
        data: {
          leftAt: new Date(),
          isConnected: false,
        }
      });

      return participant;
    }),

  // Get active sessions for current user
  getMySessions: protectedProcedure
    .query(async ({ ctx }) => {
      const sessions = await ctx.prisma.shareSession.findMany({
        where: {
          OR: [
            { creatorId: ctx.user.id },
            {
              participants: {
                some: {
                  userId: ctx.user.id,
                  leftAt: null,
                }
              }
            }
          ],
          expiresAt: { gt: new Date() }
        },
        include: {
          building: true,
          _count: {
            select: {
              participants: true,
              documents: true,
            }
          }
        },
        orderBy: { createdAt: 'desc' }
      });

      return sessions;
    }),

  // Update WebRTC signaling data
  updateSignaling: publicProcedure
    .input(z.object({
      sessionId: z.string(),
      participantId: z.string(),
      signalingData: z.string(), // JSON string of WebRTC offer/answer
    }))
    .mutation(async ({ ctx, input }) => {
      // Verify participant is in session
      const participant = await ctx.prisma.shareSessionParticipant.findFirst({
        where: {
          id: input.participantId,
          sessionId: input.sessionId,
        }
      });

      if (!participant) {
        throw new TRPCError({
          code: 'FORBIDDEN',
          message: 'You are not a participant in this session',
        });
      }

      // Update session signaling data (append to existing)
      await ctx.prisma.shareSession.update({
        where: { id: input.sessionId },
        data: {
          signalingData: input.signalingData,
          updatedAt: new Date(),
        }
      });

      return { success: true };
    }),

  // Ping to maintain connection
  ping: publicProcedure
    .input(z.object({
      participantId: z.string(),
    }))
    .mutation(async ({ ctx, input }) => {
      await ctx.prisma.shareSessionParticipant.update({
        where: { id: input.participantId },
        data: {
          lastPing: new Date(),
          isConnected: true,
        }
      });

      return { success: true };
    }),
});