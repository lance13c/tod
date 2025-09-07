package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/ciciliostudio/tod/internal/logging"
)

// ChromeDPManager manages a shared chromedp instance
type ChromeDPManager struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc
	baseURL     string
	isHeadless  bool
}

// findChrome attempts to find Chrome executable
func findChrome() (string, error) {
	// Try to find Chrome in common locations
	var paths []string
	
	switch runtime.GOOS {
	case "darwin": // macOS
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	case "linux":
		paths = []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Chromium\Application\chrome.exe`,
		}
	}
	
	// Check each path
	for _, path := range paths {
		// For macOS apps, check if file exists directly
		if runtime.GOOS == "darwin" {
			if _, err := os.Stat(path); err == nil {
				logging.Debug("Found Chrome at: %s", path)
				return path, nil
			}
		} else {
			// For Linux/Windows, use exec.LookPath
			if _, err := exec.LookPath(path); err == nil {
				logging.Debug("Found Chrome at: %s", path)
				return path, nil
			}
		}
	}
	
	// Try generic "chrome" command
	if path, err := exec.LookPath("chrome"); err == nil {
		return path, nil
	}
	
	return "", fmt.Errorf("Chrome browser not found. Please install Chrome, Chromium, or Brave")
}

// NewChromeDPManager creates a new ChromeDP manager
func NewChromeDPManager(baseURL string, headless bool) (*ChromeDPManager, error) {
	// First check if Chrome is installed
	chromePath, err := findChrome()
	if err != nil {
		return nil, err
	}
	logging.Info("Using Chrome from: %s", chromePath)
	// Start with default options but use our found Chrome path
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
	)
	
	// If not headless, disable it (default is headless)
	if !headless {
		logging.Info("Chrome will run in visible mode (headless=false)")
		opts = append(opts, 
			chromedp.Flag("headless", false),
		)
	} else {
		logging.Info("Chrome will run in headless mode (headless=true)")
	}
	
	// Add our custom options
	opts = append(opts,
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("remote-debugging-port", "9229"),
		chromedp.Flag("remote-debugging-address", "127.0.0.1"),
	)

	// Create allocator context with timeout
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Create browser context with debug logging
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(func(format string, v ...interface{}) {
			// Log chrome debug info
			logging.Debug("[Chrome] "+format, v...)
		}),
	)

	// Start Chrome - use the context directly, not a timeout context
	// The timeout would cancel the entire Chrome instance
	if err := chromedp.Run(ctx); err != nil {
		allocCancel()
		cancel()
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	manager := &ChromeDPManager{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		ctx:         ctx,
		cancel:      cancel,
		baseURL:     baseURL,
		isHeadless:  headless,
	}

	// Navigate to initial URL (optional - don't fail if site is down)
	if baseURL != "" {
		// Try to navigate but don't fail if the site isn't available
		logging.Info("Chrome started. Attempting initial navigation to %s...", baseURL)
		if err := manager.Navigate(baseURL); err != nil {
			logging.Debug("Initial navigation failed (this is OK): %v", err)
		} else {
			logging.Info("Successfully navigated to %s", baseURL)
		}
	}

	return manager, nil
}

// GetContext returns the chromedp context for running actions
func (m *ChromeDPManager) GetContext() context.Context {
	return m.ctx
}

// Navigate navigates to a URL
func (m *ChromeDPManager) Navigate(url string) error {
	// Navigate using the main context, not a timeout context
	// A timeout context would interfere with the browser's lifecycle
	err := chromedp.Run(m.ctx, chromedp.Navigate(url))
	if err != nil {
		// Check if context was cancelled
		if m.ctx.Err() != nil {
			return fmt.Errorf("Chrome context was cancelled")
		}
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	
	// Give the page a moment to start loading
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

// GetPageHTML gets the current page HTML
func (m *ChromeDPManager) GetPageHTML() (string, error) {
	var html string
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.OuterHTML(`html`, &html, chromedp.ByQuery),
	)
	return html, err
}

// GetPageInfo gets current page URL and title
func (m *ChromeDPManager) GetPageInfo() (url string, title string, err error) {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Location(&url),
		chromedp.Title(&title),
	)
	return url, title, err
}

// WaitForElement waits for an element to be visible
func (m *ChromeDPManager) WaitForElement(selector string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
	)
}

// Click clicks an element
func (m *ChromeDPManager) Click(selector string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.Click(selector, chromedp.ByQuery),
	)
}

// SmartClick attempts to click an element using multiple strategies and detects success
func (m *ChromeDPManager) SmartClick(selector string, text string) (bool, error) {
	// Get initial state for change detection
	initialURL, _, err := m.GetPageInfo()
	if err != nil {
		return false, fmt.Errorf("failed to get initial page info: %w", err)
	}

	// Strategy 1: Standard chromedp click
	logging.Debug("SmartClick: Trying standard click on selector: %s", selector)
	if err := m.Click(selector); err == nil {
		if changed := m.detectPageChange(initialURL, 500*time.Millisecond); changed {
			logging.Debug("SmartClick: Standard click successful for: %s", text)
			return true, nil
		}
	}

	// Strategy 2: JavaScript click
	logging.Debug("SmartClick: Trying JavaScript click on selector: %s", selector)
	jsScript := fmt.Sprintf(`
		const element = document.querySelector('%s');
		if (element) {
			element.click();
			true;
		} else {
			false;
		}
	`, selector)
	
	var jsResult bool
	if err := m.ExecuteScript(jsScript, &jsResult); err == nil && jsResult {
		if changed := m.detectPageChange(initialURL, 500*time.Millisecond); changed {
			logging.Debug("SmartClick: JavaScript click successful for: %s", text)
			return true, nil
		}
	}

	// Strategy 3: Dispatch click event
	logging.Debug("SmartClick: Trying event dispatch on selector: %s", selector)
	eventScript := fmt.Sprintf(`
		const element = document.querySelector('%s');
		if (element) {
			element.dispatchEvent(new MouseEvent('click', {
				view: window,
				bubbles: true,
				cancelable: true
			}));
			true;
		} else {
			false;
		}
	`, selector)
	
	if err := m.ExecuteScript(eventScript, &jsResult); err == nil && jsResult {
		if changed := m.detectPageChange(initialURL, 500*time.Millisecond); changed {
			logging.Debug("SmartClick: Event dispatch successful for: %s", text)
			return true, nil
		}
	}

	// Strategy 4: Focus and Enter key (for button-like elements)
	logging.Debug("SmartClick: Trying focus+enter on selector: %s", selector)
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Focus(selector, chromedp.ByQuery),
		chromedp.KeyEvent("`Enter`"),
	)
	
	if err == nil {
		if changed := m.detectPageChange(initialURL, 500*time.Millisecond); changed {
			logging.Debug("SmartClick: Focus+enter successful for: %s", text)
			return true, nil
		}
	}

	// Strategy 5: Try finding by text content if selector failed
	if text != "" {
		logging.Debug("SmartClick: Trying text-based click for: %s", text)
		textScript := fmt.Sprintf(`
			const elements = document.querySelectorAll('a, button, [role="button"]');
			for (let el of elements) {
				if (el.textContent && el.textContent.trim().toLowerCase().includes('%s')) {
					el.click();
					return true;
				}
			}
			return false;
		`, strings.ToLower(text))
		
		if err := m.ExecuteScript(textScript, &jsResult); err == nil && jsResult {
			if changed := m.detectPageChange(initialURL, 500*time.Millisecond); changed {
				logging.Debug("SmartClick: Text-based click successful for: %s", text)
				return true, nil
			}
		}
	}

	logging.Warn("SmartClick: All strategies failed for: %s (selector: %s)", text, selector)
	return false, nil
}

// detectPageChange checks if the page changed after an action
func (m *ChromeDPManager) detectPageChange(initialURL string, waitTime time.Duration) bool {
	time.Sleep(waitTime)
	
	currentURL, _, err := m.GetPageInfo()
	if err != nil {
		return false
	}
	
	// URL change is the most reliable indicator
	if currentURL != initialURL {
		return true
	}
	
	// TODO: Could also check for DOM changes, loading states, etc.
	return false
}

// SendKeys sends keys to an element
func (m *ChromeDPManager) SendKeys(selector string, text string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

// FillFormField fills a form field with enhanced error handling and validation
func (m *ChromeDPManager) FillFormField(selector, value string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		// First wait for element to be visible
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		// Focus the element
		chromedp.Focus(selector, chromedp.ByQuery),
		// Clear any existing value
		chromedp.Clear(selector, chromedp.ByQuery),
		// Send the new value
		chromedp.SendKeys(selector, value, chromedp.ByQuery),
		// Trigger events to ensure the form recognizes the input
		chromedp.Evaluate(fmt.Sprintf(`
			const element = document.querySelector('%s');
			if (element) {
				element.dispatchEvent(new Event('input', { bubbles: true }));
				element.dispatchEvent(new Event('change', { bubbles: true }));
			}
		`, selector), nil),
	)
}

// GetFormElements extracts form elements with enhanced detection
func (m *ChromeDPManager) GetFormElements() ([]FormElementInfo, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	script := `
		(() => {
			const forms = [];
			
			// Find all forms on the page
			document.querySelectorAll('form').forEach(form => {
				const formInfo = {
					selector: '',
					action: form.action || '',
					method: form.method || 'GET',
					fields: []
				};
				
				// Generate selector for form
				if (form.id) {
					formInfo.selector = '#' + form.id;
				} else if (form.className) {
					const classes = form.className.split(' ').filter(c => c && !c.includes('css-'));
					if (classes.length > 0) {
						formInfo.selector = '.' + classes[0];
					}
				} else {
					formInfo.selector = 'form';
				}
				
				// Find form fields
				const inputs = form.querySelectorAll('input, textarea, select');
				inputs.forEach(input => {
					if (input.type === 'hidden') return;
					if (input.style.display === 'none') return;
					if (input.offsetParent === null) return;
					
					const field = {
						type: input.type || input.tagName.toLowerCase(),
						name: input.name || '',
						placeholder: input.placeholder || '',
						required: input.required || false,
						value: input.value || '',
						selector: '',
						label: ''
					};
					
					// Generate selector
					if (input.id) {
						field.selector = '#' + input.id;
					} else if (input.name) {
						field.selector = 'input[name="' + input.name + '"]';
					} else if (input.className) {
						const classes = input.className.split(' ').filter(c => c && !c.includes('css-'));
						if (classes.length > 0) {
							field.selector = '.' + classes[0];
						}
					}
					
					// Find label
					const labels = form.querySelectorAll('label');
					for (let label of labels) {
						if (label.getAttribute('for') === input.id || 
							label.contains(input)) {
							field.label = label.textContent.trim();
							break;
						}
					}
					
					formInfo.fields.push(field);
				});
				
				// Find submit buttons
				const submitButtons = form.querySelectorAll('button[type="submit"], input[type="submit"], button:not([type])');
				submitButtons.forEach(button => {
					if (button.style.display === 'none') return;
					if (button.offsetParent === null) return;
					
					const field = {
						type: 'submit',
						name: button.name || '',
						value: button.value || button.textContent || 'Submit',
						selector: '',
						label: button.textContent || button.value || 'Submit'
					};
					
					if (button.id) {
						field.selector = '#' + button.id;
					} else if (button.name) {
						field.selector = 'button[name="' + button.name + '"]';
					} else {
						field.selector = 'button[type="submit"]';
					}
					
					formInfo.fields.push(field);
				});
				
				forms.push(formInfo);
			});
			
			return forms;
		})()
	`

	var jsElements []map[string]interface{}
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &jsElements)); err != nil {
		return nil, err
	}

	var forms []FormElementInfo
	for _, jsForm := range jsElements {
		form := FormElementInfo{
			Selector: getStringValue(jsForm["selector"]),
			Action:   getStringValue(jsForm["action"]),
			Method:   getStringValue(jsForm["method"]),
		}

		// Parse fields
		if fieldsData, ok := jsForm["fields"].([]interface{}); ok {
			for _, fieldData := range fieldsData {
				if fieldMap, ok := fieldData.(map[string]interface{}); ok {
					field := FormFieldInfo{
						Type:        getStringValue(fieldMap["type"]),
						Name:        getStringValue(fieldMap["name"]),
						Placeholder: getStringValue(fieldMap["placeholder"]),
						Value:       getStringValue(fieldMap["value"]),
						Selector:    getStringValue(fieldMap["selector"]),
						Label:       getStringValue(fieldMap["label"]),
						Required:    getBoolValue(fieldMap["required"]),
					}
					form.Fields = append(form.Fields, field)
				}
			}
		}

		forms = append(forms, form)
	}

	return forms, nil
}

// CheckForPageChanges monitors for page changes after an action
func (m *ChromeDPManager) CheckForPageChanges(initialURL string, timeout time.Duration) (*PageChangeInfo, error) {
	changes := m.PollForChanges(timeout, 200*time.Millisecond, 50*time.Millisecond)
	
	info := &PageChangeInfo{
		InitialURL: initialURL,
		Changed:    false,
	}

	for change := range changes {
		// Get current URL
		currentURL, title, err := m.GetPageInfo()
		if err == nil {
			info.FinalURL = currentURL
			info.FinalTitle = title
			
			if currentURL != initialURL {
				info.Changed = true
				info.URLChanged = true
				break
			}
		}

		// Check for specific content changes indicating success/failure
		htmlLower := strings.ToLower(change.HTML)
		
		// Check for magic link messages
		magicLinkPhrases := []string{
			"magic link sent", "check your email", "email sent", 
			"sign in link sent", "we sent you", "check your inbox",
		}
		for _, phrase := range magicLinkPhrases {
			if strings.Contains(htmlLower, phrase) {
				info.Changed = true
				info.ContentChanged = true
				info.MagicLinkDetected = true
				info.Message = "Magic link sent"
				return info, nil
			}
		}

		// Check for error messages
		errorPhrases := []string{
			"error", "invalid", "incorrect", "failed", "wrong",
			"unauthorized", "access denied", "login failed",
		}
		for _, phrase := range errorPhrases {
			if strings.Contains(htmlLower, phrase) {
				info.Changed = true
				info.ContentChanged = true
				info.ErrorDetected = true
				info.Message = "Error detected"
				return info, nil
			}
		}

		// Check for success indicators
		successPhrases := []string{
			"welcome", "dashboard", "logged in", "signed in",
			"authentication successful", "login successful",
		}
		for _, phrase := range successPhrases {
			if strings.Contains(htmlLower, phrase) {
				info.Changed = true
				info.ContentChanged = true
				info.SuccessDetected = true
				info.Message = "Login successful"
				return info, nil
			}
		}
	}

	return info, nil
}

// FormElementInfo represents information about a form
type FormElementInfo struct {
	Selector string
	Action   string
	Method   string
	Fields   []FormFieldInfo
}

// FormFieldInfo represents information about a form field
type FormFieldInfo struct {
	Type        string
	Name        string
	Placeholder string
	Value       string
	Selector    string
	Label       string
	Required    bool
}

// PageChangeInfo represents information about page changes
type PageChangeInfo struct {
	InitialURL        string
	FinalURL          string
	FinalTitle        string
	Changed           bool
	URLChanged        bool
	ContentChanged    bool
	MagicLinkDetected bool
	ErrorDetected     bool
	SuccessDetected   bool
	Message           string
}

// Screenshot takes a screenshot
func (m *ChromeDPManager) Screenshot() ([]byte, error) {
	var buf []byte
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buf, 90),
	)
	return buf, err
}

// ExecuteScript executes JavaScript
func (m *ChromeDPManager) ExecuteScript(script string, result interface{}) error {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.Evaluate(script, result),
	)
}

// WaitForPageLoad waits for the page to be fully loaded and interactive
func (m *ChromeDPManager) WaitForPageLoad(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(m.ctx, timeout)
	defer cancel()

	// Wait for the document to be ready
	script := `
		(() => {
			if (document.readyState === 'complete') return true;
			if (document.readyState === 'interactive') return true;
			return false;
		})()
	`

	// Poll until page is ready
	for {
		var ready bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(script, &ready)); err != nil {
			return fmt.Errorf("failed to check page readiness: %w", err)
		}

		if ready {
			// Add a small additional delay to ensure elements are rendered
			time.Sleep(300 * time.Millisecond)
			return nil
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for page to load")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ExtractInteractiveElements extracts interactive elements from the page
func (m *ChromeDPManager) ExtractInteractiveElements() ([]InteractiveElement, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	var elements []InteractiveElement

	// JavaScript to extract interactive elements
	script := `
		(() => {
			const elements = [];
			const selectors = [
				'button',
				'a[href]',
				'input:not([type="hidden"])',
				'select',
				'textarea',
				'[role="button"]',
				'[onclick]',
				'[data-testid]',
				'nav a',
				'.nav a',
				'.navigation a'
			];
			
			// Helper to generate better selectors
			function generateSelector(el) {
				if (el.id) return '#' + el.id;
				if (el.dataset.testid) return '[data-testid="' + el.dataset.testid + '"]';
				if (el.className) {
					const classes = el.className.split(' ').filter(c => c && !c.includes('css-'));
					if (classes.length > 0) return '.' + classes[0];
				}
				
				// For links, try to create a contains selector based on text
				if (el.tagName.toLowerCase() === 'a' && el.textContent.trim()) {
					const text = el.textContent.trim().replace(/'/g, "\\'");
					return "a:contains('" + text + "')";
				}
				
				// For buttons, same approach
				if (el.tagName.toLowerCase() === 'button' && el.textContent.trim()) {
					const text = el.textContent.trim().replace(/'/g, "\\'");
					return "button:contains('" + text + "')";
				}
				
				return el.tagName.toLowerCase();
			}
			
			// Helper to get full URL
			function getFullUrl(href) {
				if (!href) return '';
				if (href.startsWith('http')) return href;
				if (href.startsWith('/')) return window.location.origin + href;
				return window.location.href + (window.location.href.endsWith('/') ? '' : '/') + href;
			}
			
			selectors.forEach(selector => {
				document.querySelectorAll(selector).forEach(el => {
					// Check if element is visible
					if (el.offsetParent !== null || el.tagName.toLowerCase() === 'a') {
						const text = el.textContent?.trim() || el.value || el.placeholder || el.alt || '';
						const href = el.href || '';
						
						// Skip if no meaningful text and no href
						if (!text && !href) return;
						
						// Skip very long text that's likely not a navigation element
						if (text.length > 100) return;
						
						elements.push({
							tag: el.tagName.toLowerCase(),
							text: text,
							selector: generateSelector(el),
							type: el.type || '',
							href: href,
							fullUrl: getFullUrl(href),
							ariaLabel: el.getAttribute('aria-label') || '',
							title: el.getAttribute('title') || '',
							isNavigation: el.tagName.toLowerCase() === 'a' && href.length > 0,
							isButton: el.tagName.toLowerCase() === 'button' || el.getAttribute('role') === 'button'
						});
					}
				});
			});
			
			// Sort by priority: navigation links first, then buttons, then other elements
			elements.sort((a, b) => {
				if (a.isNavigation && !b.isNavigation) return -1;
				if (!a.isNavigation && b.isNavigation) return 1;
				if (a.isButton && !b.isButton) return -1;
				if (!a.isButton && b.isButton) return 1;
				return 0;
			});
			
			return elements;
		})()
	`

	var jsElements []map[string]interface{}
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &jsElements)); err != nil {
		return nil, err
	}

	// Convert to InteractiveElement
	for _, jsEl := range jsElements {
		element := InteractiveElement{
			Tag:          getStringValue(jsEl["tag"]),
			Text:         getStringValue(jsEl["text"]),
			Selector:     getStringValue(jsEl["selector"]),
			Type:         getStringValue(jsEl["type"]),
			Href:         getStringValue(jsEl["href"]),
			FullUrl:      getStringValue(jsEl["fullUrl"]),
			AriaLabel:    getStringValue(jsEl["ariaLabel"]),
			Title:        getStringValue(jsEl["title"]),
			IsNavigation: getBoolValue(jsEl["isNavigation"]),
			IsButton:     getBoolValue(jsEl["isButton"]),
		}
		
		elements = append(elements, element)
	}

	return elements, nil
}

// Close closes the browser and cleans up resources
func (m *ChromeDPManager) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.allocCancel != nil {
		m.allocCancel()
	}
}

// Helper function to safely get string value from interface{}
func getStringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Helper function to safely get bool value from interface{}
func getBoolValue(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// Global manager instance for sharing between views
var globalChromeDPManager *ChromeDPManager

// GetGlobalChromeDPManager gets or creates the global ChromeDP manager
func GetGlobalChromeDPManager(baseURL string, headless bool) (*ChromeDPManager, error) {
	// If we already have a manager, return it
	if globalChromeDPManager != nil {
		// Check if context is still valid
		select {
		case <-globalChromeDPManager.ctx.Done():
			// Context is done, need to create a new one
			logging.Debug("Previous Chrome instance was closed, creating new one")
			globalChromeDPManager = nil
		default:
			// Context is still valid
			return globalChromeDPManager, nil
		}
	}

	logging.Info("Creating new Chrome instance...")
	manager, err := NewChromeDPManager(baseURL, headless)
	if err != nil {
		return nil, err
	}

	globalChromeDPManager = manager
	return manager, nil
}

// CloseGlobalChromeDPManager closes the global manager
func CloseGlobalChromeDPManager() {
	if globalChromeDPManager != nil {
		globalChromeDPManager.Close()
		globalChromeDPManager = nil
	}
}

// HTMLChange represents an HTML change detected during polling
type HTMLChange struct {
	HTML      string
	Timestamp int64
	IsInitial bool
	NewContent string // The content that was added/changed
}

// PollForChanges monitors the page for HTML changes over a period of time
func (m *ChromeDPManager) PollForChanges(duration time.Duration, interval time.Duration, initialDelay time.Duration) <-chan HTMLChange {
	changes := make(chan HTMLChange, 10) // Buffered channel
	
	go func() {
		defer close(changes)
		
		differ := NewHTMLDiffer()
		
		// Initial delay (20-40ms) to let DOM stabilize
		time.Sleep(initialDelay)
		
		// Get initial snapshot
		initialHTML, err := m.GetPageHTML()
		if err != nil {
			logging.Debug("Failed to get initial HTML: %v", err)
			return
		}
		
		timestamp := time.Now().UnixMilli()
		changed, snapshot := differ.HasChanged(initialHTML, timestamp)
		if changed {
			changes <- HTMLChange{
				HTML:      snapshot.HTML,
				Timestamp: snapshot.Timestamp,
				IsInitial: true,
				NewContent: "",
			}
		}
		
		// Start polling
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		timeout := time.After(duration)
		
		for {
			select {
			case <-timeout:
				logging.Debug("Polling completed after %v", duration)
				return
				
			case <-ticker.C:
				html, err := m.GetPageHTML()
				if err != nil {
					logging.Debug("Failed to get HTML during polling: %v", err)
					continue
				}
				
				timestamp := time.Now().UnixMilli()
				changed, snapshot := differ.HasChanged(html, timestamp)
				
				if changed {
					// Extract what changed
					newContent := ""
					if differ.lastSnapshot != nil {
						newContent = differ.GetChangedSections(initialHTML, html)
					}
					
					changes <- HTMLChange{
						HTML:      snapshot.HTML,
						Timestamp: snapshot.Timestamp,
						IsInitial: false,
						NewContent: newContent,
					}
					
					logging.Debug("HTML change detected at %d, new content length: %d", timestamp, len(newContent))
				}
				
			case <-m.ctx.Done():
				logging.Debug("Chrome context cancelled, stopping polling")
				return
			}
		}
	}()
	
	return changes
}