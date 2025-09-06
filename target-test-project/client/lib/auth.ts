import { createAuthClient } from 'better-auth/react' // make sure to import from better-auth/react
import {
  inferAdditionalFields,
  magicLinkClient,
} from 'better-auth/client/plugins'

export const authClient = createAuthClient({
  baseURL: typeof window !== 'undefined' 
    ? `${window.location.origin}/api/auth`
    : 'http://localhost:3001/api/auth', // Use full URL for the Next.js proxy
  plugins: [
    inferAdditionalFields({
      user: {
        phone: { type: 'string' },
        isAdmin: { type: 'boolean' },
      },
    }),
    magicLinkClient(),
  ],
})

export const { signIn, signUp, useSession } = authClient
