package discovery

import (
	"regexp"
	"strings"
)

// ContextAnalyzer extracts meaningful context from source code
type ContextAnalyzer struct {
	// Cached compiled regex patterns for performance
	commentPattern     *regexp.Regexp
	functionPattern    *regexp.Regexp
	validationPattern  *regexp.Regexp
	responsePattern    *regexp.Regexp
	formFieldPattern   *regexp.Regexp
	buttonTextPattern  *regexp.Regexp
	routePattern       *regexp.Regexp
	
	// Additional cached patterns for frequently used operations
	htmlFieldPattern   *regexp.Regexp
	jsxFieldPattern    *regexp.Regexp
	buttonPattern      *regexp.Regexp
	stringPattern      *regexp.Regexp
	
	// Pattern caches for dynamic patterns
	validationPatterns []string
	responsePatterns   []string
	textPatterns       []string
	userMessagePatterns []string
	schemaPatterns     []string
	compiledValidationPatterns []regexp.Regexp
	compiledResponsePatterns   []regexp.Regexp
	compiledTextPatterns       []regexp.Regexp
	compiledUserMsgPatterns    []regexp.Regexp
	compiledSchemaPatterns     []regexp.Regexp
}

// NewContextAnalyzer creates a new context analyzer with pre-compiled patterns
func NewContextAnalyzer() *ContextAnalyzer {
	ca := &ContextAnalyzer{
		// Core patterns
		commentPattern:     regexp.MustCompile(`(?m)^\s*//\s*(.+)$|/\*\s*(.+?)\s*\*/`),
		functionPattern:    regexp.MustCompile(`function\s+(\w+)|const\s+(\w+)\s*=|(\w+)\s*:\s*\(`),
		validationPattern:  regexp.MustCompile(`\.required\(\)|\.min\(|\.max\(|\.email\(\)|\.matches\(|validate\w*|joi\.|yup\.|z\.|schema`),
		responsePattern:    regexp.MustCompile(`res\.status\(\d+\)|res\.json\(|return\s+\{|\.status\(\d+\)`),
		formFieldPattern:   regexp.MustCompile(`name=["'](\w+)["']|input\s+.*name|<input|<select|<textarea`),
		buttonTextPattern:  regexp.MustCompile(`<button[^>]*>([^<]+)</button>|button.*text.*["']([^"']+)["']`),
		routePattern:       regexp.MustCompile(`app\.(get|post|put|delete|patch)\s*\(\s*["']([^"']+)["']|router\.(get|post|put|delete|patch)`),
		
		// Additional frequently used patterns
		htmlFieldPattern: regexp.MustCompile(`(?i)<(?:input|select|textarea)[^>]*name=["']([^"']+)["'][^>]*>`),
		jsxFieldPattern:  regexp.MustCompile(`(?i)name=["']([^"']+)["']|<(?:input|Input)[^>]*`),
		buttonPattern:    regexp.MustCompile(`(?i)<button[^>]*>([^<]+)</button>`),
		stringPattern:    regexp.MustCompile(`["']([^"']{10,}?)["']`), // Strings with 10+ chars
	}
	
	// Pre-compile dynamic patterns
	ca.initDynamicPatterns()
	return ca
}

// initDynamicPatterns pre-compiles all the dynamic patterns used in analysis
func (ca *ContextAnalyzer) initDynamicPatterns() {
	// Validation patterns
	ca.validationPatterns = []string{
		`\.required\(\)`,
		`\.email\(\)`,
		`\.min\(\d+\)`,
		`\.max\(\d+\)`,
		`joi\.`,
		`yup\.`,
		`z\.`,
	}
	ca.compiledValidationPatterns = make([]regexp.Regexp, len(ca.validationPatterns))
	for i, pattern := range ca.validationPatterns {
		ca.compiledValidationPatterns[i] = *regexp.MustCompile(pattern)
	}
	
	// Response patterns
	ca.responsePatterns = []string{
		`res\.status\(\d+\)\.json\([^)]+\)`,
		`res\.json\([^)]+\)`,
		`return\s+\{[^}]*message[^}]*\}`,
		`throw\s+new\s+Error\([^)]+\)`,
		`status:\s*\d+`,
		`message:\s*["'][^"']*["']`,
	}
	ca.compiledResponsePatterns = make([]regexp.Regexp, len(ca.responsePatterns))
	for i, pattern := range ca.responsePatterns {
		ca.compiledResponsePatterns[i] = *regexp.MustCompile(pattern)
	}
	
	// Text patterns
	ca.textPatterns = []string{
		`<button[^>]*>([^<]+)</button>`,
		`placeholder\s*[:=]\s*["']([^"']+)["']`,
		`title\s*[:=]\s*["']([^"']+)["']`,
		`submitText\s*[:=]\s*["']([^"']+)["']`,
		`label\s*[:=]\s*["']([^"']+)["']`,
	}
	ca.compiledTextPatterns = make([]regexp.Regexp, len(ca.textPatterns))
	for i, pattern := range ca.textPatterns {
		ca.compiledTextPatterns[i] = *regexp.MustCompile(pattern)
	}
	
	// User message patterns
	ca.userMessagePatterns = []string{
		`["'][^"']*error[^"']*["']`,
		`["'][^"']*success[^"']*["']`,
		`["'][^"']*invalid[^"']*["']`,
		`["'][^"']*required[^"']*["']`,
		`["'][^"']*must be[^"']*["']`,
		`["'][^"']*please[^"']*["']`,
	}
	ca.compiledUserMsgPatterns = make([]regexp.Regexp, len(ca.userMessagePatterns))
	for i, pattern := range ca.userMessagePatterns {
		ca.compiledUserMsgPatterns[i] = *regexp.MustCompile(`(?i)` + pattern)
	}
	
	// Schema patterns
	ca.schemaPatterns = []string{
		`(\w+):\s*joi\.\w+`,
		`(\w+):\s*yup\.\w+`,
		`(\w+):\s*z\.\w+`,
		`(\w+):\s*schema\.\w+`,
	}
	ca.compiledSchemaPatterns = make([]regexp.Regexp, len(ca.schemaPatterns))
	for i, pattern := range ca.schemaPatterns {
		ca.compiledSchemaPatterns[i] = *regexp.MustCompile(pattern)
	}
}

// ExtractContext analyzes source code and extracts meaningful context
func (ca *ContextAnalyzer) ExtractContext(code string, filePath string) CodeContext {
	context := CodeContext{
		Comments:         ca.extractComments(code),
		ValidationRules:  ca.extractValidationRules(code),
		ResponseMessages: ca.extractResponseMessages(code),
		FormFields:       ca.extractFormFields(code),
		ButtonTexts:      ca.extractButtonTexts(code),
	}

	// Extract function body if this looks like a single function/handler
	context.FunctionBody = ca.extractMainFunctionBody(code)

	// Add UI elements from various sources
	context.UIElements = append(context.UIElements, context.FormFields...)
	context.UIElements = append(context.UIElements, context.ButtonTexts...)

	return context
}

// extractComments finds and cleans comments from the code
func (ca *ContextAnalyzer) extractComments(code string) []string {
	var comments []string
	matches := ca.commentPattern.FindAllStringSubmatch(code, -1)
	
	for _, match := range matches {
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				comment := strings.TrimSpace(match[i])
				if comment != "" && !isBoilerplateComment(comment) {
					comments = append(comments, comment)
				}
			}
		}
	}
	
	return ca.deduplicateStrings(comments)
}

// extractMainFunctionBody attempts to find the main function/handler body
func (ca *ContextAnalyzer) extractMainFunctionBody(code string) string {
	// Look for the main export or handler function
	lines := strings.Split(code, "\n")
	
	// Find likely function start
	var functionStart int = -1
	var braceCount int
	var inFunction bool
	var functionBody strings.Builder
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Look for function definitions, exports, or handlers
		if !inFunction && ca.looksLikeFunctionStart(trimmed) {
			functionStart = i
			inFunction = true
			braceCount = 0
		}
		
		if inFunction {
			functionBody.WriteString(line + "\n")
			
			// Count braces to find function end
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			
			if braceCount <= 0 && functionStart < i {
				break
			}
		}
	}
	
	return strings.TrimSpace(functionBody.String())
}

// looksLikeFunctionStart determines if a line looks like the start of a function
func (ca *ContextAnalyzer) looksLikeFunctionStart(line string) bool {
	functionIndicators := []string{
		"export default",
		"export const",
		"export function",
		"function",
		"const handler",
		"const api",
		"async function",
		"async (",
		"=> {",
	}
	
	lowerLine := strings.ToLower(line)
	for _, indicator := range functionIndicators {
		if strings.Contains(lowerLine, strings.ToLower(indicator)) {
			return true
		}
	}
	
	return false
}

// extractValidationRules finds validation patterns in the code
func (ca *ContextAnalyzer) extractValidationRules(code string) []string {
	var rules []string
	
	// Use pre-compiled patterns for performance
	for _, re := range ca.compiledValidationPatterns {
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			rules = append(rules, strings.TrimSpace(match))
		}
	}
	
	// Also look for validation error messages
	errorMessages := ca.findValidationMessages(code)
	rules = append(rules, errorMessages...)
	
	return ca.deduplicateStrings(rules)
}

// extractResponseMessages finds response messages and status codes
func (ca *ContextAnalyzer) extractResponseMessages(code string) []string {
	var messages []string
	
	// Use pre-compiled patterns for performance
	for _, re := range ca.compiledResponsePatterns {
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			messages = append(messages, strings.TrimSpace(match))
		}
	}
	
	// Extract string literals that look like user messages
	userMessages := ca.findUserMessages(code)
	messages = append(messages, userMessages...)
	
	return ca.deduplicateStrings(messages)
}

// extractFormFields finds form field names and types
func (ca *ContextAnalyzer) extractFormFields(code string) []string {
	var fields []string
	
	// HTML form fields
	htmlFieldPattern := regexp.MustCompile(`(?i)<(?:input|select|textarea)[^>]*name=["']([^"']+)["'][^>]*>`)
	matches := htmlFieldPattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}
	
	// React/JSX form fields
	jsxFieldPattern := regexp.MustCompile(`(?i)name=["']([^"']+)["']|<(?:input|Input)[^>]*`)
	jsxMatches := jsxFieldPattern.FindAllStringSubmatch(code, -1)
	for _, match := range jsxMatches {
		if len(match) > 1 && match[1] != "" {
			fields = append(fields, match[1])
		}
	}
	
	// Form validation schema fields
	schemaFields := ca.extractSchemaFields(code)
	fields = append(fields, schemaFields...)
	
	return ca.deduplicateStrings(fields)
}

// extractButtonTexts finds button text content
func (ca *ContextAnalyzer) extractButtonTexts(code string) []string {
	var texts []string
	
	// HTML buttons
	buttonPattern := regexp.MustCompile(`(?i)<button[^>]*>([^<]+)</button>`)
	matches := buttonPattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			text := strings.TrimSpace(match[1])
			if text != "" && !isGenericText(text) {
				texts = append(texts, text)
			}
		}
	}
	
	// Button text in props or variables
	textPatterns := []string{
		`buttonText\s*[:=]\s*["']([^"']+)["']`,
		`submitText\s*[:=]\s*["']([^"']+)["']`,
		`label\s*[:=]\s*["']([^"']+)["']`,
	}
	
	for _, pattern := range textPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				texts = append(texts, match[1])
			}
		}
	}
	
	return ca.deduplicateStrings(texts)
}

// Helper functions

func (ca *ContextAnalyzer) findValidationMessages(code string) []string {
	var messages []string
	
	// Look for common validation message patterns
	patterns := []string{
		`["'][^"']*required[^"']*["']`,
		`["'][^"']*invalid[^"']*["']`,
		`["'][^"']*must be[^"']*["']`,
		`["'][^"']*please[^"']*["']`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			clean := strings.Trim(match, `"'`)
			if len(clean) > 5 { // Filter out very short matches
				messages = append(messages, clean)
			}
		}
	}
	
	return messages
}

func (ca *ContextAnalyzer) findUserMessages(code string) []string {
	var messages []string
	
	// Look for user-facing message patterns
	userMessageKeywords := []string{
		"success", "error", "welcome", "thank you", "please",
		"invalid", "required", "sent", "created", "updated",
		"failed", "check your", "try again",
	}
	
	// Find string literals containing user message keywords
	stringPattern := regexp.MustCompile(`["']([^"']{10,}?)["']`) // Strings with 10+ chars
	matches := stringPattern.FindAllStringSubmatch(code, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			text := match[1]
			lowerText := strings.ToLower(text)
			
			for _, keyword := range userMessageKeywords {
				if strings.Contains(lowerText, keyword) {
					messages = append(messages, text)
					break
				}
			}
		}
	}
	
	return messages
}

func (ca *ContextAnalyzer) extractSchemaFields(code string) []string {
	var fields []string
	
	// Look for schema definitions (Joi, Yup, Zod, etc.)
	schemaPatterns := []string{
		`(\w+):\s*Joi\.\w+`,
		`(\w+):\s*yup\.\w+`,
		`(\w+):\s*z\.\w+`,
		`(\w+):\s*schema\.\w+`,
	}
	
	for _, pattern := range schemaPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				fields = append(fields, match[1])
			}
		}
	}
	
	return fields
}

func (ca *ContextAnalyzer) deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, str := range strs {
		if !seen[str] && str != "" {
			seen[str] = true
			result = append(result, str)
		}
	}
	
	return result
}

func isBoilerplateComment(comment string) bool {
	boilerplate := []string{
		"TODO", "FIXME", "HACK", "NOTE",
		"eslint-disable", "prettier-ignore",
		"Generated by", "Auto-generated",
		"@param", "@return", "@throws",
	}
	
	lowerComment := strings.ToLower(comment)
	for _, bp := range boilerplate {
		if strings.Contains(lowerComment, strings.ToLower(bp)) {
			return true
		}
	}
	
	return false
}

func isGenericText(text string) bool {
	generic := []string{
		"button", "click", "submit", "cancel",
		"ok", "yes", "no", "close", "save",
	}
	
	lowerText := strings.ToLower(strings.TrimSpace(text))
	for _, g := range generic {
		if lowerText == g {
			return true
		}
	}
	
	return false
}