"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { signOut, useSession } from "@/lib/auth-client";

export default function DashboardPage() {
  const router = useRouter();
  const { data: session, isPending } = useSession();

  useEffect(() => {
    if (!isPending && !session) {
      router.push("/login");
    }
  }, [session, isPending, router]);

  const handleSignOut = async () => {
    await signOut();
    router.push("/login");
  };

  if (isPending) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  if (!session) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:px-8">
        <div className="bg-white shadow rounded-lg p-6">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
            <p className="mt-2 text-gray-600">Welcome to your dashboard!</p>
          </div>

          <div className="space-y-4">
            <div className="bg-gray-50 p-4 rounded-md">
              <h2 className="text-lg font-semibold text-gray-900 mb-2">
                User Information
              </h2>
              <dl className="space-y-2">
                <div className="flex">
                  <dt className="font-medium text-gray-500 w-32">Email:</dt>
                  <dd className="text-gray-900">{session.user?.email}</dd>
                </div>
                {session.user?.name && (
                  <div className="flex">
                    <dt className="font-medium text-gray-500 w-32">Name:</dt>
                    <dd className="text-gray-900">{session.user.name}</dd>
                  </div>
                )}
                {session.user?.username && (
                  <div className="flex">
                    <dt className="font-medium text-gray-500 w-32">
                      Username:
                    </dt>
                    <dd className="text-gray-900">{session.user.username}</dd>
                  </div>
                )}
                <div className="flex">
                  <dt className="font-medium text-gray-500 w-32">User ID:</dt>
                  <dd className="text-gray-900">{session.user?.id}</dd>
                </div>
                <div className="flex">
                  <dt className="font-medium text-gray-500 w-32">
                    Session ID:
                  </dt>
                  <dd className="text-gray-900">{session.session?.id}</dd>
                </div>
              </dl>
            </div>

            <div className="pt-4">
              <button
                type="button"
                onClick={handleSignOut}
                className="px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
