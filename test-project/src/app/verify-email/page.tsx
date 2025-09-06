"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

export default function VerifyEmailPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<
    "verifying" | "success" | "error" | "check-email"
  >("verifying");
  const [message, setMessage] = useState("");

  useEffect(() => {
    const messageParam = searchParams.get("message");
    if (messageParam === "check-email") {
      setStatus("check-email");
      setMessage("Please check your email to verify your account.");
      return;
    }

    const token = searchParams.get("token");
    if (!token) {
      setStatus("error");
      setMessage("Invalid verification link");
      return;
    }

    // The Better Auth verification happens automatically via the API route
    // This page is just for user feedback
    const verifyEmail = async () => {
      try {
        const response = await fetch(`/api/auth/verify-email?token=${token}`);
        if (response.ok) {
          setStatus("success");
          setMessage("Email verified successfully! Redirecting to login...");
          setTimeout(() => {
            router.push("/login");
          }, 3000);
        } else {
          setStatus("error");
          setMessage(
            "Verification failed. The link may be expired or invalid.",
          );
        }
      } catch (_error) {
        setStatus("error");
        setMessage("An error occurred during verification.");
      }
    };

    verifyEmail();
  }, [searchParams, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 text-center">
        <div>
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            Email Verification
          </h2>
        </div>

        {status === "verifying" && (
          <div>
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Verifying your email...</p>
          </div>
        )}

        {status === "success" && (
          <div className="rounded-md bg-green-50 p-4">
            <div className="text-sm text-green-800">
              <svg
                className="h-12 w-12 text-green-600 mx-auto mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-label="Success checkmark"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              {message}
            </div>
          </div>
        )}

        {status === "check-email" && (
          <div className="rounded-md bg-blue-50 p-4">
            <div className="text-sm text-blue-800">
              <svg
                className="h-12 w-12 text-blue-600 mx-auto mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-label="Email icon"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                />
              </svg>
              <h3 className="font-semibold mb-2">Check your email</h3>
              <p>{message}</p>
              <p className="mt-2">
                We've sent a verification link to your email address.
              </p>
            </div>
          </div>
        )}

        {status === "error" && (
          <div className="rounded-md bg-red-50 p-4">
            <div className="text-sm text-red-800">
              <svg
                className="h-12 w-12 text-red-600 mx-auto mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-label="Error icon"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              {message}
            </div>
            <a
              href="/signup"
              className="mt-4 inline-block text-blue-600 hover:text-blue-500"
            >
              Back to Sign Up
            </a>
          </div>
        )}
      </div>
    </div>
  );
}
