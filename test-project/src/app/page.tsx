"use client";

import Link from "next/link";
import { useSession } from "@/lib/auth-client";

export default function Home() {
  const { data: session } = useSession();

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 text-center">
        <div>
          <h1 className="text-4xl font-extrabold text-gray-900">
            Welcome to Better Auth
          </h1>
          <p className="mt-3 text-lg text-gray-600">
            A modern authentication solution for Next.js
          </p>
        </div>

        <div className="mt-8 space-y-4">
          {session ? (
            <>
              <p className="text-gray-700">
                Welcome back, {session.user?.email}!
              </p>
              <Link
                href="/dashboard"
                className="inline-block w-full px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Go to Dashboard
              </Link>
            </>
          ) : (
            <>
              <Link
                href="/login"
                className="inline-block w-full px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Sign In
              </Link>
              <Link
                href="/signup"
                className="inline-block w-full px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Create Account
              </Link>
            </>
          )}
        </div>

        <div className="mt-12 text-sm text-gray-500">
          <p>Powered by Better Auth</p>
          <p className="mt-1">Email & Password Authentication</p>
        </div>
      </div>
    </div>
  );
}
