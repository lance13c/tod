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
  Chip,
  Spinner,
  Modal,
  ModalContent,
  ModalBody,
  Accordion,
  AccordionItem
} from "@nextui-org/react";
import { Mail, Lock, Eye, EyeOff, Sparkles, ArrowRight, Shield, Send, ChevronDown } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isVisible, setIsVisible] = useState(false);
  const [magicLinkEmail, setMagicLinkEmail] = useState("");
  const [magicLinkSent, setMagicLinkSent] = useState(false);
  const [magicLinkLoading, setMagicLinkLoading] = useState(false);
  const [showMagicLinkForm, setShowMagicLinkForm] = useState(false);
  const [showPasswordForm, setShowPasswordForm] = useState(false);

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
        setShowMagicLinkForm(false);
      }
    } catch (err: any) {
      setError(err.message || "Failed to send magic link. Please try again.");
    } finally {
      setMagicLinkLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
      {/* Background decorations */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-purple-200 dark:bg-purple-900/20 rounded-full blur-3xl opacity-30"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-blue-200 dark:bg-blue-900/20 rounded-full blur-3xl opacity-30"></div>
      </div>

      {/* Magic Link Loading Modal */}
      <Modal 
        isOpen={magicLinkLoading} 
        hideCloseButton
        placement="center"
        backdrop="blur"
        classNames={{
          body: "py-6",
          backdrop: "bg-[#292f46]/50 backdrop-opacity-40",
          base: "border-[#292f46] bg-gradient-to-br from-white to-gray-50 dark:from-[#19172c] dark:to-[#21232c] text-[#a8b0d3] shadow-2xl",
        }}
      >
        <ModalContent>
          <ModalBody>
            <div className="flex flex-col items-center justify-center gap-4 py-4">
              <div className="relative">
                <Spinner 
                  size="lg" 
                  color="primary"
                  classNames={{
                    circle1: "border-b-blue-500",
                    circle2: "border-b-purple-600",
                  }}
                />
                <Send className="w-6 h-6 text-blue-500 absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 animate-pulse" />
              </div>
              <div className="text-center space-y-2">
                <p className="text-lg font-semibold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                  Sending Magic Link
                </p>
                <p className="text-sm text-default-500">
                  We're sending a sign-in link to {magicLinkEmail}
                </p>
                <p className="text-xs text-default-400">
                  This may take a few seconds...
                </p>
              </div>
            </div>
          </ModalBody>
        </ModalContent>
      </Modal>

      <Card className="max-w-md w-full backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-2xl" data-testid="login-card">
        <CardHeader className="flex flex-col gap-2 items-center pb-2 pt-8 px-8">
          <div className="p-3 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl mb-2">
            <Shield className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent" data-testid="login-title">
            Welcome Back
          </h1>
          <p className="text-default-500 text-center" data-testid="login-subtitle">
            Sign in to access your account
          </p>
        </CardHeader>
        
        <CardBody className="px-8 py-6">
          {/* Magic Link Section - PRIMARY */}
          <div className="space-y-4">
            {magicLinkSent ? (
              <Card className="bg-success-50 dark:bg-success-900/20">
                <CardBody className="text-center py-3">
                  <p className="text-success font-medium" data-testid="magic-link-sent-message">
                    ✨ Magic link sent!
                  </p>
                  <p className="text-sm text-default-600 mt-1">
                    Check your email for the sign-in link
                  </p>
                </CardBody>
              </Card>
            ) : (
              <form onSubmit={handleMagicLink} className="flex flex-col gap-4">
                <Input
                  placeholder="Enter your email"
                  type="email"
                  value={magicLinkEmail}
                  onValueChange={setMagicLinkEmail}
                  variant="bordered"
                  size="lg"
                  isDisabled={magicLinkLoading}
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Mail className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  isRequired
                  data-testid="magic-link-email-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />
                
                {error && showMagicLinkForm && (
                  <Chip color="danger" variant="flat" className="w-full">
                    <span className="text-sm" data-testid="error-message">{error}</span>
                  </Chip>
                )}
                
                <Button
                  type="submit"
                  className="bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold shadow-lg hover:shadow-xl transition-all"
                  isDisabled={magicLinkLoading || !magicLinkEmail}
                  fullWidth
                  size="lg"
                  data-testid="send-magic-link-button"
                  startContent={<Sparkles className="w-4 h-4" />}
                  endContent={<ArrowRight className="w-4 h-4" />}
                >
                  Sign In with Magic Link
                </Button>
              </form>
            )}
          </div>

          <div className="relative my-6">
            <Divider />
            <span className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 px-4 text-small text-default-400">
              OR
            </span>
          </div>

          {/* Email/Password Section - SECONDARY (Collapsible) */}
          <Accordion 
            variant="light"
            className="px-0"
            itemClasses={{
              base: "py-0",
              title: "text-small text-default-500",
              trigger: "px-2 py-2 data-[hover=true]:bg-default-100 rounded-lg transition-all",
              indicator: "text-default-400",
              content: "pt-0 pb-2"
            }}
          >
            <AccordionItem
              key="password-login"
              aria-label="Sign in with password"
              title="Use email and password instead"
              indicator={<ChevronDown className="w-4 h-4" />}
            >
              <form onSubmit={handleEmailPasswordLogin} className="flex flex-col gap-4">
                <Input
                  placeholder="Enter your email"
                  type="email"
                  value={email}
                  onValueChange={setEmail}
                  variant="bordered"
                  size="lg"
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Mail className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  isRequired
                  data-testid="email-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-default-300",
                    innerWrapper: "gap-3",
                  }}
                />
                
                <Input
                  placeholder="Enter your password"
                  value={password}
                  onValueChange={setPassword}
                  variant="bordered"
                  size="lg"
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Lock className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  endContent={
                    <button
                      className="focus:outline-none"
                      type="button"
                      onClick={toggleVisibility}
                      data-testid="toggle-password-visibility"
                    >
                      {isVisible ? (
                        <EyeOff className="w-4 h-4 text-default-400" />
                      ) : (
                        <Eye className="w-4 h-4 text-default-400" />
                      )}
                    </button>
                  }
                  type={isVisible ? "text" : "password"}
                  isRequired
                  data-testid="password-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-default-300",
                    innerWrapper: "gap-3",
                  }}
                />
                
                <div className="flex justify-between items-center mb-2">
                  <Checkbox 
                    isSelected={rememberMe} 
                    onValueChange={setRememberMe}
                    size="sm"
                    data-testid="remember-me-checkbox"
                    classNames={{
                      wrapper: "after:bg-gradient-to-r after:from-gray-400 after:to-gray-500",
                    }}
                  >
                    <span className="text-small text-default-500">{" "}Remember me</span>
                  </Checkbox>
                  <Link
                    href="/forgot-password"
                    className="text-sm text-default-500 hover:text-primary hover:underline"
                    data-testid="forgot-password-link"
                  >
                    Forgot password?
                  </Link>
                </div>
                
                {error && !magicLinkEmail && (
                  <Chip color="danger" variant="flat" className="w-full">
                    <span className="text-sm" data-testid="error-message">{error}</span>
                  </Chip>
                )}
                
                <Button
                  type="submit"
                  className="bg-gradient-to-r from-gray-500 to-gray-600 text-white font-medium shadow hover:shadow-lg transition-all"
                  isLoading={isLoading}
                  fullWidth
                  size="lg"
                  data-testid="email-login-button"
                  startContent={!isLoading && <Lock className="w-4 h-4" />}
                  endContent={!isLoading && <ArrowRight className="w-4 h-4" />}
                >
                  Sign In with Password
                </Button>
              </form>
            </AccordionItem>
          </Accordion>

          <Divider className="my-6" />

          <div className="text-center">
            <p className="text-default-500 text-sm">
              Don't have an account?{" "}
              <Link 
                href="/signup" 
                className="font-semibold text-primary hover:underline"
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