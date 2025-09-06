"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { signIn } from "@/lib/auth-client";
import { 
  Card, 
  CardBody, 
  CardHeader, 
  Input, 
  Button, 
  Divider,
  Checkbox,
  Tabs,
  Tab
} from "@nextui-org/react";

export default function LoginPage() {
  const router = useRouter();
  const [loginMethod, setLoginMethod] = useState<"email" | "username">("email");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isVisible, setIsVisible] = useState(false);
  const [magicLinkEmail, setMagicLinkEmail] = useState("");
  const [magicLinkSent, setMagicLinkSent] = useState(false);
  const [magicLinkLoading, setMagicLinkLoading] = useState(false);

  const toggleVisibility = () => setIsVisible(!isVisible);

  const handleEmailPasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      const result = await signIn.email({
        email,
        password,
        rememberMe,
      });

      if (result.data?.user) {
        router.push("/dashboard");
      }
    } catch (err: any) {
      setError(err.message || "Failed to sign in. Please check your credentials.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleUsernamePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      const result = await signIn.username({
        username,
        password,
        rememberMe,
      });

      if (result.data?.user) {
        router.push("/dashboard");
      }
    } catch (err: any) {
      setError(err.message || "Failed to sign in. Please check your credentials.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleMagicLink = async (e: React.FormEvent) => {
    e.preventDefault();
    setMagicLinkLoading(true);
    setError(null);

    try {
      const result = await signIn.magicLink({
        email: magicLinkEmail,
      });

      if (result.data) {
        setMagicLinkSent(true);
      }
    } catch (err: any) {
      setError(err.message || "Failed to send magic link. Please try again.");
    } finally {
      setMagicLinkLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800 p-4">
      <Card className="max-w-md w-full" data-testid="login-card">
        <CardHeader className="flex flex-col gap-1 items-center pb-6">
          <h1 className="text-2xl font-bold" data-testid="login-title">Welcome Back</h1>
          <p className="text-small text-default-500" data-testid="login-subtitle">
            Sign in to access your account
          </p>
        </CardHeader>
        <CardBody className="gap-4">
          <Tabs 
            selectedKey={loginMethod} 
            onSelectionChange={(key) => setLoginMethod(key as "email" | "username")}
            fullWidth
            data-testid="login-method-tabs"
          >
            <Tab key="email" title="Email" data-testid="email-tab">
              <form onSubmit={handleEmailPasswordLogin} className="flex flex-col gap-4 mt-4">
                <Input
                  label="Email"
                  placeholder="Enter your email"
                  type="email"
                  value={email}
                  onValueChange={setEmail}
                  variant="bordered"
                  startContent={
                    <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  }
                  isRequired
                  data-testid="email-input"
                />
                <Input
                  label="Password"
                  placeholder="Enter your password"
                  value={password}
                  onValueChange={setPassword}
                  variant="bordered"
                  startContent={
                    <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                  }
                  endContent={
                    <button
                      className="focus:outline-none"
                      type="button"
                      onClick={toggleVisibility}
                      data-testid="toggle-password-visibility"
                    >
                      {isVisible ? (
                        <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                        </svg>
                      )}
                    </button>
                  }
                  type={isVisible ? "text" : "password"}
                  isRequired
                  data-testid="password-input"
                />
                <div className="flex justify-between items-center">
                  <Checkbox 
                    isSelected={rememberMe} 
                    onValueChange={setRememberMe}
                    size="sm"
                    data-testid="remember-me-checkbox"
                  >
                    Remember me
                  </Checkbox>
                  <Link
                    href="/forgot-password"
                    className="text-sm text-primary hover:underline"
                    data-testid="forgot-password-link"
                  >
                    Forgot password?
                  </Link>
                </div>
                {error && (
                  <div className="text-danger text-sm" data-testid="error-message">
                    {error}
                  </div>
                )}
                <Button
                  type="submit"
                  color="primary"
                  isLoading={isLoading}
                  fullWidth
                  data-testid="email-login-button"
                >
                  Sign In with Email
                </Button>
              </form>
            </Tab>
            <Tab key="username" title="Username" data-testid="username-tab">
              <form onSubmit={handleUsernamePasswordLogin} className="flex flex-col gap-4 mt-4">
                <Input
                  label="Username"
                  placeholder="Enter your username"
                  value={username}
                  onValueChange={setUsername}
                  variant="bordered"
                  startContent={
                    <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  }
                  isRequired
                  data-testid="username-input"
                />
                <Input
                  label="Password"
                  placeholder="Enter your password"
                  value={password}
                  onValueChange={setPassword}
                  variant="bordered"
                  startContent={
                    <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                  }
                  endContent={
                    <button
                      className="focus:outline-none"
                      type="button"
                      onClick={toggleVisibility}
                      data-testid="toggle-password-visibility-username"
                    >
                      {isVisible ? (
                        <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                        </svg>
                      )}
                    </button>
                  }
                  type={isVisible ? "text" : "password"}
                  isRequired
                  data-testid="password-input-username"
                />
                <div className="flex justify-between items-center">
                  <Checkbox 
                    isSelected={rememberMe} 
                    onValueChange={setRememberMe}
                    size="sm"
                    data-testid="remember-me-checkbox-username"
                  >
                    Remember me
                  </Checkbox>
                  <Link
                    href="/forgot-password"
                    className="text-sm text-primary hover:underline"
                    data-testid="forgot-password-link-username"
                  >
                    Forgot password?
                  </Link>
                </div>
                {error && (
                  <div className="text-danger text-sm" data-testid="error-message-username">
                    {error}
                  </div>
                )}
                <Button
                  type="submit"
                  color="primary"
                  isLoading={isLoading}
                  fullWidth
                  data-testid="username-login-button"
                >
                  Sign In with Username
                </Button>
              </form>
            </Tab>
          </Tabs>

          <div className="relative">
            <Divider className="my-4" />
            <span className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-background px-2 text-tiny text-default-500">
              OR
            </span>
          </div>

          {/* Magic Link Section */}
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-center" data-testid="magic-link-title">
              Sign in with Magic Link
            </h3>
            {!magicLinkSent ? (
              <form onSubmit={handleMagicLink} className="flex flex-col gap-4">
                <Input
                  label="Email for Magic Link"
                  placeholder="Enter your email"
                  type="email"
                  value={magicLinkEmail}
                  onValueChange={setMagicLinkEmail}
                  variant="bordered"
                  startContent={
                    <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  }
                  isRequired
                  data-testid="magic-link-email-input"
                />
                <Button
                  type="submit"
                  color="secondary"
                  variant="flat"
                  isLoading={magicLinkLoading}
                  fullWidth
                  data-testid="magic-link-button"
                >
                  Send Magic Link
                </Button>
              </form>
            ) : (
              <Card className="bg-success-50 dark:bg-success-900/20">
                <CardBody>
                  <p className="text-sm text-center" data-testid="magic-link-success">
                    Magic link sent! Check your email at{" "}
                    <span className="font-medium">{magicLinkEmail}</span> to sign in.
                  </p>
                </CardBody>
              </Card>
            )}
          </div>

          <Divider className="my-4" />

          <div className="text-center">
            <p className="text-sm text-default-500" data-testid="signup-prompt">
              Don't have an account?{" "}
              <Link 
                href="/signup" 
                className="text-primary hover:underline font-medium"
                data-testid="signup-link"
              >
                Sign up
              </Link>
            </p>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}