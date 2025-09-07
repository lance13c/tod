package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// DebuggerTarget represents a Chrome DevTools target
type DebuggerTarget struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// DebuggerScanResult contains the results of scanning for Chrome debugger instances
type DebuggerScanResult struct {
	Port    int
	Targets []DebuggerTarget
}

// ScanForChromeDebugger scans common Chrome debugger ports for open instances
func ScanForChromeDebugger() ([]DebuggerScanResult, error) {
	// Common Chrome debugger ports (9229 first as it's our default)
	ports := []int{9229, 9222, 9223, 9224, 9225}
	
	var results []DebuggerScanResult
	var lastError error
	
	for _, port := range ports {
		// Try both localhost and 127.0.0.1
		for _, host := range []string{"localhost", "127.0.0.1"} {
			targets, err := getTargetsFromPortAndHost(port, host)
			if err == nil && len(targets) > 0 {
				results = append(results, DebuggerScanResult{
					Port:    port,
					Targets: targets,
				})
				break // Found on this port, no need to try other host
			} else if err != nil {
				lastError = err
			}
		}
	}
	
	// If no results found and we have an error, return it for debugging
	if len(results) == 0 && lastError != nil {
		return results, fmt.Errorf("no debugger instances found, last error: %w", lastError)
	}
	
	return results, nil
}

// getTargetsFromPort fetches available targets from a Chrome debugger port
func getTargetsFromPort(port int) ([]DebuggerTarget, error) {
	return getTargetsFromPortAndHost(port, "localhost")
}

// getTargetsFromPortAndHost fetches available targets from a Chrome debugger port with specific host
func getTargetsFromPortAndHost(port int, host string) ([]DebuggerTarget, error) {
	url := fmt.Sprintf("http://%s:%d/json/list", host, port)
	
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var targets []DebuggerTarget
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, err
	}
	
	// Log what we got for debugging
	fmt.Fprintf(os.Stderr, "Got %d targets from %s:%d\n", len(targets), host, port)
	for i, target := range targets {
		fmt.Fprintf(os.Stderr, "  Target %d: Type=%s, URL=%s, WebSocket=%s\n", 
			i, target.Type, target.URL, target.WebSocketDebuggerURL)
	}
	
	// Return all targets, not just pages (sometimes the type field differs)
	// We'll filter them in the UI if needed
	return targets, nil
}

// GetPageHTML connects to a Chrome debugger target and retrieves the full page HTML
func GetPageHTML(debuggerURL string) (string, error) {
	// Connect directly to the existing tab's WebSocket debugger URL
	// Using WithTargetID to connect to existing tab instead of creating new one
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debuggerURL)
	defer allocCancel()
	
	// Create context that attaches to the existing tab
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(""))
	defer cancel()
	
	// Set a longer timeout for complex pages
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	var html string
	var pageURL string
	
	err := chromedp.Run(ctx,
		// First get the page URL to understand what we're dealing with
		chromedp.Location(&pageURL),
		
		// Wait for the page to be in a ready state
		chromedp.WaitReady("body", chromedp.ByQuery),
		
		// Additional wait for dynamic content to load
		chromedp.Sleep(2*time.Second),
		
		// Try multiple methods to get the HTML
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Method 1: Try JavaScript evaluation for fully rendered content
			var jsHTML string
			err := chromedp.Evaluate(`
				(() => {
					// Wait a moment for any final rendering
					return new Promise(resolve => {
						setTimeout(() => {
							resolve(document.documentElement.outerHTML);
						}, 500);
					});
				})()
			`, &jsHTML).Do(ctx)
			
			if err == nil && jsHTML != "" && len(jsHTML) > 100 {
				html = jsHTML
				return nil
			}
			
			// Method 2: Try getting innerHTML if outerHTML is minimal
			err = chromedp.Evaluate(`document.documentElement.innerHTML`, &jsHTML).Do(ctx)
			if err == nil && jsHTML != "" && len(jsHTML) > 100 {
				html = "<!DOCTYPE html><html>" + jsHTML + "</html>"
				return nil
			}
			
			// Method 3: Use DOM API with full depth
			rootNode, err := dom.GetDocument().WithDepth(-1).WithPierce(true).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to get document: %w", err)
			}
			
			// Get the full document including shadow DOM
			htmlContent, err := dom.GetOuterHTML().
				WithNodeID(rootNode.NodeID).
				WithBackendNodeID(rootNode.BackendNodeID).
				Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to get outer HTML: %w", err)
			}
			
			if htmlContent != "" && len(htmlContent) > 100 {
				html = htmlContent
				return nil
			}
			
			// Method 4: As a last resort, try to serialize the DOM
			var serialized string
			err = chromedp.Evaluate(`
				new XMLSerializer().serializeToString(document)
			`, &serialized).Do(ctx)
			if err == nil && serialized != "" {
				html = serialized
				return nil
			}
			
			return fmt.Errorf("all methods failed to get substantial HTML content (URL: %s)", pageURL)
		}),
	)
	
	if err != nil {
		return "", fmt.Errorf("failed to get page HTML: %w", err)
	}
	
	// If we still got minimal HTML, add a note
	if len(html) < 200 {
		html = fmt.Sprintf("<!-- WARNING: Minimal HTML captured. Page might be protected or not fully loaded -->\n<!-- URL: %s -->\n%s", pageURL, html)
	}
	
	return html, nil
}

// SavePageHTML saves the full HTML of a page to a file
func SavePageHTML(debuggerURL string, filepath string) error {
	html, err := GetPageHTML(debuggerURL)
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}
	
	return nil
}

// GetPageInfo retrieves basic information about a page from a debugger target
func GetPageInfo(debuggerURL string) (*PageInfo, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debuggerURL)
	defer allocCancel()
	
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	
	// Set a timeout for the operation
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	var title, url string
	err := chromedp.Run(ctx,
		chromedp.Title(&title),
		chromedp.Location(&url),
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}
	
	return &PageInfo{
		URL:   url,
		Title: title,
	}, nil
}