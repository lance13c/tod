'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { Button } from '~/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '~/components/ui/card'
import { User, Settings, Bell, LogOut, MapPin, FileText, Users } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '~/components/ui/avatar'
import { useSession, authClient } from '~/lib/auth'
import { useRouter } from 'next/navigation'
import { toast } from 'sonner'

export default function Dashboard() {
  const { data: session, isPending } = useSession()
  const router = useRouter()
  
  useEffect(() => {
    // If not loading and no session, redirect to login
    if (!isPending && !session) {
      router.push('/auth/login')
    }
  }, [session, isPending, router])

  const handleSignOut = async () => {
    await authClient.signOut({
      fetchOptions: {
        onSuccess: () => {
          toast.success('Signed out successfully')
          router.push('/')
        },
      },
    })
  }

  if (isPending) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading...</p>
        </div>
      </div>
    )
  }

  if (!session?.user) {
    return null
  }

  const user = session.user
  const joinedDate = new Date(user.createdAt || Date.now()).toLocaleDateString('en-US', { 
    month: 'long', 
    year: 'numeric' 
  })

  return (
    <>
      <div className="flex justify-between items-center mb-6">
        <h1 className='text-2xl font-bold text-gray-800'>
          Welcome back, {user.name || user.email?.split('@')[0]}!
        </h1>
        <Button 
          onClick={handleSignOut}
          variant="outline" 
          className="border-red-200 text-red-700 hover:bg-red-50"
        >
          <LogOut className="mr-2 h-4 w-4" />
          Sign Out
        </Button>
      </div>

      <div className='grid grid-cols-1 md:grid-cols-3 gap-6'>
        <Card className='border-indigo-200'>
          <CardHeader>
            <CardTitle className='text-indigo-700 flex items-center'>
              <User className='mr-2 h-5 w-5' />
              Profile
            </CardTitle>
            <CardDescription>Manage your personal information</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='flex flex-col items-center mb-4'>
              <Avatar className='h-20 w-20 mb-2'>
                <AvatarImage
                  src='/placeholder.svg?height=80&width=80'
                  alt={user.name}
                />
                <AvatarFallback className='bg-indigo-100 text-indigo-700 text-xl'>
                  {(user.name || user.email || 'U')
                    .split(' ')
                    .map((n) => n[0])
                    .join('')
                    .toUpperCase()
                    .slice(0, 2)}
                </AvatarFallback>
              </Avatar>
            </div>
            <div className='space-y-2'>
              <p className='text-sm text-gray-600'>
                <span className='font-medium'>Name:</span> {user.name || 'Not set'}
              </p>
              <p className='text-sm text-gray-600'>
                <span className='font-medium'>Email:</span> {user.email}
              </p>
              {user.phone && (
                <p className='text-sm text-gray-600'>
                  <span className='font-medium'>Phone:</span> {user.phone}
                </p>
              )}
              <p className='text-sm text-gray-600'>
                <span className='font-medium'>Member since:</span>{' '}
                {joinedDate}
              </p>
              {user.isAdmin && (
                <p className='text-sm text-indigo-600 font-medium'>
                  Administrator
                </p>
              )}
              <Link href='/dashboard/profile' className='w-full block'>
                <Button
                  variant='outline'
                  className='mt-4 w-full border-indigo-200 text-indigo-700 hover:bg-indigo-50'
                >
                  View Profile
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>

        <Card className='border-indigo-200'>
          <CardHeader>
            <CardTitle className='text-indigo-700 flex items-center'>
              <Settings className='mr-2 h-5 w-5' />
              Account Settings
            </CardTitle>
            <CardDescription>Manage your account preferences</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='space-y-4'>
              <Link href='/dashboard/change-password' className='w-full block'>
                <Button
                  variant='outline'
                  className='w-full justify-start border-indigo-200 text-gray-700 hover:bg-indigo-50'
                >
                  Change Password
                </Button>
              </Link>
              <Button
                variant='outline'
                className='w-full justify-start border-indigo-200 text-gray-700 hover:bg-indigo-50'
              >
                Two-Factor Authentication
              </Button>
              <Button
                variant='outline'
                className='w-full justify-start border-indigo-200 text-gray-700 hover:bg-indigo-50'
              >
                Privacy Settings
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card className='border-indigo-200'>
          <CardHeader>
            <CardTitle className='text-indigo-700 flex items-center'>
              <Bell className='mr-2 h-5 w-5' />
              Notifications
            </CardTitle>
            <CardDescription>Recent activity and alerts</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='space-y-4'>
              <div className='p-3 bg-indigo-50 rounded-md border border-indigo-100'>
                <p className='text-sm font-medium text-indigo-700'>
                  Welcome to Better Auth!
                </p>
                <p className='text-xs text-gray-600 mt-1'>
                  Thank you for joining our platform.
                </p>
              </div>
              <div className='p-3 bg-gray-50 rounded-md border border-gray-100'>
                <p className='text-sm font-medium text-gray-700'>
                  Profile created successfully
                </p>
                <p className='text-xs text-gray-600 mt-1'>
                  Your profile has been set up.
                </p>
              </div>
              <Button
                variant='outline'
                className='w-full border-indigo-200 text-indigo-700 hover:bg-indigo-50'
              >
                View All Notifications
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className='mt-6'>
        <Card className='border-green-200'>
          <CardHeader>
            <CardTitle className='text-green-700 flex items-center'>
              <Users className='mr-2 h-5 w-5' />
              GroupUp Sessions
            </CardTitle>
            <CardDescription>Share files with people nearby</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='grid grid-cols-1 sm:grid-cols-2 gap-4'>
              <Link href='/sessions/create'>
                <Button className='w-full bg-green-600 hover:bg-green-700 text-white'>
                  <MapPin className='mr-2 h-4 w-4' />
                  Create Session
                </Button>
              </Link>
              <Link href='/sessions/join'>
                <Button 
                  variant='outline' 
                  className='w-full border-green-200 text-green-700 hover:bg-green-50'
                >
                  <Users className='mr-2 h-4 w-4' />
                  Join Session
                </Button>
              </Link>
            </div>
            <div className='mt-4 p-4 bg-green-50 rounded-lg border border-green-100'>
              <h4 className='text-sm font-medium text-green-800 mb-2'>What is GroupUp?</h4>
              <p className='text-xs text-gray-600'>
                GroupUp lets you instantly share files with people in the same location. 
                Create a session, share the code, and start collaborating without complex setup.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
