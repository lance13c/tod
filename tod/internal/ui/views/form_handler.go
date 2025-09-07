package views

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/browser"
	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/logging"
)

// FormFieldType represents the type of form field
type FormFieldType int

const (
	EmailField FormFieldType = iota
	PasswordField
	UsernameField
	SubmitButton
	TextInput
	Unknown
)

// FormField represents a detected form field
type FormField struct {
	Type        FormFieldType
	Selector    string
	Name        string
	Label       string
	Placeholder string
	Required    bool
	Value       string
	IsVisible   bool
}

// LoginForm represents a detected login form
type LoginForm struct {
	URL           string
	Domain        string
	EmailField    *FormField
	PasswordField *FormField
	UsernameField *FormField
	SubmitButton  *FormField
	OtherFields   []FormField
	IsComplete    bool
	IsMagicLink   bool // True if form only has email field
}

// FormHandler manages form detection and interaction
type FormHandler struct {
	chromeDPManager *browser.ChromeDPManager
	currentForm     *LoginForm
	savedUsers      []config.TestUser
	domain          string
}

// NewFormHandler creates a new form handler
func NewFormHandler(manager *browser.ChromeDPManager) *FormHandler {
	return &FormHandler{
		chromeDPManager: manager,
		savedUsers:      []config.TestUser{},
	}
}

// DetectLoginForm analyzes the current page for login forms
func (f *FormHandler) DetectLoginForm() (*LoginForm, error) {
	if f.chromeDPManager == nil {
		return nil, fmt.Errorf("Chrome not connected")
	}

	// Get current page info
	currentURL, _, err := f.chromeDPManager.GetPageInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}

	parsedURL, err := url.Parse(currentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	f.domain = parsedURL.Host

	// Detect form fields using JavaScript
	formFields, err := f.extractFormFields()
	if err != nil {
		return nil, fmt.Errorf("failed to extract form fields: %w", err)
	}

	// Build login form structure
	form := &LoginForm{
		URL:    currentURL,
		Domain: f.domain,
	}

	for _, field := range formFields {
		switch field.Type {
		case EmailField:
			form.EmailField = &field
		case PasswordField:
			form.PasswordField = &field
		case UsernameField:
			form.UsernameField = &field
		case SubmitButton:
			form.SubmitButton = &field
		default:
			form.OtherFields = append(form.OtherFields, field)
		}
	}

	// Determine if this is a magic link form (email only)
	form.IsMagicLink = form.EmailField != nil && form.PasswordField == nil

	// Check if form is complete (has required fields)
	form.IsComplete = f.isFormComplete(form)

	f.currentForm = form
	return form, nil
}

// extractFormFields uses JavaScript to extract form fields from the page
func (f *FormHandler) extractFormFields() ([]FormField, error) {
	script := `
		(() => {
			const fields = [];
			
			// Look for form inputs
			const inputs = document.querySelectorAll('input, button[type="submit"]');
			
			inputs.forEach(input => {
				// Skip hidden inputs
				if (input.type === 'hidden' || input.style.display === 'none') {
					return;
				}
				
				// Skip if not visible
				if (input.offsetParent === null) {
					return;
				}
				
				const field = {
					selector: '',
					name: input.name || '',
					type: input.type || '',
					placeholder: input.placeholder || '',
					value: input.value || '',
					required: input.required || false,
					tagName: input.tagName.toLowerCase(),
					id: input.id || '',
					className: input.className || '',
					label: ''
				};
				
				// Generate selector - prefer ID, then name, then class
				if (input.id) {
					field.selector = '#' + input.id;
				} else if (input.name) {
					field.selector = 'input[name="' + input.name + '"]';
				} else if (input.className) {
					const classes = input.className.split(' ').filter(c => c && !c.includes('css-'));
					if (classes.length > 0) {
						field.selector = '.' + classes[0];
					}
				} else {
					field.selector = input.tagName.toLowerCase();
				}
				
				// Find associated label
				const labels = document.querySelectorAll('label');
				for (let label of labels) {
					if (label.getAttribute('for') === input.id || 
						label.contains(input) ||
						(input.name && label.textContent.toLowerCase().includes(input.name.toLowerCase()))) {
						field.label = label.textContent.trim();
						break;
					}
				}
				
				fields.push(field);
			});
			
			return fields;
		})()
	`

	var rawFields []map[string]interface{}
	if err := f.chromeDPManager.ExecuteScript(script, &rawFields); err != nil {
		return nil, err
	}

	var fields []FormField
	for _, raw := range rawFields {
		field := FormField{
			Selector:    getStringValue(raw["selector"]),
			Name:        getStringValue(raw["name"]),
			Placeholder: getStringValue(raw["placeholder"]),
			Value:       getStringValue(raw["value"]),
			Required:    getBoolValue(raw["required"]),
			IsVisible:   true,
		}

		// Determine field type based on input type, name, placeholder, and label
		field.Type = f.determineFieldType(raw)
		field.Label = f.generateFieldLabel(raw, field.Type)

		fields = append(fields, field)
	}

	return fields, nil
}

// determineFieldType determines the type of form field
func (f *FormHandler) determineFieldType(raw map[string]interface{}) FormFieldType {
	inputType := strings.ToLower(getStringValue(raw["type"]))
	name := strings.ToLower(getStringValue(raw["name"]))
	placeholder := strings.ToLower(getStringValue(raw["placeholder"]))
	label := strings.ToLower(getStringValue(raw["label"]))
	tagName := strings.ToLower(getStringValue(raw["tagName"]))

	// Check for submit button
	if inputType == "submit" || tagName == "button" {
		return SubmitButton
	}

	// Check for password field
	if inputType == "password" || 
		strings.Contains(name, "password") || 
		strings.Contains(placeholder, "password") ||
		strings.Contains(label, "password") {
		return PasswordField
	}

	// Check for email field
	if inputType == "email" ||
		strings.Contains(name, "email") ||
		strings.Contains(placeholder, "email") ||
		strings.Contains(label, "email") ||
		strings.Contains(placeholder, "@") {
		return EmailField
	}

	// Check for username field
	if strings.Contains(name, "username") ||
		strings.Contains(name, "user") ||
		strings.Contains(placeholder, "username") ||
		strings.Contains(label, "username") {
		return UsernameField
	}

	// Check for login-related text inputs
	if inputType == "text" &&
		(strings.Contains(name, "login") ||
		 strings.Contains(placeholder, "login") ||
		 strings.Contains(label, "login")) {
		return UsernameField
	}

	if inputType == "text" || inputType == "" {
		return TextInput
	}

	return Unknown
}

// generateFieldLabel generates a human-readable label for the field
func (f *FormHandler) generateFieldLabel(raw map[string]interface{}, fieldType FormFieldType) string {
	label := getStringValue(raw["label"])
	if label != "" {
		return label
	}

	placeholder := getStringValue(raw["placeholder"])
	if placeholder != "" {
		return placeholder
	}

	name := getStringValue(raw["name"])
	if name != "" {
		return strings.Title(strings.ReplaceAll(name, "_", " "))
	}

	// Default labels based on field type
	switch fieldType {
	case EmailField:
		return "Email Address"
	case PasswordField:
		return "Password"
	case UsernameField:
		return "Username"
	case SubmitButton:
		return "Submit"
	default:
		return "Input"
	}
}

// isFormComplete checks if the form has the required fields for login
func (f *FormHandler) isFormComplete(form *LoginForm) bool {
	// Must have a submit button
	if form.SubmitButton == nil {
		return false
	}

	// Must have either email or username
	hasIdentifier := form.EmailField != nil || form.UsernameField != nil
	if !hasIdentifier {
		return false
	}

	return true
}

// FillField fills a form field with the given value
func (f *FormHandler) FillField(field *FormField, value string) error {
	if f.chromeDPManager == nil {
		return fmt.Errorf("Chrome not connected")
	}

	if field == nil || field.Selector == "" {
		return fmt.Errorf("invalid field")
	}

	// Wait for the field to be visible
	if err := f.chromeDPManager.WaitForElement(field.Selector); err != nil {
		return fmt.Errorf("field not found: %w", err)
	}

	// Clear the field first
	if err := f.clearField(field.Selector); err != nil {
		logging.Debug("Warning: failed to clear field %s: %v", field.Selector, err)
	}

	// Fill the field
	if err := f.chromeDPManager.SendKeys(field.Selector, value); err != nil {
		return fmt.Errorf("failed to fill field: %w", err)
	}

	field.Value = value
	return nil
}

// clearField clears a form field
func (f *FormHandler) clearField(selector string) error {
	script := fmt.Sprintf(`
		const element = document.querySelector('%s');
		if (element) {
			element.value = '';
			element.dispatchEvent(new Event('input', { bubbles: true }));
			element.dispatchEvent(new Event('change', { bubbles: true }));
		}
	`, selector)

	return f.chromeDPManager.ExecuteScript(script, nil)
}

// SubmitForm submits the current form
func (f *FormHandler) SubmitForm() error {
	if f.currentForm == nil || f.currentForm.SubmitButton == nil {
		return fmt.Errorf("no form or submit button found")
	}

	// Click the submit button
	if err := f.chromeDPManager.Click(f.currentForm.SubmitButton.Selector); err != nil {
		return fmt.Errorf("failed to submit form: %w", err)
	}

	return nil
}

// WaitForPageChange waits for the page to change after form submission
func (f *FormHandler) WaitForPageChange(timeout time.Duration) (*PageChangeResult, error) {
	if f.chromeDPManager == nil {
		return nil, fmt.Errorf("Chrome not connected")
	}

	initialURL, _, err := f.chromeDPManager.GetPageInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial page info: %w", err)
	}

	// Monitor for changes using polling
	changes := f.chromeDPManager.PollForChanges(timeout, 500*time.Millisecond, 100*time.Millisecond)
	
	result := &PageChangeResult{
		InitialURL: initialURL,
		Success:    false,
	}

	for change := range changes {
		// Get current page info
		currentURL, title, err := f.chromeDPManager.GetPageInfo()
		if err != nil {
			logging.Debug("Warning: failed to get page info during monitoring: %v", err)
			continue
		}

		result.FinalURL = currentURL
		result.FinalTitle = title

		// Check if URL changed (navigation occurred)
		if currentURL != initialURL {
			result.Success = true
			result.NavigationOccurred = true
			result.Message = "Page navigation detected"
			break
		}

		// Check for magic link messages in the content
		if f.containsMagicLinkMessage(change.HTML) {
			result.Success = true
			result.MagicLinkSent = true
			result.Message = "Magic link sent - check email"
			break
		}

		// Check for error messages
		if f.containsErrorMessage(change.HTML) {
			result.Success = false
			result.ErrorDetected = true
			result.Message = "Form submission error detected"
			break
		}
	}

	return result, nil
}

// containsMagicLinkMessage checks if the HTML contains magic link success messages
func (f *FormHandler) containsMagicLinkMessage(html string) bool {
	html = strings.ToLower(html)
	magicLinkPhrases := []string{
		"magic link sent",
		"check your email",
		"email sent",
		"sign in link sent",
		"login link sent",
		"we sent you",
		"check your inbox",
		"email has been sent",
		"sign-in link",
		"login link",
	}

	for _, phrase := range magicLinkPhrases {
		if strings.Contains(html, phrase) {
			return true
		}
	}

	return false
}

// containsErrorMessage checks if the HTML contains error messages
func (f *FormHandler) containsErrorMessage(html string) bool {
	html = strings.ToLower(html)
	errorPhrases := []string{
		"error",
		"invalid",
		"incorrect",
		"failed",
		"wrong",
		"not found",
		"unauthorized",
		"access denied",
		"login failed",
		"authentication failed",
	}

	for _, phrase := range errorPhrases {
		if strings.Contains(html, phrase) {
			return true
		}
	}

	return false
}

// GetCurrentForm returns the currently detected form
func (f *FormHandler) GetCurrentForm() *LoginForm {
	return f.currentForm
}

// GetDomain returns the current domain
func (f *FormHandler) GetDomain() string {
	return f.domain
}

// PageChangeResult represents the result of waiting for page changes
type PageChangeResult struct {
	InitialURL          string
	FinalURL            string
	FinalTitle          string
	Success             bool
	NavigationOccurred  bool
	MagicLinkSent       bool
	ErrorDetected       bool
	Message             string
}

// Helper functions (reused from chromedp_manager.go)
func getStringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getBoolValue(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}