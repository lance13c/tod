package email

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/llm"
)

// ExtractorService handles LLM-powered extraction of authentication data from emails
type ExtractorService struct {
	llmClient llm.Client
}

// ExtractionResult contains the result of LLM-based email content extraction
type ExtractionResult struct {
	Success    bool    `json:"success"`
	Type       string  `json:"type"`        // "verification_code", "magic_link", "2fa_code"
	Value      string  `json:"value"`       // The extracted value
	Confidence float64 `json:"confidence"`  // Confidence score from LLM
	Email      *Email  `json:"email,omitempty"` // Source email
	Error      string  `json:"error,omitempty"`
}

// AuthType represents different types of authentication data to extract
type AuthType string

const (
	AuthTypeMagicLink        AuthType = "magic_link"
	AuthTypeVerificationCode AuthType = "verification_code"
	AuthType2FA              AuthType = "2fa_code"
	AuthTypeSMS              AuthType = "sms_code"
)

// NewExtractorService creates a new email extraction service
func NewExtractorService(llmClient llm.Client) *ExtractorService {
	return &ExtractorService{
		llmClient: llmClient,
	}
}

// ExtractAuthData extracts authentication data from recent emails using LLM
func (e *ExtractorService) ExtractAuthData(emails []*Email, authType AuthType, context string) (*ExtractionResult, error) {
	if len(emails) == 0 {
		return &ExtractionResult{
			Success: false,
			Error:   "no emails found",
		}, nil
	}

	// Try each email until we find authentication data
	for _, email := range emails {
		result := e.extractFromSingleEmail(email, authType, context)
		if result.Success {
			return result, nil
		}
	}

	return &ExtractionResult{
		Success: false,
		Error:   fmt.Sprintf("no %s found in %d recent emails", authType, len(emails)),
	}, nil
}

// extractFromSingleEmail attempts to extract auth data from a single email using LLM
func (e *ExtractorService) extractFromSingleEmail(email *Email, authType AuthType, context string) *ExtractionResult {
	// For now, let's also try simple regex patterns as fallback
	// but prioritize LLM analysis for generic extraction
	if fallbackResult := e.tryFallbackExtraction(email, authType); fallbackResult.Success {
		fallbackResult.Email = email
		return fallbackResult
	}

	// TODO: When LLM integration is ready, use it here
	// Build LLM prompt based on auth type
	_ = e.buildExtractionPrompt(email, authType, context)
	
	// For now, return fallback result even if not successful
	result := &ExtractionResult{
		Success: false,
		Type:    string(authType),
		Email:   email,
		Error:   "LLM extraction not yet implemented - using pattern matching only",
	}

	return result
}

// tryFallbackExtraction uses regex patterns as fallback when LLM is not available
func (e *ExtractorService) tryFallbackExtraction(email *Email, authType AuthType) *ExtractionResult {
	switch authType {
	case AuthTypeMagicLink:
		return e.extractMagicLink(email)
	case AuthTypeVerificationCode, AuthType2FA, AuthTypeSMS:
		return e.extractVerificationCode(email)
	}

	return &ExtractionResult{
		Success: false,
		Type:    string(authType),
		Error:   "unsupported auth type",
	}
}

// extractMagicLink finds URLs in email content that look like magic links
func (e *ExtractorService) extractMagicLink(email *Email) *ExtractionResult {
	// Look for URLs in the email body
	urlRegex := regexp.MustCompile(`https?://[^\s<>"']+(?:verify|auth|login|confirm|activate|magic)[^\s<>"']*`)
	
	content := email.Body + " " + email.Snippet
	matches := urlRegex.FindAllString(content, -1)
	
	if len(matches) > 0 {
		// Return the first URL found
		return &ExtractionResult{
			Success:    true,
			Type:       string(AuthTypeMagicLink),
			Value:      matches[0],
			Confidence: 0.8, // High confidence for URL pattern match
		}
	}

	// Also look for any HTTPS URL as potential magic link
	generalURLRegex := regexp.MustCompile(`https://[^\s<>"']+`)
	generalMatches := generalURLRegex.FindAllString(content, -1)
	
	if len(generalMatches) > 0 {
		// Filter out common non-auth URLs
		for _, url := range generalMatches {
			if !strings.Contains(strings.ToLower(url), "unsubscribe") &&
			   !strings.Contains(strings.ToLower(url), "privacy") &&
			   !strings.Contains(strings.ToLower(url), "terms") {
				return &ExtractionResult{
					Success:    true,
					Type:       string(AuthTypeMagicLink),
					Value:      url,
					Confidence: 0.6, // Lower confidence for general URL
				}
			}
		}
	}

	return &ExtractionResult{
		Success: false,
		Type:    string(AuthTypeMagicLink),
		Error:   "no magic link URLs found",
	}
}

// extractVerificationCode finds numeric codes in email content
func (e *ExtractorService) extractVerificationCode(email *Email) *ExtractionResult {
	content := email.Subject + " " + email.Body + " " + email.Snippet
	
	// Look for common verification code patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(\d{6})\b`),                    // 6-digit code
		regexp.MustCompile(`\b(\d{4})\b`),                    // 4-digit code  
		regexp.MustCompile(`\b(\d{8})\b`),                    // 8-digit code
		regexp.MustCompile(`code[:\s]+(\d{4,8})`),            // "code: 123456"
		regexp.MustCompile(`verification[:\s]+(\d{4,8})`),    // "verification: 123456"
		regexp.MustCompile(`token[:\s]+(\d{4,8})`),           // "token: 123456"
	}

	for i, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		if len(matches) > 0 {
			code := matches[0][1]
			confidence := 0.9 - float64(i)*0.1 // Higher confidence for more specific patterns
			
			return &ExtractionResult{
				Success:    true,
				Type:       string(AuthTypeVerificationCode),
				Value:      code,
				Confidence: confidence,
			}
		}
	}

	return &ExtractionResult{
		Success: false,
		Type:    string(AuthTypeVerificationCode),
		Error:   "no verification code found",
	}
}

// buildExtractionPrompt creates an LLM prompt for extracting auth data
func (e *ExtractorService) buildExtractionPrompt(email *Email, authType AuthType, context string) string {
	basePrompt := fmt.Sprintf(`
You are analyzing an email to extract authentication information.

Context: %s
Email Subject: %s
Email From: %s  
Email Content: %s

Task: Extract the %s from this email.
`, context, email.Subject, email.From, email.Body, authType)

	switch authType {
	case AuthTypeMagicLink:
		return basePrompt + `
Look for URLs in the email that are likely to be magic login links.
Magic links typically contain words like 'verify', 'login', 'auth', 'confirm', or 'activate'.
Return only the complete URL, nothing else.
If no magic link is found, return "NOT_FOUND".`

	case AuthTypeVerificationCode, AuthType2FA, AuthTypeSMS:
		return basePrompt + `
Look for numeric verification codes in the email.
These are usually 4-8 digit numbers, often labeled as "code", "verification code", "token", or similar.
Return only the numeric code, nothing else.  
If no code is found, return "NOT_FOUND".`

	default:
		return basePrompt + `
Extract the relevant authentication information from this email.
Return only the extracted value, nothing else.
If nothing is found, return "NOT_FOUND".`
	}
}

// WaitForAuthEmail polls for authentication emails and extracts data
func (e *ExtractorService) WaitForAuthEmail(client *Client, authType AuthType, context string, timeout time.Duration) (*ExtractionResult, error) {
	startTime := time.Now()
	checkInterval := 2 * time.Second

	for time.Since(startTime) < timeout {
		// Get emails from the last minute
		emails, err := client.GetRecentEmails(1 * time.Minute)
		if err != nil {
			time.Sleep(checkInterval)
			continue
		}

		// Try to extract auth data
		result, err := e.ExtractAuthData(emails, authType, context)
		if err != nil {
			time.Sleep(checkInterval)
			continue
		}

		if result.Success {
			return result, nil
		}

		time.Sleep(checkInterval)
	}

	return &ExtractionResult{
		Success: false,
		Error:   fmt.Sprintf("timeout waiting for %s email", authType),
	}, nil
}

// WaitForAuthEmailContinuous polls for authentication emails continuously with configurable interval
func (e *ExtractorService) WaitForAuthEmailContinuous(client *Client, authType AuthType, context string, checkInterval time.Duration, maxTimeout time.Duration) (*ExtractionResult, error) {
	startTime := time.Now()
	
	// Default check interval to 5 seconds if not specified
	if checkInterval == 0 {
		checkInterval = 5 * time.Second
	}
	
	// Default max timeout to 2 minutes if not specified
	if maxTimeout == 0 {
		maxTimeout = 2 * time.Minute
	}

	for time.Since(startTime) < maxTimeout {
		// Get emails from the last 2 minutes (wider window for continuous scanning)
		emails, err := client.GetRecentEmails(2 * time.Minute)
		if err != nil {
			time.Sleep(checkInterval)
			continue
		}

		// Try to extract auth data
		result, err := e.ExtractAuthData(emails, authType, context)
		if err != nil {
			time.Sleep(checkInterval)
			continue
		}

		if result.Success {
			return result, nil
		}

		time.Sleep(checkInterval)
	}

	return &ExtractionResult{
		Success: false,
		Error:   fmt.Sprintf("timeout waiting for %s email after %v", authType, maxTimeout),
	}, nil
}