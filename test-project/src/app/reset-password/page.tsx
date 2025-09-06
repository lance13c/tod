"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

export default function ResetPasswordPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [status, setStatus] = useState<
    "idle" | "loading" | "success" | "error"
  >("idle");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (password !== confirmPassword) {
      setStatus("error");
      setMessage("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      setStatus("error");
      setMessage("Password must be at least 8 characters");
      return;
    }

    const token = searchParams.get("token");
    if (!token) {
      setStatus("error");
      setMessage("Invalid reset link");
      return;
    }

    setStatus("loading");
    setMessage("");

    try {
      const response = await fetch("/api/auth/reset-password", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ token, password }),
      });

      if (response.ok) {
        setStatus("success");
        setMessage("Password reset successfully! Redirecting to login...");
        setTimeout(() => {
          router.push("/login");
        }, 3000);
      } else {
        setStatus("error");
        setMessage("Failed to reset password. The link may be expired.");
      }
    } catch (_error) {
      setStatus("error");
      setMessage("An error occurred. Please try again.");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
            Create new password
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Enter your new password below
          </p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700"
              >
                New Password
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Minimum 8 characters"
                disabled={status === "loading"}
              />
            </div>

            <div>
              <label
                htmlFor="confirmPassword"
                className="block text-sm font-medium text-gray-700"
              >
                Confirm Password
              </label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Re-enter password"
                disabled={status === "loading"}
              />
            </div>
          </div>

          {status === "success" && (
            <div className="rounded-md bg-green-50 p-4">
              <div className="text-sm text-green-800">{message}</div>
            </div>
          )}

          {status === "error" && (
            <div className="rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{message}</div>
            </div>
          )}

          <div>
            <button
              type="submit"
              disabled={status === "loading" || status === "success"}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {status === "loading" ? "Resetting..." : "Reset password"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
