package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// PageAction represents a discoverable action on a web page
type PageAction struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`        // "link", "button", "input", "select", "clickable"
	Category    string            `json:"category"`    // "Navigation", "Form", "Authentication", "Interactive"
	Description string            `json:"description"` // User-friendly description
	Text        string            `json:"text"`        // Visible text
	Selector    string            `json:"selector"`    // CSS selector to target element
	Attributes  map[string]string `json:"attributes"`  // Element attributes (href, value, etc.)
	Position    Position          `json:"position"`    // Element position on page
	Icon        string            `json:"icon"`        // Unicode icon for display
	Priority    int               `json:"priority"`    // Higher = more important
}

// Position represents the location of an element on the page
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PageAnalyzer handles discovering actions from web pages
type PageAnalyzer struct {
	ctx context.Context
}

// NewPageAnalyzer creates a new page analyzer
func NewPageAnalyzer(ctx context.Context) *PageAnalyzer {
	return &PageAnalyzer{
		ctx: ctx,
	}
}

// DiscoverActions discovers all interactive actions on the current page
func (pa *PageAnalyzer) DiscoverActions() ([]PageAction, error) {
	// JavaScript code to discover all interactive elements
	discoverScript := `
		(function() {
			const actions = [];
			let actionId = 1;

			// Helper function to get element text
			function getElementText(el) {
				// Try various text sources in order of preference
				return el.textContent?.trim() || 
					   el.value || 
					   el.placeholder || 
					   el.alt || 
					   el.title || 
					   el.getAttribute('aria-label') || 
					   el.getAttribute('data-testid') || 
					   '';
			}

			// Helper function to generate CSS selector
			function generateSelector(el) {
				if (el.id) return '#' + el.id;
				if (el.className) {
					const classes = el.className.split(' ').filter(c => c.trim());
					if (classes.length > 0) return '.' + classes[0];
				}
				return el.tagName.toLowerCase();
			}

			// Helper function to get element position
			function getPosition(el) {
				const rect = el.getBoundingClientRect();
				return {
					x: rect.left,
					y: rect.top,
					width: rect.width,
					height: rect.height
				};
			}

			// Helper function to categorize elements
			function categorizeElement(el, type) {
				const text = getElementText(el).toLowerCase();
				const attrs = Array.from(el.attributes).reduce((acc, attr) => {
					acc[attr.name] = attr.value;
					return acc;
				}, {});

				// Authentication patterns
				if (text.includes('sign in') || text.includes('login') || 
					text.includes('sign up') || text.includes('register') ||
					text.includes('logout') || text.includes('sign out') ||
					el.type === 'password' || attrs.name?.includes('password') ||
					attrs.name?.includes('email') || el.type === 'email') {
					return 'Authentication';
				}

				// Navigation patterns
				if (type === 'link' || text.includes('home') || text.includes('dashboard') ||
					text.includes('menu') || text.includes('nav') || text.includes('back')) {
					return 'Navigation';
				}

				// Form patterns
				if (type === 'input' || type === 'select' || type === 'textarea' ||
					text.includes('submit') || text.includes('save') || text.includes('send')) {
					return 'Form';
				}

				return 'Interactive';
			}

			// Helper function to generate user-friendly description
			function generateDescription(el, type, category) {
				const text = getElementText(el);
				const tag = el.tagName.toLowerCase();
				
				switch (type) {
					case 'link':
						const href = el.href;
						if (href && href !== window.location.href) {
							const path = new URL(href).pathname;
							return text ? 'Navigate to "' + text + '"' : 'Go to ' + path;
						}
						return text ? 'Click "' + text + '"' : 'Click link';
					
					case 'button':
						if (category === 'Authentication') {
							if (text.toLowerCase().includes('sign in') || text.toLowerCase().includes('login')) {
								return 'Sign in to your account';
							}
							if (text.toLowerCase().includes('sign up') || text.toLowerCase().includes('register')) {
								return 'Create new account';
							}
						}
						return text ? 'Click "' + text + '" button' : 'Click button';
					
					case 'input':
						const inputType = el.type || 'text';
						const placeholder = el.placeholder || '';
						const label = el.getAttribute('aria-label') || '';
						
						if (inputType === 'email') return 'Fill email address';
						if (inputType === 'password') return 'Enter password';
						if (inputType === 'search') return 'Enter search query';
						if (inputType === 'checkbox') return 'Toggle ' + (text || label || 'checkbox');
						if (inputType === 'radio') return 'Select ' + (text || label || 'option');
						
						if (placeholder) return 'Fill "' + placeholder + '"';
						if (label) return 'Fill "' + label + '"';
						return 'Fill ' + inputType + ' field';
					
					case 'select':
						return 'Select from "' + (text || 'dropdown') + '"';
					
					default:
						return text ? 'Interact with "' + text + '"' : 'Click element';
				}
			}

			// Helper function to get priority based on element importance
			function getPriority(el, type, category) {
				const text = getElementText(el).toLowerCase();
				let priority = 50; // Base priority

				// High priority for authentication
				if (category === 'Authentication') priority += 30;
				
				// High priority for primary buttons
				if (el.classList.contains('primary') || el.classList.contains('btn-primary')) priority += 20;
				
				// Common important actions
				if (text.includes('sign in') || text.includes('login')) priority += 25;
				if (text.includes('submit') || text.includes('save') || text.includes('send')) priority += 15;
				if (text.includes('search')) priority += 10;
				if (text.includes('home') || text.includes('dashboard')) priority += 10;
				
				// Form fields get medium priority
				if (type === 'input') priority += 5;
				
				// Links get base priority
				if (type === 'link') priority += 0;
				
				return priority;
			}

			// Helper function to get icon based on category and type
			function getIcon(category, type, el) {
				// Return empty string - no emojis
				return '';
			}

			// Discover links
			document.querySelectorAll('a[href]').forEach(link => {
				if (link.offsetParent !== null) { // Only visible elements
					const text = getElementText(link);
					const category = categorizeElement(link, 'link');
					
					actions.push({
						id: 'link_' + actionId++,
						type: 'link',
						category: category,
						description: generateDescription(link, 'link', category),
						text: text,
						selector: generateSelector(link),
						attributes: {
							href: link.href,
							target: link.target || '_self'
						},
						position: getPosition(link),
						icon: getIcon(category, 'link', link),
						priority: getPriority(link, 'link', category)
					});
				}
			});

			// Discover buttons
			document.querySelectorAll('button, input[type="button"], input[type="submit"]').forEach(button => {
				if (button.offsetParent !== null) {
					const text = getElementText(button);
					const category = categorizeElement(button, 'button');
					
					actions.push({
						id: 'button_' + actionId++,
						type: 'button',
						category: category,
						description: generateDescription(button, 'button', category),
						text: text,
						selector: generateSelector(button),
						attributes: {
							type: button.type,
							disabled: button.disabled.toString()
						},
						position: getPosition(button),
						icon: getIcon(category, 'button', button),
						priority: getPriority(button, 'button', category)
					});
				}
			});

			// Discover input fields
			document.querySelectorAll('input, textarea').forEach(input => {
				if (input.offsetParent !== null && input.type !== 'hidden') {
					const text = getElementText(input);
					const category = categorizeElement(input, 'input');
					
					actions.push({
						id: 'input_' + actionId++,
						type: 'input',
						category: category,
						description: generateDescription(input, 'input', category),
						text: text,
						selector: generateSelector(input),
						attributes: {
							type: input.type,
							name: input.name || '',
							placeholder: input.placeholder || '',
							required: input.required.toString()
						},
						position: getPosition(input),
						icon: getIcon(category, 'input', input),
						priority: getPriority(input, 'input', category)
					});
				}
			});

			// Discover select elements
			document.querySelectorAll('select').forEach(select => {
				if (select.offsetParent !== null) {
					const text = getElementText(select);
					const category = categorizeElement(select, 'select');
					
					actions.push({
						id: 'select_' + actionId++,
						type: 'select',
						category: category,
						description: generateDescription(select, 'select', category),
						text: text,
						selector: generateSelector(select),
						attributes: {
							name: select.name || '',
							multiple: select.multiple.toString()
						},
						position: getPosition(select),
						icon: getIcon(category, 'select', select),
						priority: getPriority(select, 'select', category)
					});
				}
			});

			// Discover clickable elements (with event listeners)
			document.querySelectorAll('[onclick], [data-testid]').forEach(el => {
				if (el.offsetParent !== null && !['a', 'button', 'input', 'select'].includes(el.tagName.toLowerCase())) {
					const text = getElementText(el);
					const category = categorizeElement(el, 'clickable');
					
					actions.push({
						id: 'clickable_' + actionId++,
						type: 'clickable',
						category: category,
						description: generateDescription(el, 'clickable', category),
						text: text,
						selector: generateSelector(el),
						attributes: {
							tag: el.tagName.toLowerCase(),
							testid: el.getAttribute('data-testid') || ''
						},
						position: getPosition(el),
						icon: getIcon(category, 'clickable', el),
						priority: getPriority(el, 'clickable', category)
					});
				}
			});

			// Sort by priority (highest first)
			actions.sort((a, b) => b.priority - a.priority);

			return JSON.stringify(actions);
		})();
	`

	var result string
	err := chromedp.Run(pa.ctx,
		chromedp.Evaluate(discoverScript, &result),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to discover page actions: %w", err)
	}

	// Handle empty or invalid results
	if result == "" || result == "undefined" || result == "null" {
		return []PageAction{}, nil
	}

	var actions []PageAction
	if err := json.Unmarshal([]byte(result), &actions); err != nil {
		return nil, fmt.Errorf("failed to parse discovered actions - result was: %s, error: %w", result, err)
	}

	// Post-process actions to clean up and enhance descriptions
	actions = pa.enhanceActions(actions)

	return actions, nil
}

// enhanceActions post-processes discovered actions to improve descriptions and filtering
func (pa *PageAnalyzer) enhanceActions(actions []PageAction) []PageAction {
	enhanced := make([]PageAction, 0, len(actions))

	for _, action := range actions {
		// Skip actions with empty descriptions or very short text
		if action.Description == "" || len(action.Description) < 3 {
			continue
		}

		// Clean up descriptions
		action.Description = strings.TrimSpace(action.Description)
		if len(action.Description) > 60 {
			action.Description = action.Description[:57] + "..."
		}

		// Ensure we have valid selectors
		if action.Selector == "" {
			continue
		}

		enhanced = append(enhanced, action)
	}

	return enhanced
}

// GetPageTitle returns the current page title
func (pa *PageAnalyzer) GetPageTitle() (string, error) {
	var title string
	err := chromedp.Run(pa.ctx,
		chromedp.Title(&title),
	)
	return title, err
}

// GetPageURL returns the current page URL
func (pa *PageAnalyzer) GetPageURL() (string, error) {
	var url string
	err := chromedp.Run(pa.ctx,
		chromedp.Location(&url),
	)
	return url, err
}

// WaitForPageLoad waits for the page to be fully loaded
func (pa *PageAnalyzer) WaitForPageLoad() error {
	return chromedp.Run(pa.ctx,
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Small delay to ensure JS has run
	)
}