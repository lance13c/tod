package testing

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lance13c/tod/internal/browser"
	"github.com/lance13c/tod/internal/llm"
)

// ActionDiscovery handles discovering untested actions from HTML
type ActionDiscovery struct {
	llmClient   llm.Client
	projectRoot string
}

// NewActionDiscovery creates a new action discovery instance
func NewActionDiscovery(llmClient llm.Client, projectRoot string) *ActionDiscovery {
	return &ActionDiscovery{
		llmClient:   llmClient,
		projectRoot: projectRoot,
	}
}

// DiscoveredAction represents an action found in the HTML that may need testing
type DiscoveredAction struct {
	Description  string `json:"description"`
	Element      string `json:"element"`
	Selector     string `json:"selector"`
	Action       string `json:"action"`
	TestScenario string `json:"test_scenario"`
	IsTested     bool   `json:"is_tested"`
	Priority     string `json:"priority"` // high, medium, low
	JavaScript   string `json:"javascript"` // Executable JavaScript code
	UserInput    string `json:"user_input"` // The exact user input for this action
}

// DiscoverActionsFromHTML analyzes HTML and finds untested user actions
func (ad *ActionDiscovery) DiscoverActionsFromHTML(ctx context.Context, htmlContent string, existingTests []string) ([]DiscoveredAction, string, string, error) {
	return ad.DiscoverActionsFromHTMLWithContext(ctx, htmlContent, existingTests, "")
}

// DiscoverActionsFromHTMLWithContext analyzes HTML with optional user context
func (ad *ActionDiscovery) DiscoverActionsFromHTMLWithContext(ctx context.Context, htmlContent string, existingTests []string, userContext string) ([]DiscoveredAction, string, string, error) {
	// Simplify HTML first
	simplifiedHTML, err := browser.SimplifyHTML(htmlContent)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to simplify HTML: %w", err)
	}

	// Extract interactive elements
	elements, err := browser.ExtractInteractiveElements(htmlContent)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to extract elements: %w", err)
	}

	// Build prompt for LLM
	prompt := ad.buildDiscoveryPromptWithContext(simplifiedHTML, elements, existingTests, userContext)

	// Use AnalyzeCode method with the HTML as "code"
	analysis, err := ad.llmClient.AnalyzeCode(ctx, prompt, "page.html")
	if err != nil {
		return nil, prompt, "", fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse LLM response from the analysis notes
	response := analysis.Notes
	actions := ad.parseActionsFromResponse(response)

	// Check against existing tests
	for i := range actions {
		actions[i].IsTested = ad.isActionTested(actions[i], existingTests)
	}

	return actions, prompt, response, nil
}

// buildDiscoveryPrompt creates the LLM prompt for action discovery
func (ad *ActionDiscovery) buildDiscoveryPrompt(simplifiedHTML string, elements []browser.InteractiveElement, existingTests []string) string {
	return ad.buildDiscoveryPromptWithContext(simplifiedHTML, elements, existingTests, "")
}

// buildDiscoveryPromptWithContext creates the LLM prompt with optional user context
func (ad *ActionDiscovery) buildDiscoveryPromptWithContext(simplifiedHTML string, elements []browser.InteractiveElement, existingTests []string, userContext string) string {
	var prompt strings.Builder

	prompt.WriteString("You are a test automation expert analyzing a web page to identify possible user actions.\n\n")
	
	// Add user context if provided
	if userContext != "" {
		prompt.WriteString("USER CONTEXT:\n")
		prompt.WriteString(userContext)
		prompt.WriteString("\n\n")
	}
	
	prompt.WriteString("PAGE ANALYSIS:\n")
	prompt.WriteString(fmt.Sprintf("Found %d interactive elements on the page.\n\n", len(elements)))
	
	prompt.WriteString("KEY ELEMENTS:\n")
	// Show first 20 most important elements
	count := 0
	for _, elem := range elements {
		if count >= 20 {
			break
		}
		if elem.Text != "" || elem.Href != "" {
			prompt.WriteString(fmt.Sprintf("- %s: \"%s\"\n", elem.Tag, elem.Text))
			count++
		}
	}
	prompt.WriteString("\n")

	prompt.WriteString("HTML STRUCTURE (simplified):\n")
	// Limit HTML to first 3000 chars for initial discovery
	if len(simplifiedHTML) > 3000 {
		prompt.WriteString(simplifiedHTML[:3000])
		prompt.WriteString("\n... (truncated)\n")
	} else {
		prompt.WriteString(simplifiedHTML)
	}
	prompt.WriteString("\n\n")

	prompt.WriteString("TASK:\n")
	prompt.WriteString("Identify the 5 most important user actions on this page.\n")
	prompt.WriteString("Focus on actions that users would typically perform.\n")
	
	prompt.WriteString("PRIORITIZE IN THIS ORDER:\n")
	prompt.WriteString("1. NAVIGATION LINKS - Any <a> tags with href attributes that take users to different pages\n")
	prompt.WriteString("2. PRIMARY ACTIONS - Main buttons and CTAs (login, signup, start trial, etc.)\n") 
	prompt.WriteString("3. FORM INTERACTIONS - Input fields, dropdowns, form submissions\n")
	prompt.WriteString("4. SECONDARY ACTIONS - Support links, social media, less critical buttons\n\n")
	
	prompt.WriteString("NAVIGATION IDENTIFICATION:\n")
	prompt.WriteString("- Look for <a> elements with href attributes\n")
	prompt.WriteString("- Common navigation patterns: 'About', 'Features', 'Pricing', 'Contact', 'Blog'\n")
	prompt.WriteString("- Menu items, header/footer links, breadcrumbs\n")
	prompt.WriteString("- Use natural language: 'Go to pricing page' NOT 'Click pricing link'\n\n")

	prompt.WriteString("For each action, provide a simple, natural description and priority.\n")
	prompt.WriteString("Keep descriptions conversational and clear.\n")
	prompt.WriteString("Format: ACTION_DESCRIPTION | priority\n")
	prompt.WriteString("Example:\n")
	prompt.WriteString("Go to the pricing page | high\n")
	prompt.WriteString("Navigate to features section | high\n")
	prompt.WriteString("Sign in with your account | high\n")
	prompt.WriteString("Click the Start Sharing Now button | medium\n")
	prompt.WriteString("View contact information | low\n\n")

	prompt.WriteString("List the top 5 actions (no numbers):")

	return prompt.String()
}

// parseActionsFromResponse parses the LLM response into discovered actions
func (ad *ActionDiscovery) parseActionsFromResponse(response string) []DiscoveredAction {
	var actions []DiscoveredAction

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove numbering if present
		if idx := strings.IndexAny(line, "0123456789"); idx == 0 {
			// Skip past number and any following punctuation
			for i, ch := range line {
				if ch != '.' && ch != ')' && ch != ' ' && (ch < '0' || ch > '9') {
					line = line[i:]
					break
				}
			}
		}

		// Parse the simplified format: DESCRIPTION | priority
		parts := strings.Split(line, "|")
		if len(parts) >= 1 {
			description := strings.TrimSpace(parts[0])
			priority := "medium"
			
			if len(parts) >= 2 {
				priority = strings.ToLower(strings.TrimSpace(parts[1]))
			}

			// Create action with just description and priority
			// Selector and JavaScript will be generated later
			action := DiscoveredAction{
				Description: description,
				Priority:    priority,
				Action:      "pending", // Will be determined when generating code
			}

			actions = append(actions, action)
		}
	}

	return actions
}

// cleanDescription cleans up action descriptions
func (ad *ActionDiscovery) cleanDescription(desc string) string {
	// Remove underscores and convert to sentence case
	desc = strings.ReplaceAll(desc, "_", " ")
	desc = strings.ToLower(desc)
	if len(desc) > 0 {
		desc = strings.ToUpper(string(desc[0])) + desc[1:]
	}
	return desc
}

// isActionTested checks if an action is already covered by existing tests
func (ad *ActionDiscovery) isActionTested(action DiscoveredAction, existingTests []string) bool {
	// Simple heuristic: check if selector or key terms appear in test files
	searchTerms := []string{
		action.Selector,
		strings.ToLower(action.Description),
	}

	// Extract key terms from selector
	if strings.Contains(action.Selector, "data-testid=") {
		start := strings.Index(action.Selector, "'")
		if start >= 0 {
			end := strings.Index(action.Selector[start+1:], "'")
			if end >= 0 {
				testId := action.Selector[start+1 : start+1+end]
				searchTerms = append(searchTerms, testId)
			}
		}
	}

	// Check each test file content
	for _, testContent := range existingTests {
		testLower := strings.ToLower(testContent)
		for _, term := range searchTerms {
			if term != "" && strings.Contains(testLower, strings.ToLower(term)) {
				return true
			}
		}
	}

	return false
}

// GenerateTestSuggestions creates test code suggestions for discovered actions
func (ad *ActionDiscovery) GenerateTestSuggestions(ctx context.Context, actions []DiscoveredAction, framework string) (string, error) {
	// Setup logging
	logFile, err := os.OpenFile(".tod/api_calls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("GenerateTestSuggestions called with %d actions, framework: %s", len(actions), framework)
	}

	// Filter to only untested actions
	untestedActions := []DiscoveredAction{}
	for _, action := range actions {
		if !action.IsTested {
			untestedActions = append(untestedActions, action)
		}
	}

	if logFile != nil {
		logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Filtered to %d untested actions", len(untestedActions))
	}

	if len(untestedActions) == 0 {
		if logFile != nil {
			logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("No untested actions found, returning empty string")
		}
		return "", nil
	}

	// Build prompt for test generation
	prompt := ad.buildTestGenerationPrompt(untestedActions, framework)

	if logFile != nil {
		logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("Built prompt for test generation, length: %d", len(prompt))
		logger.Printf("Calling LLM API with prompt:\n%s", prompt)
	}

	// Get test suggestions from LLM using AnalyzeCode
	analysis, err := ad.llmClient.AnalyzeCode(ctx, prompt, "test-generation.txt")
	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
			logger.Printf("ERROR from LLM API: %v", err)
		}
		return "", fmt.Errorf("failed to generate test suggestions: %w", err)
	}

	if logFile != nil {
		logger := log.New(logFile, "[ACTION_DISCOVERY] ", log.LstdFlags|log.Lmicroseconds)
		logger.Printf("LLM API returned successfully, response length: %d", len(analysis.Notes))
		logger.Printf("Response: %s", analysis.Notes)
	}

	return analysis.Notes, nil
}

// GenerateActionCode generates executable JavaScript for a specific action
func (ad *ActionDiscovery) GenerateActionCode(ctx context.Context, action DiscoveredAction, htmlContent string) (DiscoveredAction, error) {
	// Build prompt for code generation
	prompt := ad.buildCodeGenerationPrompt(action, htmlContent)
	
	// Get code from LLM
	analysis, err := ad.llmClient.AnalyzeCode(ctx, prompt, "action-code.js")
	if err != nil {
		return action, fmt.Errorf("failed to generate action code: %w", err)
	}
	
	// Parse the response to extract selector and JavaScript
	updatedAction := ad.parseCodeResponse(action, analysis.Notes)
	
	return updatedAction, nil
}

// buildCodeGenerationPrompt creates a prompt for generating executable code
func (ad *ActionDiscovery) buildCodeGenerationPrompt(action DiscoveredAction, htmlContent string) string {
	var prompt strings.Builder
	
	prompt.WriteString("Generate executable JavaScript code for the following user action:\n\n")
	
	// If user provided specific input, prioritize it
	if action.UserInput != "" {
		prompt.WriteString(fmt.Sprintf("USER REQUEST: %s\n", action.UserInput))
		prompt.WriteString("(This is what the user specifically asked to do - prioritize this)\n\n")
	}
	
	prompt.WriteString(fmt.Sprintf("DISCOVERED ACTION: %s\n", action.Description))
	prompt.WriteString(fmt.Sprintf("PRIORITY: %s\n\n", action.Priority))
	
	prompt.WriteString("HTML CONTEXT (relevant portion):\n")
	// Extract relevant HTML based on both description and user input
	searchText := action.Description
	if action.UserInput != "" {
		searchText = action.UserInput + " " + action.Description
	}
	relevantHTML := ad.extractRelevantHTML(searchText, htmlContent)
	prompt.WriteString(relevantHTML)
	prompt.WriteString("\n\n")
	
	prompt.WriteString("TASK:\n")
	prompt.WriteString("Generate ChromeDP-compatible JavaScript to perform this EXACT action.\n")
	prompt.WriteString("The code should:\n")
	prompt.WriteString("1. Find the correct element on the page\n")
	prompt.WriteString("2. Perform the appropriate action (click, type, etc.)\n")
	prompt.WriteString("3. Handle common edge cases\n")
	prompt.WriteString("4. Return true if successful, false if not\n\n")
	
	prompt.WriteString("Return a JSON object with:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"selector\": \"CSS selector or XPath\",\n")
	prompt.WriteString("  \"action\": \"click|type|select|navigate\",\n")
	prompt.WriteString("  \"javascript\": \"executable JavaScript code\",\n")
	prompt.WriteString("  \"fallback\": \"alternative JavaScript if primary fails\"\n")
	prompt.WriteString("}\n\n")
	
	prompt.WriteString("EXAMPLES:\n\n")
	
	prompt.WriteString("For navigation actions like 'Go to pricing page':\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"selector\": \"a:contains('Pricing')\",\n")
	prompt.WriteString("  \"action\": \"click\",\n")
	prompt.WriteString("  \"javascript\": \"(() => { const links = Array.from(document.querySelectorAll('a')); const el = links.find(a => a.textContent.toLowerCase().includes('pricing') || a.href.includes('pricing')); if (el) { el.click(); return true; } return false; })()\",\n")
	prompt.WriteString("  \"fallback\": \"(() => { const el = document.querySelector('a[href*=\\\"/pricing\\\"]') || document.querySelector('nav a[href*=\\\"price\\\"]'); if (el) { el.click(); return true; } return false; })()\"\n")
	prompt.WriteString("}\n\n")
	
	prompt.WriteString("For button actions like 'Click Sign In button':\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"selector\": \"button:contains('Sign In')\",\n")
	prompt.WriteString("  \"action\": \"click\",\n")
	prompt.WriteString("  \"javascript\": \"(() => { const el = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Sign In')); if (el) { el.click(); return true; } return false; })()\",\n")
	prompt.WriteString("  \"fallback\": \"(() => { const el = document.querySelector('[aria-label*=\\\"sign\\\"]') || document.querySelector('a[href*=\\\"/signin\\\"]'); if (el) { el.click(); return true; } return false; })()\"\n")
	prompt.WriteString("}\n\n")
	
	prompt.WriteString("IMPORTANT FOR NAVIGATION:\n")
	prompt.WriteString("- Always prioritize <a> elements for navigation\n")
	prompt.WriteString("- Check both textContent and href attribute for matches\n") 
	prompt.WriteString("- Use case-insensitive matching\n")
	prompt.WriteString("- Include fallback selectors for href patterns\n\n")
	
	prompt.WriteString("Generate the JSON for the requested action:")
	
	return prompt.String()
}

// extractRelevantHTML extracts HTML around elements mentioned in the action
func (ad *ActionDiscovery) extractRelevantHTML(description, htmlContent string) string {
	// Simple extraction based on keywords in description
	keywords := strings.Fields(strings.ToLower(description))
	
	lines := strings.Split(htmlContent, "\n")
	relevant := []string{}
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, keyword := range keywords {
			// Skip common words
			if len(keyword) < 4 || keyword == "click" || keyword == "button" || keyword == "with" {
				continue
			}
			if strings.Contains(lineLower, keyword) {
				relevant = append(relevant, strings.TrimSpace(line))
				break
			}
		}
		if len(relevant) > 20 {
			break
		}
	}
	
	if len(relevant) == 0 {
		// Return first 500 chars if no matches
		if len(htmlContent) > 500 {
			return htmlContent[:500] + "..."
		}
		return htmlContent
	}
	
	return strings.Join(relevant, "\n")
}

// parseCodeResponse parses the LLM response containing JavaScript code
func (ad *ActionDiscovery) parseCodeResponse(action DiscoveredAction, response string) DiscoveredAction {
	// Try to parse as JSON
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		
		// Simple JSON parsing (could use encoding/json for robustness)
		if selector := ad.extractJSONField(jsonStr, "selector"); selector != "" {
			action.Selector = selector
		}
		if actionType := ad.extractJSONField(jsonStr, "action"); actionType != "" {
			action.Action = actionType
		}
		if javascript := ad.extractJSONField(jsonStr, "javascript"); javascript != "" {
			action.JavaScript = javascript
		} else if fallback := ad.extractJSONField(jsonStr, "fallback"); fallback != "" {
			action.JavaScript = fallback
		}
	}
	
	// If no JavaScript was extracted, create a simple one based on description
	if action.JavaScript == "" {
		action.JavaScript = ad.generateFallbackJavaScript(action.Description)
	}
	
	return action
}

// extractJSONField extracts a field value from JSON string
func (ad *ActionDiscovery) extractJSONField(json, field string) string {
	pattern := fmt.Sprintf("\"%s\"\\s*:\\s*\"", field)
	start := strings.Index(json, pattern)
	if start < 0 {
		return ""
	}
	
	start += len(pattern)
	end := start
	escaped := false
	
	for end < len(json) {
		if json[end] == '\\' && !escaped {
			escaped = true
		} else if json[end] == '"' && !escaped {
			break
		} else {
			escaped = false
		}
		end++
	}
	
	if end > start {
		value := json[start:end]
		// Unescape quotes
		value = strings.ReplaceAll(value, "\\\"", "\"")
		value = strings.ReplaceAll(value, "\\\\", "\\")
		return value
	}
	
	return ""
}

// generateFallbackJavaScript generates simple JavaScript based on action description
func (ad *ActionDiscovery) generateFallbackJavaScript(description string) string {
	descLower := strings.ToLower(description)
	
	// Extract text to search for
	var searchText string
	if strings.Contains(descLower, "'") {
		start := strings.Index(descLower, "'")
		end := strings.LastIndex(descLower, "'")
		if end > start {
			searchText = description[start+1 : end]
		}
	} else {
		// Use key words from description
		words := strings.Fields(description)
		for _, word := range words {
			if len(word) > 4 && !strings.Contains(strings.ToLower(word), "click") {
				searchText = word
				break
			}
		}
	}
	
	if searchText == "" {
		searchText = "Submit"
	}
	
	// Generate generic click JavaScript
	return fmt.Sprintf(`(() => {
		const elements = document.querySelectorAll('a, button, [role="button"], [onclick]');
		for (let el of elements) {
			if (el.textContent && el.textContent.includes('%s')) {
				el.click();
				return true;
			}
		}
		return false;
	})()`, searchText)
}

// DiscoverIncrementalActions analyzes new/changed HTML content for additional actions
func (ad *ActionDiscovery) DiscoverIncrementalActions(ctx context.Context, newContent string, existingActions []DiscoveredAction, existingTests []string) ([]DiscoveredAction, error) {
	if strings.TrimSpace(newContent) == "" {
		return nil, nil // No new content to analyze
	}

	// Create a simplified prompt focused on just the new content
	prompt := ad.buildIncrementalDiscoveryPrompt(newContent, existingActions)

	// Use AnalyzeCode method to get new actions
	analysis, err := ad.llmClient.AnalyzeCode(ctx, prompt, "incremental-discovery.html")
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed for incremental discovery: %w", err)
	}

	// Parse new actions from response
	newActions := ad.parseActionsFromResponse(analysis.Notes)

	// Check against existing tests
	for i := range newActions {
		newActions[i].IsTested = ad.isActionTested(newActions[i], existingTests)
	}

	// Filter out actions that are similar to existing ones
	filteredActions := ad.deduplicateActions(newActions, existingActions)

	return filteredActions, nil
}

// buildIncrementalDiscoveryPrompt creates a focused prompt for analyzing new content
func (ad *ActionDiscovery) buildIncrementalDiscoveryPrompt(newContent string, existingActions []DiscoveredAction) string {
	var prompt strings.Builder

	prompt.WriteString("You are analyzing NEW content that appeared on a web page after initial load.\n\n")

	prompt.WriteString("EXISTING ACTIONS ALREADY DISCOVERED:\n")
	for i, action := range existingActions {
		if i >= 10 {
			prompt.WriteString("... (and more)\n")
			break
		}
		prompt.WriteString(fmt.Sprintf("- %s\n", action.Description))
	}
	prompt.WriteString("\n")

	prompt.WriteString("NEW CONTENT THAT APPEARED:\n")
	if len(newContent) > 1500 {
		prompt.WriteString(newContent[:1500])
		prompt.WriteString("\n... (truncated)\n")
	} else {
		prompt.WriteString(newContent)
	}
	prompt.WriteString("\n\n")

	prompt.WriteString("TASK:\n")
	prompt.WriteString("Identify NEW user actions from this content that are NOT already covered.\n")
	prompt.WriteString("Focus on:\n")
	prompt.WriteString("1. New buttons that appeared (pricing options, CTAs)\n")
	prompt.WriteString("2. New navigation links that loaded\n")
	prompt.WriteString("3. New interactive elements (forms, dropdowns)\n")
	prompt.WriteString("4. Any dynamic content that enables new user actions\n\n")

	prompt.WriteString("AVOID:\n")
	prompt.WriteString("- Duplicating existing actions\n")
	prompt.WriteString("- Minor variations of existing actions\n")
	prompt.WriteString("- Actions on elements that were already discovered\n\n")

	prompt.WriteString("Format: ACTION_DESCRIPTION | priority\n")
	prompt.WriteString("Examples:\n")
	prompt.WriteString("Select Pro pricing plan | high\n")
	prompt.WriteString("View pricing details | medium\n")
	prompt.WriteString("Contact sales team | medium\n\n")

	prompt.WriteString("List only NEW actions (max 5):")

	return prompt.String()
}

// deduplicateActions removes actions that are too similar to existing ones
func (ad *ActionDiscovery) deduplicateActions(newActions []DiscoveredAction, existingActions []DiscoveredAction) []DiscoveredAction {
	var filtered []DiscoveredAction

	for _, newAction := range newActions {
		isDuplicate := false
		newDesc := strings.ToLower(strings.TrimSpace(newAction.Description))

		// Check against existing actions
		for _, existing := range existingActions {
			existingDesc := strings.ToLower(strings.TrimSpace(existing.Description))

			// Simple similarity check
			if ad.actionsAreSimilar(newDesc, existingDesc) {
				isDuplicate = true
				break
			}
		}

		// Also check against already filtered actions to avoid duplicates within new actions
		if !isDuplicate {
			for _, filtered := range filtered {
				filteredDesc := strings.ToLower(strings.TrimSpace(filtered.Description))
				if ad.actionsAreSimilar(newDesc, filteredDesc) {
					isDuplicate = true
					break
				}
			}
		}

		if !isDuplicate {
			filtered = append(filtered, newAction)
		}
	}

	return filtered
}

// actionsAreSimilar checks if two action descriptions are too similar
func (ad *ActionDiscovery) actionsAreSimilar(desc1, desc2 string) bool {
	// Exact match
	if desc1 == desc2 {
		return true
	}

	// Check if one contains the other
	if strings.Contains(desc1, desc2) || strings.Contains(desc2, desc1) {
		return true
	}

	// Check for similar key words
	words1 := strings.Fields(desc1)
	words2 := strings.Fields(desc2)

	if len(words1) < 2 || len(words2) < 2 {
		return false
	}

	// Count common meaningful words (longer than 3 characters)
	commonWords := 0
	for _, w1 := range words1 {
		if len(w1) <= 3 {
			continue
		}
		for _, w2 := range words2 {
			if len(w2) <= 3 {
				continue
			}
			if w1 == w2 {
				commonWords++
				break
			}
		}
	}

	// If most meaningful words overlap, consider it similar
	meaningfulWords := 0
	for _, w := range words1 {
		if len(w) > 3 {
			meaningfulWords++
		}
	}
	for _, w := range words2 {
		if len(w) > 3 {
			meaningfulWords++
		}
	}

	if meaningfulWords > 0 && float64(commonWords)/float64(meaningfulWords) >= 0.5 {
		return true
	}

	return false
}

// MergeActions merges new actions with existing ones, maintaining priority order
func (ad *ActionDiscovery) MergeActions(existing []DiscoveredAction, newActions []DiscoveredAction) []DiscoveredAction {
	// Deduplicate new actions against existing ones
	filtered := ad.deduplicateActions(newActions, existing)

	// Combine all actions
	combined := append(existing, filtered...)

	// Sort by priority: high -> medium -> low
	var high, medium, low []DiscoveredAction
	for _, action := range combined {
		switch strings.ToLower(action.Priority) {
		case "high":
			high = append(high, action)
		case "medium":
			medium = append(medium, action)
		default:
			low = append(low, action)
		}
	}

	// Combine in priority order
	result := append(high, medium...)
	result = append(result, low...)

	return result
}

// buildTestGenerationPrompt creates a prompt for generating test code
func (ad *ActionDiscovery) buildTestGenerationPrompt(actions []DiscoveredAction, framework string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Generate %s test code for the following untested user actions:\n\n", framework))

	for i, action := range actions {
		prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, action.Description))
		prompt.WriteString(fmt.Sprintf("   Selector: %s\n", action.Selector))
		prompt.WriteString(fmt.Sprintf("   Action: %s\n", action.Action))
		prompt.WriteString(fmt.Sprintf("   Scenario: %s\n", action.TestScenario))
		prompt.WriteString(fmt.Sprintf("   Priority: %s\n\n", action.Priority))
	}

	prompt.WriteString("Generate concise, well-structured test cases.\n")
	prompt.WriteString("Use proper assertions and follow testing best practices.\n")
	prompt.WriteString("Include both positive and negative test cases where appropriate.\n")

	return prompt.String()
}