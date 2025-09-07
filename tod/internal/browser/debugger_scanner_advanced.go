package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// GetPageHTMLAdvanced uses advanced Chrome DevTools Protocol methods to capture page content
func GetPageHTMLAdvanced(debuggerURL string) (string, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debuggerURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var finalHTML string
	
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Enable necessary domains
			if err := page.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable page domain: %w", err)
			}
			
			if err := dom.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable DOM domain: %w", err)
			}
			
			if err := runtime.Enable().Do(ctx); err != nil {
				return fmt.Errorf("failed to enable runtime domain: %w", err)
			}
			
			// Wait for the page to be fully loaded
			time.Sleep(3 * time.Second)
			
			// Try to get the page source using Page.getResourceTree
			frameTree, err := page.GetResourceTree().Do(ctx)
			if err == nil && frameTree != nil && frameTree.Frame != nil {
				// Try to capture the main frame's HTML  
				content, err := page.GetResourceContent(frameTree.Frame.ID, frameTree.Frame.URL).Do(ctx)
				if err == nil && len(content) > 0 {
					finalHTML = string(content)
					return nil
				}
			}
			
			// Alternative: Use Runtime.evaluate to get all content including dynamic updates
			res, exp, err := runtime.Evaluate(`
				(() => {
					// Function to get all text content from the page
					function getAllContent() {
						let content = '<!DOCTYPE html>\n<html>\n<head>\n';
						
						// Get head content
						const head = document.head;
						if (head) {
							content += head.innerHTML;
						}
						content += '</head>\n<body>\n';
						
						// Get body content
						const body = document.body;
						if (body) {
							content += body.innerHTML;
						}
						content += '</body>\n</html>';
						
						return content;
					}
					
					// Try to get the full page
					try {
						// First try the standard way
						const html = document.documentElement.outerHTML;
						if (html && html.length > 100) {
							return html;
						}
					} catch (e) {
						// Continue to fallback
					}
					
					// Fallback to manual reconstruction
					return getAllContent();
				})()
			`).Do(ctx)
			
			if err != nil {
				return fmt.Errorf("failed to evaluate JavaScript: %w", err)
			}
			
			if exp != nil {
				return fmt.Errorf("JavaScript evaluation exception: %v", exp)
			}
			
			// Extract the string value from the result
			if res != nil && res.Type == runtime.TypeString {
				// The value is already a string in the result
				finalHTML = strings.Trim(string(res.Value), "\"")
			}
			
			if finalHTML == "" {
				return fmt.Errorf("unable to extract HTML content")
			}
			
			return nil
		}),
	)
	
	if err != nil {
		return "", fmt.Errorf("failed to get advanced page HTML: %w", err)
	}
	
	return finalHTML, nil
}

// GetPageSnapshot captures the page using CDP snapshot
func GetPageSnapshot(debuggerURL string) (string, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debuggerURL)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var snapshot string
	
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Capture a snapshot of the current page
			documents, err := page.CaptureSnapshot().Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to capture snapshot: %w", err)
			}
			
			// The documents is already a string containing the snapshot
			if documents != "" {
				snapshot = documents
			}
			
			if snapshot == "" {
				// Fallback to simple DOM capture
				rootNode, err := dom.GetDocument().WithDepth(-1).Do(ctx)
				if err != nil {
					return fmt.Errorf("failed to get document: %w", err)
				}
				
				htmlContent, err := dom.GetOuterHTML().WithNodeID(rootNode.NodeID).Do(ctx)
				if err != nil {
					return fmt.Errorf("failed to get HTML: %w", err)
				}
				
				snapshot = htmlContent
			}
			
			return nil
		}),
	)
	
	if err != nil {
		return "", fmt.Errorf("failed to get page snapshot: %w", err)
	}
	
	return snapshot, nil
}

