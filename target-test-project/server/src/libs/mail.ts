import nodemailer from 'nodemailer'
import type { Transporter } from 'nodemailer'

// Environment variables
const SMTP_HOST = process.env.SMTP_HOST
const SMTP_PORT = process.env.SMTP_PORT ? parseInt(process.env.SMTP_PORT) : 587
const SMTP_SECURE = process.env.SMTP_SECURE === 'true'
const SMTP_USER = process.env.SMTP_USER
const SMTP_PASS = process.env.SMTP_PASS
const FROM_EMAIL = process.env.FROM_EMAIL || 'noreply@example.com'
const FROM_NAME = process.env.FROM_NAME || 'App'

// For development: console logging
const DEV_MODE = process.env.SMTP_HOST === 'console' || !SMTP_HOST

// Create reusable transporter
let transporter: Transporter | null = null

if (!DEV_MODE && SMTP_HOST && SMTP_USER && SMTP_PASS) {
  console.log('🔧 Configuring SMTP with:', {
    host: SMTP_HOST,
    port: SMTP_PORT,
    secure: SMTP_SECURE,
    user: SMTP_USER
  })
  
  transporter = nodemailer.createTransport({
    host: SMTP_HOST,
    port: SMTP_PORT,
    secure: SMTP_SECURE, // true for 465, false for 587
    auth: {
      user: SMTP_USER,
      pass: SMTP_PASS
    },
    requireTLS: !SMTP_SECURE, // Require STARTTLS for port 587
    tls: {
      rejectUnauthorized: false // Allow self-signed certificates for dev
    },
    debug: true, // Enable debug output
    logger: true // Log to console
  })

  // Verify connection configuration on startup (async, don't block)
  setTimeout(() => {
    transporter?.verify((error) => {
      if (error) {
        console.error('⚠️ SMTP verification failed (non-blocking):', error.message)
        console.log('Email sending will be attempted when needed.')
      } else {
        console.log('✅ SMTP server verified and ready')
      }
    })
  }, 1000)
}

export const sendEmail = async ({ 
  to, 
  subject, 
  text, 
  html 
}: {
  to: string
  subject: string
  text: string
  html?: string
}) => {
  // Development mode: Console logging
  if (DEV_MODE) {
    console.log('\n📧 EMAIL (Development Mode - Console)')
    console.log('=====================================')
    console.log(`From: ${FROM_NAME} <${FROM_EMAIL}>`)
    console.log(`To: ${to}`)
    console.log(`Subject: ${subject}`)
    console.log('-------------------------------------')
    console.log('Text Content:')
    console.log(text)
    if (html && html !== text) {
      console.log('-------------------------------------')
      console.log('HTML Content:')
      console.log(html)
    }
    console.log('=====================================\n')
    return { success: true, mode: 'console' }
  }

  // Production mode: Send via SMTP
  if (!transporter) {
    console.error('Email transporter not configured')
    return { 
      success: false, 
      error: 'Email service not configured. Please check SMTP settings.' 
    }
  }

  try {
    const info = await transporter.sendMail({
      from: `"${FROM_NAME}" <${FROM_EMAIL}>`,
      to,
      subject,
      text,
      html: html || text,
    })

    console.log('✅ Email sent successfully')
    console.log('Message ID:', info.messageId)
    
    return { 
      success: true, 
      messageId: info.messageId,
      accepted: info.accepted,
      rejected: info.rejected
    }
  } catch (error) {
    console.error('❌ Error sending email:', error)
    return { 
      success: false, 
      error: error instanceof Error ? error.message : 'Unknown error occurred' 
    }
  }
}

// Test function for verifying email configuration
export const testEmailConnection = async () => {
  if (DEV_MODE) {
    console.log('📧 Email is in development mode (console logging)')
    return { success: true, mode: 'development' }
  }

  if (!transporter) {
    return { 
      success: false, 
      error: 'Email transporter not configured' 
    }
  }

  try {
    await transporter.verify()
    console.log('✅ Email connection verified successfully')
    return { success: true, mode: 'smtp' }
  } catch (error) {
    console.error('❌ Email connection failed:', error)
    return { 
      success: false, 
      error: error instanceof Error ? error.message : 'Unknown error' 
    }
  }
}