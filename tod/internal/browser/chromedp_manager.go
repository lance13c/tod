package browser

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"
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
				log.Printf("Found Chrome at: %s", path)
				return path, nil
			}
		} else {
			// For Linux/Windows, use exec.LookPath
			if _, err := exec.LookPath(path); err == nil {
				log.Printf("Found Chrome at: %s", path)
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
	log.Printf("Using Chrome from: %s", chromePath)
	// Start with default options but use our found Chrome path
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),
	)
	
	// If not headless, disable it (default is headless)
	if !headless {
		opts = append(opts, 
			chromedp.Flag("headless", false),
		)
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
			log.Printf("[Chrome] "+format, v...)
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
		log.Printf("Chrome started. Attempting initial navigation to %s...", baseURL)
		if err := manager.Navigate(baseURL); err != nil {
			log.Printf("Initial navigation failed (this is OK): %v", err)
		} else {
			log.Printf("Successfully navigated to %s", baseURL)
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

// SendKeys sends keys to an element
func (m *ChromeDPManager) SendKeys(selector string, text string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
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
			log.Printf("Previous Chrome instance was closed, creating new one")
			globalChromeDPManager = nil
		default:
			// Context is still valid
			return globalChromeDPManager, nil
		}
	}

	log.Printf("Creating new Chrome instance...")
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
			log.Printf("Failed to get initial HTML: %v", err)
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
				log.Printf("Polling completed after %v", duration)
				return
				
			case <-ticker.C:
				html, err := m.GetPageHTML()
				if err != nil {
					log.Printf("Failed to get HTML during polling: %v", err)
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
					
					log.Printf("HTML change detected at %d, new content length: %d", timestamp, len(newContent))
				}
				
			case <-m.ctx.Done():
				log.Printf("Chrome context cancelled, stopping polling")
				return
			}
		}
	}()
	
	return changes
}