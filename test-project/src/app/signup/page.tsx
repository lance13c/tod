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
  Progress
} from "@nextui-org/react";

export default function SignUpPage() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    email: "",
    username: "",
    password: "",
    confirmPassword: "",
    name: "",
  });
  const [isVisible, setIsVisible] = useState(false);
  const [isConfirmVisible, setIsConfirmVisible] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

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
      await signUp.email({
        email: formData.email,
        password: formData.password,
        name: formData.name,
        username: formData.username,
        callbackURL: "/verify-email",
      });
      router.push("/verify-email?message=check-email");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to create account");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800 p-4">
      <Card className="max-w-md w-full" data-testid="signup-card">
        <CardHeader className="flex flex-col gap-1 items-center pb-6">
          <h1 className="text-2xl font-bold" data-testid="signup-title">Create Account</h1>
          <p className="text-small text-default-500" data-testid="signup-subtitle">
            Join us to get started with your journey
          </p>
        </CardHeader>
        <CardBody>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <Input
              label="Full Name"
              placeholder="John Doe"
              value={formData.name}
              onValueChange={(value) => handleChange("name", value)}
              variant="bordered"
              startContent={
                <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
              }
              data-testid="name-input"
            />

            <Input
              label="Username"
              placeholder="johndoe"
              value={formData.username}
              onValueChange={(value) => handleChange("username", value)}
              variant="bordered"
              isRequired
              startContent={
                <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 12a4 4 0 10-8 0 4 4 0 008 0zm0 0v1.5a2.5 2.5 0 005 0V12a9 9 0 10-9 9m4.5-1.206a8.959 8.959 0 01-4.5 1.207" />
                </svg>
              }
              data-testid="username-input"
            />

            <Input
              label="Email"
              placeholder="john@example.com"
              type="email"
              value={formData.email}
              onValueChange={(value) => handleChange("email", value)}
              variant="bordered"
              isRequired
              startContent={
                <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
              }
              data-testid="email-input"
            />

            <div className="space-y-2">
              <Input
                label="Password"
                placeholder="Minimum 8 characters"
                value={formData.password}
                onValueChange={(value) => handleChange("password", value)}
                variant="bordered"
                isRequired
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
                data-testid="password-input"
              />
              {formData.password && (
                <div className="space-y-1">
                  <div className="flex justify-between items-center text-tiny">
                    <span>Password strength:</span>
                    <span className={`text-${getPasswordStrengthColor()}`}>
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
              label="Confirm Password"
              placeholder="Re-enter password"
              value={formData.confirmPassword}
              onValueChange={(value) => handleChange("confirmPassword", value)}
              variant="bordered"
              isRequired
              startContent={
                <svg className="w-4 h-4 text-default-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              }
              endContent={
                <button
                  className="focus:outline-none"
                  type="button"
                  onClick={toggleConfirmVisibility}
                  data-testid="toggle-confirm-password-visibility"
                >
                  {isConfirmVisible ? (
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
              type={isConfirmVisible ? "text" : "password"}
              data-testid="confirm-password-input"
              color={formData.confirmPassword && formData.password !== formData.confirmPassword ? "danger" : "default"}
              errorMessage={formData.confirmPassword && formData.password !== formData.confirmPassword ? "Passwords do not match" : ""}
            />

            <Checkbox 
              isSelected={agreedToTerms} 
              onValueChange={setAgreedToTerms}
              size="sm"
              data-testid="terms-checkbox"
            >
              <span className="text-small">
                I agree to the{" "}
                <Link href="/terms" className="text-primary hover:underline">
                  Terms and Conditions
                </Link>{" "}
                and{" "}
                <Link href="/privacy" className="text-primary hover:underline">
                  Privacy Policy
                </Link>
              </span>
            </Checkbox>

            {error && (
              <Card className="bg-danger-50 dark:bg-danger-900/20">
                <CardBody>
                  <p className="text-sm text-danger" data-testid="error-message">
                    {error}
                  </p>
                </CardBody>
              </Card>
            )}

            <Button
              type="submit"
              color="primary"
              isLoading={loading}
              isDisabled={!agreedToTerms}
              fullWidth
              data-testid="signup-button"
            >
              Create Account
            </Button>

            <Divider className="my-2" />

            <div className="text-center">
              <p className="text-sm text-default-500" data-testid="login-prompt">
                Already have an account?{" "}
                <Link 
                  href="/login" 
                  className="text-primary hover:underline font-medium"
                  data-testid="login-link"
                >
                  Sign in
                </Link>
              </p>
            </div>
          </form>
        </CardBody>
      </Card>
    </div>
  );
}