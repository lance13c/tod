import { PrismaClient } from "@prisma/client";
import { betterAuth } from "better-auth";
import { prismaAdapter } from "better-auth/adapters/prisma";
import { sendEmail } from "better-auth/plugins/email";
import nodemailer from "nodemailer";

const prisma = new PrismaClient();

const transporter = nodemailer.createTransport({
  host: process.env.SMTP_HOST,
  port: Number(process.env.SMTP_PORT),
  secure: process.env.SMTP_SECURE === "true",
  auth: {
    user: process.env.SMTP_USER,
    pass: process.env.SMTP_PASS,
  },
});

export const auth = betterAuth({
  database: prismaAdapter(prisma, {
    provider: "sqlite",
  }),
  emailAndPassword: {
    enabled: true,
    requireEmailVerification: true,
  },
  emailVerification: {
    sendOnSignUp: true,
    autoSignInAfterVerification: true,
  },
  plugins: [
    sendEmail({
      sendVerificationEmail: async ({ user, url }) => {
        await transporter.sendMail({
          from: `"${process.env.FROM_NAME}" <${process.env.FROM_EMAIL}>`,
          to: user.email,
          subject: "Verify your email address",
          html: `
            <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
              <h2>Welcome to ${process.env.FROM_NAME}!</h2>
              <p>Please verify your email address by clicking the link below:</p>
              <a href="${url}" style="display: inline-block; padding: 12px 24px; background-color: #3b82f6; color: white; text-decoration: none; border-radius: 6px;">Verify Email</a>
              <p style="margin-top: 20px; color: #666;">Or copy and paste this link:</p>
              <p style="color: #666; word-break: break-all;">${url}</p>
              <hr style="margin-top: 30px; border: none; border-top: 1px solid #e5e5e5;">
              <p style="color: #999; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
            </div>
          `,
        });
      },
      sendResetPasswordEmail: async ({ user, url }) => {
        await transporter.sendMail({
          from: `"${process.env.FROM_NAME}" <${process.env.FROM_EMAIL}>`,
          to: user.email,
          subject: "Reset your password",
          html: `
            <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
              <h2>Password Reset Request</h2>
              <p>We received a request to reset your password. Click the link below to create a new password:</p>
              <a href="${url}" style="display: inline-block; padding: 12px 24px; background-color: #3b82f6; color: white; text-decoration: none; border-radius: 6px;">Reset Password</a>
              <p style="margin-top: 20px; color: #666;">Or copy and paste this link:</p>
              <p style="color: #666; word-break: break-all;">${url}</p>
              <p style="margin-top: 20px; color: #666;">This link will expire in 1 hour.</p>
              <hr style="margin-top: 30px; border: none; border-top: 1px solid #e5e5e5;">
              <p style="color: #999; font-size: 12px;">If you didn't request a password reset, you can safely ignore this email.</p>
            </div>
          `,
        });
      },
    }),
  ],
  trustedOrigins: ["http://localhost:3000"],
});
