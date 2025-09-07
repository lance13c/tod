"use client";

import Link from "next/link";
import { useSession } from "@/lib/auth-client";
import { Button, Card, CardBody } from "@nextui-org/react";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const { data: session, isPending } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (!isPending && session) {
      router.push("/dashboard");
    }
  }, [session, isPending, router]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <main className="max-w-7xl mx-auto px-4 py-12">
        {/* Hero Section */}
        <div className="text-center mb-16">
          <h1 className="text-5xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
            GroupUp
          </h1>
          <p className="text-xl md:text-2xl text-gray-600 dark:text-gray-300 mb-8 max-w-3xl mx-auto">
            Instantly share files with people nearby. No signup required.
          </p>
          <div className="flex flex-col items-center gap-3">
            <Button 
              as={Link}
              href="/share"
              size="lg" 
              color="secondary" 
              className="font-bold text-lg px-8 py-6 shadow-xl hover:shadow-2xl transition-all transform hover:scale-105 bg-gradient-to-r from-purple-500 to-pink-500 text-white"
              variant="shadow"
            >
              Start Sharing Now →
            </Button>
            {session ? (
              <Button 
                as={Link}
                href="/dashboard"
                size="md" 
                variant="flat" 
                className="font-medium text-sm hover:bg-default-200 transition-all"
              >
                Go to Dashboard
              </Button>
            ) : (
              <div className="flex items-center gap-2 mt-2">
                <span className="text-sm text-gray-500">Already have an account?</span>
                <Button 
                  as={Link}
                  href="/login"
                  size="sm" 
                  variant="light" 
                  className="font-medium underline hover:no-underline"
                >
                  Sign In
                </Button>
              </div>
            )}
          </div>
        </div>

        {/* Features Grid */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3 mb-16">
          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Instant Setup</h2>
              <p className="text-gray-600 dark:text-gray-400">
                Start sharing in seconds. No account needed for quick file transfers.
              </p>
            </CardBody>
          </Card>
          
          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Secure P2P</h2>
              <p className="text-gray-600 dark:text-gray-400">
                Files transfer directly between devices. Your data never touches our servers.
              </p>
            </CardBody>
          </Card>
          
          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Location Based</h2>
              <p className="text-gray-600 dark:text-gray-400">
                Automatically connect with people in your building or nearby area.
              </p>
            </CardBody>
          </Card>

          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-orange-600 dark:text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Guest Friendly</h2>
              <p className="text-gray-600 dark:text-gray-400">
                No signup required. Join sessions instantly and upgrade later if needed.
              </p>
            </CardBody>
          </Card>

          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-pink-100 dark:bg-pink-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-pink-600 dark:text-pink-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Rich Media</h2>
              <p className="text-gray-600 dark:text-gray-400">
                Share photos, documents, and files with beautiful gallery views.
              </p>
            </CardBody>
          </Card>

          <Card className="bg-white/80 dark:bg-gray-800/80 backdrop-blur hover:scale-105 transition-transform cursor-default">
            <CardBody className="text-center p-8">
              <div className="w-16 h-16 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-indigo-600 dark:text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold mb-3">Mobile First</h2>
              <p className="text-gray-600 dark:text-gray-400">
                Designed for mobile devices with touch-friendly interfaces.
              </p>
            </CardBody>
          </Card>
        </div>

        {/* CTA Section */}
        <div className="text-center bg-gradient-to-r from-blue-600 to-purple-600 rounded-2xl p-12 text-white">
          <h2 className="text-3xl font-bold mb-4">Ready to Share?</h2>
          <p className="text-xl mb-8 opacity-90">
            Join thousands who are sharing files instantly and securely.
          </p>
          <Button 
            as={Link}
            href="/share"
            size="lg" 
            className="bg-white text-blue-600 font-semibold hover:bg-gray-100 shadow-lg hover:shadow-xl transition-all"
          >
            Get Started Free →
          </Button>
        </div>

        {/* Existing Auth Section for logged in users */}
        {session && (
          <div className="mt-12 text-center">
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              Welcome back, {session.user?.email}!
            </p>
            <Button 
              as={Link}
              href="/organizations"
              variant="bordered"
              className="border-2 hover:bg-default-100 transition-all"
            >
              Browse Organizations →
            </Button>
          </div>
        )}
      </main>
    </div>
  );
}