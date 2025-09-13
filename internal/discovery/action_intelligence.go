package discovery

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lance13c/tod/internal/types"
)

// CodeContext provides additional context for understanding code intent
type CodeContext struct {
	FunctionBody     string   `json:"function_body"`
	Comments         []string `json:"comments"`
	ValidationRules  []string `json:"validation_rules"`
	UIElements       []string `json:"ui_elements"`
	ResponseMessages []string `json:"response_messages"`
	FormFields       []string `json:"form_fields"`
	ButtonTexts      []string `json:"button_texts"`
}

// InferUserAction uses code patterns and context to determine user intent
func InferUserAction(endpoint, method, filePath, codeContext string) types.CodeAction {
	action := types.CodeAction{
		ID:           generateActionID(endpoint, method),
		LastModified: time.Now(),
		Implementation: types.TechnicalDetails{
			Endpoint:   endpoint,
			Method:     method,
			SourceFile: filePath,
		},
	}

	// Analyze the code context for patterns
	lowerCode := strings.ToLower(codeContext)
	lowerEndpoint := strings.ToLower(endpoint)

	// Authentication patterns
	if containsAuthPatterns(endpoint, codeContext) {
		action.Category = "Authentication"
		
		if containsPattern(lowerCode, []string{"magic", "passwordless", "email-link", "magic-link"}) {
			action.Name = "Sign in with magic link"
			action.Description = "Request a passwordless sign-in link via email"
			action.Type = "form_submit"
			action.Inputs = []types.UserInput{
				{Name: "email", Label: "Email address", Type: "email", Required: true, Example: "user@example.com"},
			}
			action.Expects = types.UserExpectation{
				Success:   "Email sent with magic link",
				Failure:   "Invalid email or error message shown",
				Validates: []string{"Email is valid", "Magic link sent"},
			}
		} else if containsPattern(lowerEndpoint, []string{"register", "signup", "sign-up"}) {
			action.Name = "Create account"
			action.Description = "Register for a new account"
			action.Type = "form_submit"
			action.Inputs = inferRegistrationInputs(codeContext)
			action.Expects = types.UserExpectation{
				Success:   "Account created, possibly redirected to dashboard",
				Failure:   "Registration error shown",
				Validates: []string{"Account created", "User data stored"},
			}
		} else if containsPattern(lowerEndpoint, []string{"login", "signin", "sign-in", "auth"}) {
			action.Name = "Sign in"
			action.Description = "Sign into existing account"
			action.Type = "form_submit"
			action.Inputs = inferLoginInputs(codeContext)
			action.Expects = types.UserExpectation{
				Success:   "Redirected to dashboard or main app",
				Failure:   "Invalid credentials error shown",
				Validates: []string{"User authenticated", "Session created"},
			}
		} else if containsPattern(lowerEndpoint, []string{"logout", "signout", "sign-out"}) {
			action.Name = "Sign out"
			action.Description = "End current session and sign out"
			action.Type = "button_click"
			action.Expects = types.UserExpectation{
				Success:   "Redirected to login or home page",
				Validates: []string{"Session ended", "User signed out"},
			}
		}
	}

	// Commerce and subscription patterns
	if containsCommercePatterns(lowerCode, lowerEndpoint) {
		action.Category = "Commerce"
		
		if containsPattern(lowerEndpoint, []string{"checkout", "purchase", "payment"}) {
			action.Name = "Complete purchase"
			action.Description = "Complete payment and finalize purchase"
			action.Type = "form_submit"
			action.Inputs = inferCheckoutInputs(codeContext)
			action.Expects = types.UserExpectation{
				Success:   "Payment processed, order confirmed",
				Failure:   "Payment failed, error message shown",
				Validates: []string{"Payment processed", "Order created"},
			}
		} else if containsPattern(lowerCode, []string{"trial", "free-trial", "start-trial"}) {
			action.Name = "Start free trial"
			action.Description = "Begin trial period with full access"
			action.Type = "button_click"
			action.Expects = types.UserExpectation{
				Success:   "Trial activated, access granted",
				Validates: []string{"Trial subscription created"},
			}
		} else if containsPattern(lowerEndpoint, []string{"cart", "add-to-cart"}) {
			action.Name = "Add to cart"
			action.Description = "Add item to shopping cart"
			action.Type = "button_click"
		}
	}

	// Navigation patterns
	if method == "GET" && !strings.Contains(lowerEndpoint, "api") {
		action.Category = "Navigation"
		action.Type = "page_visit"
		action.Name = "Go to " + humanizePath(endpoint)
		action.Description = "Navigate to the " + humanizePath(endpoint) + " page"
		action.Expects = types.UserExpectation{
			Success: humanizePath(endpoint) + " page loads successfully",
		}
	}

	// Profile and settings patterns
	if containsProfilePatterns(lowerEndpoint, lowerCode) {
		action.Category = "Profile"
		
		if containsPattern(lowerEndpoint, []string{"profile", "account"}) {
			if method == "GET" {
				action.Name = "View profile"
				action.Type = "page_visit"
			} else {
				action.Name = "Update profile"
				action.Type = "form_submit"
				action.Inputs = inferProfileInputs(codeContext)
			}
		} else if containsPattern(lowerEndpoint, []string{"settings"}) {
			action.Name = "Update settings"
			action.Type = "form_submit"
		}
	}

	// Data and reporting patterns
	if containsDataPatterns(lowerEndpoint, lowerCode) {
		action.Category = "Data"
		
		if containsPattern(lowerEndpoint, []string{"export", "download"}) {
			action.Name = "Export data"
			action.Description = "Download data in specified format"
			action.Type = "button_click"
		} else if containsPattern(lowerEndpoint, []string{"report", "analytics", "dashboard"}) {
			action.Name = "View " + extractDataType(endpoint)
			action.Type = "page_visit"
		}
	}

	// If we couldn't determine a specific pattern, create a generic action
	if action.Name == "" {
		action.Name = generateGenericActionName(endpoint, method)
		action.Category = "General"
		action.Description = fmt.Sprintf("Perform %s operation", strings.ToUpper(method))
	}

	return action
}

// Pattern matching helpers

func containsAuthPatterns(endpoint, code string) bool {
	authKeywords := []string{
		"auth", "login", "signin", "signup", "register",
		"logout", "signout", "session", "password", "email",
		"magic", "token", "jwt", "oauth", "user", "account",
	}
	return containsPattern(strings.ToLower(endpoint+" "+code), authKeywords)
}

func containsCommercePatterns(code, endpoint string) bool {
	commerceKeywords := []string{
		"checkout", "payment", "purchase", "cart", "order",
		"subscription", "trial", "billing", "stripe", "paypal",
		"price", "product", "buy", "sell",
	}
	return containsPattern(code+" "+endpoint, commerceKeywords)
}

func containsProfilePatterns(endpoint, code string) bool {
	profileKeywords := []string{
		"profile", "account", "settings", "preferences", "user",
		"update", "edit", "personal", "info", "details",
	}
	return containsPattern(endpoint+" "+code, profileKeywords)
}

func containsDataPatterns(endpoint, code string) bool {
	dataKeywords := []string{
		"export", "download", "report", "analytics", "dashboard",
		"data", "csv", "json", "pdf", "chart", "graph", "stats",
	}
	return containsPattern(endpoint+" "+code, dataKeywords)
}

func containsPattern(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// Input inference helpers

func inferRegistrationInputs(code string) []types.UserInput {
	inputs := []types.UserInput{
		{Name: "email", Label: "Email address", Type: "email", Required: true, Example: "user@example.com"},
	}
	
	if strings.Contains(strings.ToLower(code), "password") {
		inputs = append(inputs, types.UserInput{
			Name: "password", Label: "Password", Type: "password", Required: true, Example: "********",
		})
	}
	
	if containsPattern(strings.ToLower(code), []string{"name", "firstname", "lastname"}) {
		inputs = append(inputs, types.UserInput{
			Name: "name", Label: "Full name", Type: "text", Required: true, Example: "John Smith",
		})
	}
	
	if containsPattern(strings.ToLower(code), []string{"company", "organization"}) {
		inputs = append(inputs, types.UserInput{
			Name: "company", Label: "Company name", Type: "text", Required: false, Example: "Acme Corp",
		})
	}
	
	return inputs
}

func inferLoginInputs(code string) []types.UserInput {
	inputs := []types.UserInput{
		{Name: "email", Label: "Email address", Type: "email", Required: true, Example: "user@example.com"},
	}
	
	if strings.Contains(strings.ToLower(code), "password") {
		inputs = append(inputs, types.UserInput{
			Name: "password", Label: "Password", Type: "password", Required: true, Example: "********",
		})
	}
	
	return inputs
}

func inferCheckoutInputs(code string) []types.UserInput {
	inputs := []types.UserInput{}
	
	if containsPattern(strings.ToLower(code), []string{"card", "credit", "payment"}) {
		inputs = append(inputs, 
			types.UserInput{Name: "cardNumber", Label: "Card number", Type: "text", Required: true, Example: "4242424242424242"},
			types.UserInput{Name: "expiryDate", Label: "Expiry date", Type: "text", Required: true, Example: "12/25"},
			types.UserInput{Name: "cvc", Label: "CVC", Type: "text", Required: true, Example: "123"},
		)
	}
	
	if containsPattern(strings.ToLower(code), []string{"billing", "address"}) {
		inputs = append(inputs,
			types.UserInput{Name: "address", Label: "Billing address", Type: "text", Required: true, Example: "123 Main St"},
			types.UserInput{Name: "city", Label: "City", Type: "text", Required: true, Example: "New York"},
			types.UserInput{Name: "zipCode", Label: "ZIP code", Type: "text", Required: true, Example: "10001"},
		)
	}
	
	return inputs
}

func inferProfileInputs(code string) []types.UserInput {
	inputs := []types.UserInput{}
	
	if containsPattern(strings.ToLower(code), []string{"name", "firstname", "lastname"}) {
		inputs = append(inputs, types.UserInput{
			Name: "name", Label: "Full name", Type: "text", Required: true, Example: "John Smith",
		})
	}
	
	if containsPattern(strings.ToLower(code), []string{"email"}) {
		inputs = append(inputs, types.UserInput{
			Name: "email", Label: "Email address", Type: "email", Required: true, Example: "user@example.com",
		})
	}
	
	if containsPattern(strings.ToLower(code), []string{"phone", "mobile"}) {
		inputs = append(inputs, types.UserInput{
			Name: "phone", Label: "Phone number", Type: "tel", Required: false, Example: "+1 (555) 123-4567",
		})
	}
	
	return inputs
}

// Utility helpers

func generateActionID(endpoint, method string) string {
	// Convert /api/auth/magic-link POST to sign_in_magic_link
	cleaned := strings.ReplaceAll(endpoint, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.Trim(cleaned, "_")
	cleaned = regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(cleaned, "_")
	cleaned = strings.ToLower(cleaned)
	
	methodLower := strings.ToLower(method)
	
	// Clean up common patterns
	if strings.HasPrefix(cleaned, "_api_") {
		cleaned = strings.TrimPrefix(cleaned, "_api_")
	}
	if strings.HasPrefix(cleaned, "api_") {
		cleaned = strings.TrimPrefix(cleaned, "api_")
	}
	
	return fmt.Sprintf("%s_%s", methodLower, cleaned)
}

func humanizePath(path string) string {
	// Convert /user-profile to "user profile"
	cleaned := strings.Trim(path, "/")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "/", " ")
	
	// Remove common prefixes
	cleaned = strings.TrimPrefix(cleaned, "api ")
	cleaned = strings.TrimPrefix(cleaned, "v1 ")
	cleaned = strings.TrimPrefix(cleaned, "v2 ")
	
	return strings.ToLower(strings.TrimSpace(cleaned))
}

func extractDataType(endpoint string) string {
	// Extract meaningful part from endpoint like /api/reports/analytics -> "analytics report"
	parts := strings.Split(endpoint, "/")
	if len(parts) > 0 {
		return humanizePath(parts[len(parts)-1])
	}
	return "data"
}

func generateGenericActionName(endpoint, method string) string {
	humanPath := humanizePath(endpoint)
	
	switch strings.ToUpper(method) {
	case "GET":
		return "View " + humanPath
	case "POST":
		return "Create " + humanPath
	case "PUT", "PATCH":
		return "Update " + humanPath
	case "DELETE":
		return "Delete " + humanPath
	default:
		return strings.ToUpper(method) + " " + humanPath
	}
}