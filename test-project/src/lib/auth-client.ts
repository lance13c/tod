import { createAuthClient } from "better-auth/react";
import { magicLinkClient, usernameClient } from "better-auth/client/plugins";

export const authClient = createAuthClient({
  baseURL: process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3001",
  plugins: [magicLinkClient(), usernameClient()],
});

export const { signIn, signUp, signOut, useSession, magicLink } = authClient;
