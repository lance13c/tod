package browser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Client represents a headless browser automation client
type Client struct {
	ctx         context.Context
	allocCancel context.CancelFunc
	ctxCancel   context.CancelFunc
	
	// Current state
	currentURL string
	baseURL    string
	
	// Page analysis
	pageAnalyzer *PageAnalyzer
}

// PageInfo contains information about the current page
type PageInfo struct {
	URL         string
	Title       string
	Description string
}

// Permission errors
var (
	ErrScreenRecordingPermission = errors.New("Chrome requires Screen Recording permission")
	ErrAccessibilityPermission   = errors.New("Chrome requires Accessibility permission")
	ErrChromeNotFound           = errors.New("Chrome browser not found")
)

// NewClient creates a new headless browser client
func NewClient(baseURL string) (*Client, error) {
	// Check permissions before creating client
	if err := CheckPermissions(); err != nil {
		return nil, err
	}

	// Create chrome instance with enhanced flags
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.Flag("enable-logging", true),
		chromedp.Flag("disable-web-security", true),
	)
	
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	
	// Create a context
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	
	return &Client{
		ctx:          ctx,
		allocCancel:  allocCancel,
		ctxCancel:    ctxCancel,
		baseURL:      baseURL,
		pageAnalyzer: NewPageAnalyzer(ctx),
	}, nil
}

// NavigateToURL navigates to a specific URL
func (c *Client) NavigateToURL(urlPath string) (*PageInfo, error) {
	// Resolve the URL
	targetURL, err := c.resolveURL(urlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve URL: %w", err)
	}
	
	// Create timeout context for this operation (increased to 45 seconds for slow pages)
	ctx, cancel := context.WithTimeout(c.ctx, 45*time.Second)
	defer cancel()
	
	var title string
	var currentURL string
	
	err = chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Location(&currentURL),
		chromedp.Title(&title),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", targetURL, err)
	}
	
	c.currentURL = currentURL
	
	return &PageInfo{
		URL:         currentURL,
		Title:       title,
		Description: fmt.Sprintf("Successfully navigated to %s", title),
	}, nil
}

// GetCurrentPage returns information about the current page
func (c *Client) GetCurrentPage() (*PageInfo, error) {
	if c.currentURL == "" {
		return &PageInfo{
			URL:         "Not connected",
			Title:       "Tod Adventure",
			Description: "Ready to begin your testing journey",
		}, nil
	}
	
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	
	var title string
	var currentURL string
	
	err := chromedp.Run(ctx,
		chromedp.Location(&currentURL),
		chromedp.Title(&title),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get current page info: %w", err)
	}
	
	return &PageInfo{
		URL:         currentURL,
		Title:       title,
		Description: fmt.Sprintf("Currently viewing %s", title),
	}, nil
}

// ClickElement clicks on an element by selector
func (c *Client) ClickElement(selector string) error {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()
	
	// Try multiple selector strategies
	selectors := strings.Split(selector, ", ")
	
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		
		// Skip selectors with :contains() as ChromeDP doesn't support them
		if strings.Contains(sel, ":contains(") {
			continue
		}
		
		err := chromedp.Run(ctx,
			chromedp.WaitVisible(sel, chromedp.ByQuery),
			chromedp.Click(sel, chromedp.ByQuery),
		)
		
		if err == nil {
			return nil // Success
		}
	}
	
	return fmt.Errorf("could not find element with selector: %s", selector)
}

// FillField fills a form field with a value
func (c *Client) FillField(selector, value string) error {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()
	
	// Try multiple selector strategies
	selectors := strings.Split(selector, ", ")
	
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		
		// Skip selectors with :contains() as ChromeDP doesn't support them
		if strings.Contains(sel, ":contains(") {
			continue
		}
		
		err := chromedp.Run(ctx,
			chromedp.WaitVisible(sel, chromedp.ByQuery),
			chromedp.Clear(sel, chromedp.ByQuery),
			chromedp.SendKeys(sel, value, chromedp.ByQuery),
		)
		
		if err == nil {
			return nil // Success
		}
	}
	
	return fmt.Errorf("could not find field with selector: %s", selector)
}

// CaptureScreenshot captures a screenshot of the current viewport
func (c *Client) CaptureScreenshot() ([]byte, error) {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}
	return buf, nil
}

// CaptureFullPageScreenshot captures a screenshot of the entire page
func (c *Client) CaptureFullPageScreenshot() ([]byte, error) {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()
	
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to capture full page screenshot: %w", err)
	}
	return buf, nil
}

// CaptureElementScreenshot captures a screenshot of a specific element
func (c *Client) CaptureElementScreenshot(selector string) ([]byte, error) {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	
	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to capture element screenshot: %w", err)
	}
	return buf, nil
}

// GetPageActions discovers all interactive actions on the current page
func (c *Client) GetPageActions() ([]PageAction, error) {
	// Create timeout context for this operation
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	
	// Update page analyzer with new context
	analyzer := NewPageAnalyzer(ctx)
	
	// Wait for page to be fully loaded
	if err := analyzer.WaitForPageLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}
	
	// Discover actions
	actions, err := analyzer.DiscoverActions()
	if err != nil {
		return nil, fmt.Errorf("failed to discover page actions: %w", err)
	}
	
	return actions, nil
}

// RefreshPageActions forces a refresh of the page actions
func (c *Client) RefreshPageActions() ([]PageAction, error) {
	return c.GetPageActions()
}

// Close closes the browser client and releases resources
func (c *Client) Close() error {
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	if c.allocCancel != nil {
		c.allocCancel()
	}
	return nil
}

// resolveURL resolves relative URLs to absolute URLs
func (c *Client) resolveURL(urlPath string) (string, error) {
	if urlPath == "" {
		return c.baseURL, nil
	}
	
	// If it's already a complete URL, use it as is
	if u, err := url.Parse(urlPath); err == nil && u.Scheme != "" {
		return urlPath, nil
	}
	
	// Resolve relative to base URL
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	
	relative, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}
	
	resolved := base.ResolveReference(relative)
	return resolved.String(), nil
}

// CheckPermissions checks if Chrome has the necessary permissions
func CheckPermissions() error {
	// Only check permissions on macOS
	if runtime.GOOS != "darwin" {
		return nil
	}

	// Create a temporary context for permission testing
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("use-mock-keychain", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a short timeout for permission check
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to navigate to a simple page
	err := chromedp.Run(ctx,
		chromedp.Navigate("data:text/html,<html><body><h1>Permission Test</h1></body></html>"),
		chromedp.WaitVisible("h1", chromedp.ByQuery),
	)

	if err != nil {
		return DetectPermissionError(err)
	}

	return nil
}

// DetectPermissionError analyzes Chrome errors to identify permission issues
func DetectPermissionError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// Check for screen recording permission issues
	if strings.Contains(errMsg, "screen recording") ||
		strings.Contains(errMsg, "screencapture") ||
		strings.Contains(errMsg, "cannot capture") {
		return ErrScreenRecordingPermission
	}

	// Check for accessibility permission issues
	if strings.Contains(errMsg, "accessibility") ||
		strings.Contains(errMsg, "ui element") {
		return ErrAccessibilityPermission
	}

	// Check for Chrome not found
	if strings.Contains(errMsg, "executable not found") ||
		strings.Contains(errMsg, "chrome not found") {
		return ErrChromeNotFound
	}

	// Return original error if not a known permission issue
	return err
}

// GetPermissionInstructions returns platform-specific instructions for granting permissions
func GetPermissionInstructions(err error) string {
	switch err {
	case ErrScreenRecordingPermission:
		return "Chrome needs Screen Recording permission for browser automation.\n\n" +
			"To grant permission:\n" +
			"1. Open System Settings → Privacy & Security → Screen Recording\n" +
			"2. Enable 'Google Chrome' in the list\n" +
			"3. Restart the Tod application\n\n" +
			"Press 'o' to open System Settings now"
	case ErrAccessibilityPermission:
		return "Chrome needs Accessibility permission for advanced automation.\n\n" +
			"To grant permission:\n" +
			"1. Open System Settings → Privacy & Security → Accessibility\n" +
			"2. Enable 'Google Chrome' in the list\n" +
			"3. Restart the Tod application\n\n" +
			"Press 'o' to open System Settings now"
	case ErrChromeNotFound:
		return "Google Chrome browser not found.\n\n" +
			"Please install Google Chrome from:\n" +
			"https://www.google.com/chrome/"
	default:
		return fmt.Sprintf("Browser error: %v", err)
	}
}