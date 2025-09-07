"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { signUp } from "@/lib/auth-client";
import { 
  Card, 
  CardBody, 
  CardHeader, 
  Input, 
  Button,
  Checkbox,
  Divider,
  Progress,
  Chip,
  Accordion,
  AccordionItem,
  Modal,
  ModalContent,
  ModalBody,
  Spinner
} from "@nextui-org/react";
import { Mail, User, Lock, Eye, EyeOff, ArrowRight, UserPlus, Shield, Sparkles, ChevronDown, Send } from "lucide-react";

export default function SignUpPage() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    email: "",
    password: "",
    confirmPassword: "",
    name: "",
  });
  const [isVisible, setIsVisible] = useState(false);
  const [isConfirmVisible, setIsConfirmVisible] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [magicLinkEmail, setMagicLinkEmail] = useState("");
  const [magicLinkName, setMagicLinkName] = useState("");
  const [magicLinkSent, setMagicLinkSent] = useState(false);
  const [magicLinkLoading, setMagicLinkLoading] = useState(false);
  const [magicLinkAgreed, setMagicLinkAgreed] = useState(false);

  const toggleVisibility = () => setIsVisible(!isVisible);
  const toggleConfirmVisibility = () => setIsConfirmVisible(!isConfirmVisible);

  const handleChange = (field: string, value: string) => {
    setFormData({
      ...formData,
      [field]: value,
    });
  };

  // Password strength calculator
  const calculatePasswordStrength = (password: string): number => {
    let strength = 0;
    if (password.length >= 8) strength += 25;
    if (password.length >= 12) strength += 25;
    if (/[a-z]/.test(password) && /[A-Z]/.test(password)) strength += 25;
    if (/\d/.test(password)) strength += 12.5;
    if (/[^a-zA-Z\d]/.test(password)) strength += 12.5;
    return strength;
  };

  const passwordStrength = calculatePasswordStrength(formData.password);
  const getPasswordStrengthColor = () => {
    if (passwordStrength < 30) return "danger";
    if (passwordStrength < 60) return "warning";
    if (passwordStrength < 80) return "primary";
    return "success";
  };

  const getPasswordStrengthText = () => {
    if (passwordStrength < 30) return "Weak";
    if (passwordStrength < 60) return "Fair";
    if (passwordStrength < 80) return "Good";
    return "Strong";
  };

  const handleMagicLinkSignup = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!magicLinkAgreed) {
      setError("Please agree to the terms and conditions");
      return;
    }

    setMagicLinkLoading(true);

    try {
      const result = await signUp.magicLink({
        email: magicLinkEmail,
        name: magicLinkName,
      });
      
      if (result.data) {
        setMagicLinkSent(true);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to send magic link");
      console.error(err);
    } finally {
      setMagicLinkLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!agreedToTerms) {
      setError("Please agree to the terms and conditions");
      return;
    }

    if (formData.password !== formData.confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (formData.password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    setLoading(true);

    try {
      const result = await signUp.email({
        email: formData.email,
        password: formData.password,
        name: formData.name,
        callbackURL: "/dashboard",
      });
      
      // If user is already logged in after signup, go to dashboard
      if (result.data?.user) {
        router.push("/dashboard");
      } else {
        // Otherwise show the email verification message
        router.push("/verify-email?message=check-email");
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create account");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900 flex items-center justify-center p-4">
      {/* Background decorations */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-purple-200 dark:bg-purple-900/20 rounded-full blur-3xl opacity-30"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-blue-200 dark:bg-blue-900/20 rounded-full blur-3xl opacity-30"></div>
      </div>

      <Card className="max-w-md w-full backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-2xl" data-testid="signup-card">
        <CardHeader className="flex flex-col gap-2 items-center pb-2 pt-8 px-8">
          <div className="p-3 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl mb-2">
            <UserPlus className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent" data-testid="signup-title">
            Create Account
          </h1>
          <p className="text-default-500 text-center" data-testid="signup-subtitle">
            Join us to get started with your journey
          </p>
        </CardHeader>
        
        <CardBody className="pb-8 px-8">
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
                      Creating Your Account
                    </p>
                    <p className="text-sm text-default-500">
                      We're sending a magic link to {magicLinkEmail}
                    </p>
                    <p className="text-xs text-default-400">
                      This may take a few seconds...
                    </p>
                  </div>
                </div>
              </ModalBody>
            </ModalContent>
          </Modal>

          {/* Magic Link Section - PRIMARY */}
          <div className="space-y-4">
            {magicLinkSent ? (
              <Card className="bg-success-50 dark:bg-success-900/20">
                <CardBody className="text-center py-3">
                  <p className="text-success font-medium" data-testid="magic-link-sent-message">
                    ✨ Magic link sent!
                  </p>
                  <p className="text-sm text-default-600 mt-1">
                    Check your email to complete sign up
                  </p>
                </CardBody>
              </Card>
            ) : (
              <form onSubmit={handleMagicLinkSignup} className="flex flex-col gap-4">
                <Input
                  placeholder="Enter your full name"
                  value={magicLinkName}
                  onValueChange={setMagicLinkName}
                  variant="bordered"
                  size="lg"
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <User className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  data-testid="magic-link-name-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />

                <Input
                  placeholder="Enter your email"
                  type="email"
                  value={magicLinkEmail}
                  onValueChange={setMagicLinkEmail}
                  variant="bordered"
                  size="lg"
                  isRequired
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Mail className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  data-testid="magic-link-email-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />

                <div className="flex items-start gap-2 mb-2">
                  <Checkbox 
                    isSelected={magicLinkAgreed} 
                    onValueChange={setMagicLinkAgreed}
                    size="sm"
                    data-testid="magic-link-terms-checkbox"
                    classNames={{
                      wrapper: "after:bg-gradient-to-r after:from-blue-500 after:to-purple-500 mt-1",
                    }}
                  />
                  <span className="text-small leading-snug pt-0.5">
                    I agree to the{" "}
                    <Link href="/terms" className="text-primary hover:underline">
                      Terms and Conditions
                    </Link>{" "}
                    and{" "}
                    <Link href="/privacy" className="text-primary hover:underline">
                      Privacy Policy
                    </Link>
                  </span>
                </div>
                
                {error && magicLinkEmail && (
                  <Chip color="danger" variant="flat" className="w-full">
                    <span className="text-sm" data-testid="error-message">{error}</span>
                  </Chip>
                )}
                
                <Button
                  type="submit"
                  className="bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold shadow-lg hover:shadow-xl transition-all"
                  isDisabled={magicLinkLoading || !magicLinkEmail || !magicLinkAgreed}
                  fullWidth
                  size="lg"
                  data-testid="send-magic-link-button"
                  startContent={<Sparkles className="w-4 h-4" />}
                  endContent={<ArrowRight className="w-4 h-4" />}
                >
                  Sign Up with Magic Link
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
              key="password-signup"
              aria-label="Sign up with password"
              title="Use email and password instead"
              indicator={<ChevronDown className="w-4 h-4" />}
            >
              <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                <Input
                  placeholder="Enter your full name"
                  value={formData.name}
                  onValueChange={(value) => handleChange("name", value)}
                  variant="bordered"
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <User className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  data-testid="name-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />

                <Input
                  placeholder="Enter your email"
                  type="email"
                  value={formData.email}
                  onValueChange={(value) => handleChange("email", value)}
                  variant="bordered"
                  isRequired
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Mail className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  data-testid="email-input"
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />

                <div className="space-y-2">
                  <Input
                    placeholder="Create a password (min 8 characters)"
                    value={formData.password}
                    onValueChange={(value) => handleChange("password", value)}
                    variant="bordered"
                    isRequired
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
                    data-testid="password-input"
                    classNames={{
                      label: "text-default-600",
                      inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                      innerWrapper: "gap-3",
                    }}
                  />
                  {formData.password && (
                    <div className="space-y-1">
                      <div className="flex justify-between items-center text-tiny">
                        <span className="text-default-500">Password strength:</span>
                        <span className={`font-medium`}>
                          {getPasswordStrengthText()}
                        </span>
                      </div>
                      <Progress 
                        value={passwordStrength} 
                        color={getPasswordStrengthColor()}
                        size="sm"
                        data-testid="password-strength-indicator"
                      />
                    </div>
                  )}
                </div>

                <Input
                  placeholder="Confirm your password"
                  value={formData.confirmPassword}
                  onValueChange={(value) => handleChange("confirmPassword", value)}
                  variant="bordered"
                  isRequired
                  startContent={
                    <div className="pointer-events-none flex items-center">
                      <Shield className="w-4 h-4 text-default-400" />
                    </div>
                  }
                  endContent={
                    <button
                      className="focus:outline-none"
                      type="button"
                      onClick={toggleConfirmVisibility}
                      data-testid="toggle-confirm-password-visibility"
                    >
                      {isConfirmVisible ? (
                        <EyeOff className="w-4 h-4 text-default-400" />
                      ) : (
                        <Eye className="w-4 h-4 text-default-400" />
                      )}
                    </button>
                  }
                  type={isConfirmVisible ? "text" : "password"}
                  data-testid="confirm-password-input"
                  color={formData.confirmPassword && formData.password !== formData.confirmPassword ? "danger" : "default"}
                  errorMessage={formData.confirmPassword && formData.password !== formData.confirmPassword ? "Passwords do not match" : ""}
                  classNames={{
                    label: "text-default-600",
                    inputWrapper: "border-default-200 data-[hover=true]:border-primary",
                    innerWrapper: "gap-3",
                  }}
                />

                <div className="flex items-start gap-2 mb-2">
                  <Checkbox 
                    isSelected={agreedToTerms} 
                    onValueChange={setAgreedToTerms}
                    size="sm"
                    data-testid="terms-checkbox"
                    classNames={{
                      wrapper: "after:bg-gradient-to-r after:from-blue-500 after:to-purple-500 mt-1",
                    }}
                  />
                  <span className="text-small leading-snug pt-0.5">
                    I agree to the{" "}
                    <Link href="/terms" className="text-primary hover:underline">
                      Terms and Conditions
                    </Link>{" "}
                    and{" "}
                    <Link href="/privacy" className="text-primary hover:underline">
                      Privacy Policy
                    </Link>
                  </span>
                </div>

                {error && !magicLinkEmail && (
                  <Chip color="danger" variant="flat" className="w-full">
                    <span className="text-sm" data-testid="error-message">{error}</span>
                  </Chip>
                )}

                <Button
                  type="submit"
                  className="bg-gradient-to-r from-gray-500 to-gray-600 text-white font-medium shadow hover:shadow-lg transition-all"
                  isLoading={loading}
                  isDisabled={!agreedToTerms}
                  fullWidth
                  size="lg"
                  data-testid="signup-button"
                  endContent={!loading && <ArrowRight className="w-4 h-4" />}
                >
                  Create Account with Password
                </Button>
              </form>
            </AccordionItem>
          </Accordion>

          <Divider className="my-4" />

          <div className="text-center">
            <p className="text-default-500 text-sm" data-testid="login-prompt">
              Already have an account?{" "}
              <Link 
                href="/login" 
                className="font-semibold text-primary hover:underline"
                data-testid="login-link"
              >
                Sign in
              </Link>
            </p>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}