import { createTRPCRouter } from '@/server/trpc';
import { organizationRouter } from './routers/organization';
import { shareSessionRouter } from './routers/share-session';

/**
 * This is the primary router for your server.
 * All routers added in /api/routers should be manually added here.
 */
export const appRouter = createTRPCRouter({
  organization: organizationRouter,
  shareSession: shareSessionRouter,
});

// export type definition of API
export type AppRouter = typeof appRouter;