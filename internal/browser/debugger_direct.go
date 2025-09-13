package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// GetPageHTMLDirect connects directly to a Chrome DevTools WebSocket and gets the HTML
func GetPageHTMLDirect(wsURL string) (string, error) {
	// Log the WebSocket URL we're trying to connect to
	fmt.Fprintf(os.Stderr, "GetPageHTMLDirect: Connecting to WebSocket: %s\n", wsURL)
	
	// Connect to the WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	conn, httpResp, err := dialer.Dial(wsURL, http.Header{})
	if err != nil {
		if httpResp != nil {
			fmt.Fprintf(os.Stderr, "WebSocket connection failed with status: %d\n", httpResp.StatusCode)
		}
		return "", fmt.Errorf("failed to connect to WebSocket %s: %w", wsURL, err)
	}
	defer conn.Close()
	
	fmt.Fprintf(os.Stderr, "WebSocket connection established successfully\n")
	
	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	
	// Counter for message IDs
	messageID := 1
	
	// Helper function to send a command
	sendCommand := func(method string, params interface{}) error {
		msg := map[string]interface{}{
			"id":     messageID,
			"method": method,
			"params": params,
		}
		messageID++
		return conn.WriteJSON(msg)
	}
	
	// Helper to read until we get a response with matching ID
	readResponseForID := func(expectedID int) (map[string]interface{}, error) {
		for {
			var resp map[string]interface{}
			if err := conn.ReadJSON(&resp); err != nil {
				return nil, err
			}
			// Check if this is an event (no ID) or our response
			if id, hasID := resp["id"]; hasID {
				if int(id.(float64)) == expectedID {
					return resp, nil
				}
			}
			// If it's an event or different ID, log and continue
			fmt.Fprintf(os.Stderr, "Skipping message: %v\n", resp["method"])
		}
	}
	
	// Enable necessary domains
	currentID := messageID
	if err := sendCommand("DOM.enable", map[string]interface{}{}); err != nil {
		return "", fmt.Errorf("failed to enable DOM: %w", err)
	}
	readResponseForID(currentID) // Read the acknowledgment
	
	currentID = messageID
	if err := sendCommand("Runtime.enable", map[string]interface{}{}); err != nil {
		return "", fmt.Errorf("failed to enable Runtime: %w", err)
	}
	readResponseForID(currentID) // Read the acknowledgment
	
	// Enable Page domain for better control
	currentID = messageID
	if err := sendCommand("Page.enable", map[string]interface{}{}); err != nil {
		return "", fmt.Errorf("failed to enable Page: %w", err)
	}
	readResponseForID(currentID) // Read the acknowledgment
	
	// Wait a moment for the page to stabilize
	time.Sleep(2 * time.Second)
	
	// Try to get HTML using JavaScript evaluation with a more robust approach
	currentID = messageID
	if err := sendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": `
			(() => {
				// Try multiple methods to get the HTML
				try {
					// Method 1: Standard outerHTML
					const html = document.documentElement.outerHTML;
					if (html && html.length > 100) {
						return html;
					}
				} catch (e) {}
				
				try {
					// Method 2: Reconstruct from innerHTML
					const head = document.head ? document.head.innerHTML : '';
					const body = document.body ? document.body.innerHTML : '';
					if (head || body) {
						return '<!DOCTYPE html><html><head>' + head + '</head><body>' + body + '</body></html>';
					}
				} catch (e) {}
				
				try {
					// Method 3: Clone and serialize
					const clone = document.documentElement.cloneNode(true);
					return new XMLSerializer().serializeToString(clone);
				} catch (e) {}
				
				// Fallback: Return what we can
				return document.documentElement.outerHTML || document.body.outerHTML || '<html>Unable to capture</html>';
			})()
		`,
		"returnByValue": true,
		"awaitPromise": false,
	}); err != nil {
		return "", fmt.Errorf("failed to send evaluate command: %w", err)
	}
	
	// Read the result with correct ID
	resp, err := readResponseForID(currentID)
	if err != nil {
		return "", fmt.Errorf("failed to read evaluate response: %w", err)
	}
	
	// Log the response for debugging
	fmt.Fprintf(os.Stderr, "Runtime.evaluate response type: %T\n", resp["result"])
	
	// Extract the HTML from the response - the structure is different than expected
	if result, ok := resp["result"].(map[string]interface{}); ok {
		// The value might be directly in result, not nested
		if value, ok := result["value"].(string); ok && value != "" {
			fmt.Fprintf(os.Stderr, "Got HTML via Runtime.evaluate, length: %d\n", len(value))
			return value, nil
		}
		// Try the nested structure as well
		if resultData, ok := result["result"].(map[string]interface{}); ok {
			if value, ok := resultData["value"].(string); ok && value != "" {
				fmt.Fprintf(os.Stderr, "Got HTML via Runtime.evaluate (nested), length: %d\n", len(value))
				return value, nil
			}
		}
	}
	
	fmt.Fprintf(os.Stderr, "Runtime.evaluate didn't return HTML, trying DOM API fallback\n")
	
	// Fallback: Try using DOM.getDocument and DOM.getOuterHTML
	currentID = messageID
	if err := sendCommand("DOM.getDocument", map[string]interface{}{
		"depth": -1,
		"pierce": true,
	}); err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}
	
	resp, err = readResponseForID(currentID)
	if err != nil {
		return "", fmt.Errorf("failed to read document response: %w", err)
	}
	
	// Extract root node ID
	var rootNodeID int64
	fmt.Fprintf(os.Stderr, "DOM.getDocument response: %+v\n", resp)
	
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if root, ok := result["root"].(map[string]interface{}); ok {
			if nodeID, ok := root["nodeId"].(float64); ok {
				rootNodeID = int64(nodeID)
				fmt.Fprintf(os.Stderr, "Found root node ID: %d\n", rootNodeID)
			} else {
				fmt.Fprintf(os.Stderr, "nodeId type: %T\n", root["nodeId"])
			}
		} else {
			fmt.Fprintf(os.Stderr, "No root in result\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "No result in response\n")
	}
	
	if rootNodeID == 0 {
		return "", fmt.Errorf("failed to get root node ID from response")
	}
	
	// Get the outer HTML
	currentID = messageID
	if err := sendCommand("DOM.getOuterHTML", map[string]interface{}{
		"nodeId": rootNodeID,
	}); err != nil {
		return "", fmt.Errorf("failed to get outer HTML: %w", err)
	}
	
	resp, err = readResponseForID(currentID)
	if err != nil {
		return "", fmt.Errorf("failed to read outer HTML response: %w", err)
	}
	
	// Extract the HTML
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if html, ok := result["outerHTML"].(string); ok {
			return html, nil
		}
	}
	
	return "", fmt.Errorf("unable to extract HTML from response")
}

// GetPageInfoDirect gets basic page info using direct WebSocket connection
func GetPageInfoDirect(wsURL string) (map[string]string, error) {
	// Connect to the WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	conn, _, err := dialer.Dial(wsURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()
	
	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	info := make(map[string]string)
	messageID := 1
	
	// Get page info
	msg := map[string]interface{}{
		"id":     messageID,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{
			"expression": `JSON.stringify({
				title: document.title,
				url: window.location.href,
				readyState: document.readyState,
				contentLength: document.documentElement.innerHTML.length
			})`,
			"returnByValue": true,
		},
	}
	
	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	
	var resp map[string]interface{}
	if err := conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Extract the info
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if resultData, ok := result["result"].(map[string]interface{}); ok {
			if value, ok := resultData["value"].(string); ok {
				var pageInfo map[string]interface{}
				if err := json.Unmarshal([]byte(value), &pageInfo); err == nil {
					for k, v := range pageInfo {
						info[k] = fmt.Sprintf("%v", v)
					}
				}
			}
		}
	}
	
	return info, nil
}

// ExtractTargetID extracts the target ID from a WebSocket debugger URL
func ExtractTargetID(wsURL string) string {
	// WebSocket URL format: ws://localhost:9222/devtools/page/TARGET_ID
	parts := strings.Split(wsURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// NavigateToURLDirect navigates to a URL using Chrome DevTools Protocol directly
func NavigateToURLDirect(wsURL string, targetURL string) error {
	// Log the WebSocket URL we're trying to connect to
	fmt.Fprintf(os.Stderr, "NavigateToURLDirect: Connecting to WebSocket: %s\n", wsURL)
	
	// Connect to the WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	
	conn, httpResp, err := dialer.Dial(wsURL, http.Header{})
	if err != nil {
		if httpResp != nil {
			fmt.Fprintf(os.Stderr, "WebSocket connection failed with status: %d\n", httpResp.StatusCode)
		}
		return fmt.Errorf("failed to connect to WebSocket %s: %w", wsURL, err)
	}
	defer conn.Close()
	
	fmt.Fprintf(os.Stderr, "WebSocket connection established successfully\n")
	
	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	
	// Counter for message IDs
	messageID := 1
	
	// Helper function to send a command
	sendCommand := func(method string, params interface{}) error {
		msg := map[string]interface{}{
			"id":     messageID,
			"method": method,
			"params": params,
		}
		messageID++
		return conn.WriteJSON(msg)
	}
	
	// Helper to read until we get a response with matching ID
	readResponseForID := func(expectedID int) (map[string]interface{}, error) {
		for {
			var resp map[string]interface{}
			if err := conn.ReadJSON(&resp); err != nil {
				return nil, err
			}
			// Check if this is an event (no ID) or our response
			if id, hasID := resp["id"]; hasID {
				if int(id.(float64)) == expectedID {
					return resp, nil
				}
			}
			// If it's an event or different ID, log and continue
			if method, hasMethod := resp["method"]; hasMethod {
				fmt.Fprintf(os.Stderr, "Skipping event: %v\n", method)
			}
		}
	}
	
	// Enable Page domain
	currentID := messageID
	if err := sendCommand("Page.enable", map[string]interface{}{}); err != nil {
		return fmt.Errorf("failed to enable Page: %w", err)
	}
	readResponseForID(currentID) // Read the acknowledgment
	
	// Navigate to the URL
	fmt.Fprintf(os.Stderr, "Navigating to URL: %s\n", targetURL)
	currentID = messageID
	if err := sendCommand("Page.navigate", map[string]interface{}{
		"url": targetURL,
	}); err != nil {
		return fmt.Errorf("failed to send navigate command: %w", err)
	}
	
	// Read the navigation response
	resp, err := readResponseForID(currentID)
	if err != nil {
		return fmt.Errorf("failed to read navigate response: %w", err)
	}
	
	// Check if navigation was successful
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if frameId, ok := result["frameId"].(string); ok && frameId != "" {
			fmt.Fprintf(os.Stderr, "Navigation successful, frameId: %s\n", frameId)
		}
		if errorText, ok := result["errorText"].(string); ok && errorText != "" {
			return fmt.Errorf("navigation failed: %s", errorText)
		}
	}
	
	// Wait for page to load
	fmt.Fprintf(os.Stderr, "Waiting for page to load...\n")
	time.Sleep(2 * time.Second)
	
	return nil
}