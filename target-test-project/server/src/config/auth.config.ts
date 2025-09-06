import { betterAuth } from 'better-auth'
import { prismaAdapter } from 'better-auth/adapters/prisma'
import { magicLink } from 'better-auth/plugins'
import prisma from './db.config'
import { BETTER_AUTH_URL, APP_URL, sendEmail } from '~/libs'

export const auth = betterAuth({
  database: prismaAdapter(prisma, {
    provider: 'sqlite'
  }),
  baseURL: 'http://localhost:8000/api/auth', // API server URL for auth endpoints
  appURL: APP_URL, // Client app URL for redirects
  trustedOrigins: [
    APP_URL,
    'http://localhost:3001', // Allow Next.js on port 3001
    'http://localhost:3000'
  ],
  user: {
    modelName: 'users',
    additionalFields: {
      phone: { type: 'string', nullable: true, returned: true },
      isAdmin: { type: 'boolean', default: false, returned: true },
    },
    changeEmail: {
      enabled: true,
      sendChangeEmailVerification: async ({ user, newEmail, url, token }) => {
        // Send change email verification
        sendEmail({
          to: newEmail,
          subject: 'Verify your new email',
          text: `Click the link to verify your new email: ${url}`,
        })
      },
    },
  },
  session: { modelName: 'sessions' },
  account: {
    modelName: 'accounts',
    accountLinking: {
      enabled: true,
      trustedProviders: ['github', 'google', 'email-password'],
      allowDifferentEmails: false,
      sendAccountLinkingEmail: async ({ user, url, token }) => {
        // Send account linking email
        sendEmail({
          to: user.email,
          subject: 'Link your account',
          text: `Click the link to confirm linking your account: ${url}`,
        })
      },
    },
  },
  emailAndPassword: {
    enabled: true,
    disableSignUp: false, // Enable/Disable sign up
    minPasswordLength: 8,
    maxPasswordLength: 128,
    autoSignIn: true,
    // Password hashing configuration
    password: {
      hash: async (password) => {
        return await Bun.password.hash(password, {
          algorithm: 'bcrypt',
          cost: 10,
        })
      },
      verify: async ({ password, hash }) => {
        return await Bun.password.verify(password, hash)
      },
    },
    // Email verification configuration
    requireEmailVerification: false,
    emailVerification: {
      sendVerificationEmail: async ({ user, url, token }) => {
        await sendEmail({
          to: user.email,
          subject: 'Verify your email',
          text: `Click the link to verify your email: ${url}`,
        })
      },
      sendOnSignUp: true,
      autoSignInAfterVerification: true,
      expiresIn: 3600, // 1 hour
    },
    // Password reset configuration
    sendResetPassword: async ({ user, url, token }, request) => {
      await sendEmail({
        to: user.email,
        subject: 'Reset your password',
        text: `Click the link to reset your password: ${url}`,
      })
    },
  },
  plugins: [
    magicLink({
      sendMagicLink: async ({ email, url, token }) => {
        console.log('🔮 Sending magic link to:', email)
        console.log('📎 Magic link URL:', url)
        
        // Ensure the URL points to the server API, not the client
        const verifyUrl = url.replace('http://localhost:3001/api/auth', 'http://localhost:8000/api/auth')
        console.log('✅ Corrected verify URL:', verifyUrl)
        
        const result = await sendEmail({
          to: email,
          subject: 'Sign in to GroupUp',
          text: `Click the link to sign in to your account:\n\n${verifyUrl}\n\nThis link will expire in 5 minutes.`,
          html: `
            <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
              <h2>Sign in to GroupUp</h2>
              <p>Click the button below to sign in to your account:</p>
              <a href="${verifyUrl}" style="display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0;">
                Sign In
              </a>
              <p style="color: #666; font-size: 14px;">Or copy and paste this link in your browser:</p>
              <p style="color: #666; font-size: 14px; word-break: break-all;">${verifyUrl}</p>
              <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
              <p style="color: #999; font-size: 12px;">This link will expire in 5 minutes. If you didn't request this email, you can safely ignore it.</p>
            </div>
          `
        })
        console.log('📧 Magic link email result:', result)
        if (!result?.success) {
          console.error('Failed to send magic link email:', result?.error)
          throw new Error('Failed to send magic link email')
        }
      },
    }),
  ],
  socialProviders: {
    github: {
      clientId: process.env.GITHUB_CLIENT_ID as string,
      clientSecret: process.env.GITHUB_CLIENT_SECRET as string,
    },
    google: {
      clientId: process.env.GOOGLE_CLIENT_ID as string,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET as string,
    },
  },
})