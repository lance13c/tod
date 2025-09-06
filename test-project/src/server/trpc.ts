/**
 * tRPC server configuration
 * This is where we create the tRPC context and procedures
 */

import { initTRPC, TRPCError } from '@trpc/server';
import { type FetchCreateContextFnOptions } from '@trpc/server/adapters/fetch';
import superjson from 'superjson';
import { ZodError } from 'zod';
import { auth } from '@/lib/auth';
import { prisma } from '@/server/db';

/**
 * 1. CONTEXT
 * This section defines the "context" that will be available
 * in all of your tRPC procedures
 */
export const createTRPCContext = async (opts: FetchCreateContextFnOptions) => {
  // Get the session from Better Auth
  const session = await auth.api.getSession({
    headers: opts.req.headers,
  });

  return {
    session,
    user: session?.user || null,
    prisma,
    headers: opts.req.headers,
  };
};

/**
 * 2. INITIALIZATION
 * This is where the tRPC API is initialized, connecting the context and transformer
 */
const t = initTRPC.context<typeof createTRPCContext>().create({
  transformer: superjson,
  errorFormatter({ shape, error }) {
    return {
      ...shape,
      data: {
        ...shape.data,
        zodError:
          error.cause instanceof ZodError ? error.cause.flatten() : null,
      },
    };
  },
});

/**
 * 3. ROUTER & PROCEDURE
 * These are the pieces you use to build your tRPC API
 */
export const createTRPCRouter = t.router;
export const createCallerFactory = t.createCallerFactory;

/**
 * Public (unauthenticated) procedure
 * Use this for procedures that don't require authentication
 */
export const publicProcedure = t.procedure;

/**
 * Protected (authenticated) procedure
 * Use this for procedures that require a logged-in user
 */
export const protectedProcedure = t.procedure.use(({ ctx, next }) => {
  if (!ctx.session || !ctx.user) {
    throw new TRPCError({ code: 'UNAUTHORIZED' });
  }
  return next({
    ctx: {
      // infers the `session` and `user` as non-nullable
      session: ctx.session,
      user: ctx.user,
    },
  });
});

/**
 * Admin procedure
 * Use this for procedures that require admin privileges
 */
export const adminProcedure = protectedProcedure.use(({ ctx, next }) => {
  // You would check for admin role here
  // For now, we'll just use the protected procedure
  // if (ctx.user.role !== 'ADMIN') {
  //   throw new TRPCError({ code: 'FORBIDDEN' });
  // }
  return next();
});