import { PrismaClient } from '@prisma/client'

const prismaClientSingleton = () => {
  return new PrismaClient({
    log: process.env.NODE_ENV === 'development' ? ['query', 'error', 'warn'] : ['error'],
  })
}

declare global {
  var prisma: undefined | ReturnType<typeof prismaClientSingleton>
}

const prisma = globalThis.prisma ?? prismaClientSingleton()

if (process.env.NODE_ENV !== 'production') globalThis.prisma = prisma

export const DB = async () => {
  try {
    await prisma.$connect()
    console.log('✅ Database Connected via Prisma')
    return prisma
  } catch (err: any) {
    console.error(`❌ Database Connection Error: ${err.message}`)
    process.exit(1)
  }
}

export default prisma