package browser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ChromeTarget represents a Chrome DevTools target
type ChromeTarget struct {
	ID                    string `json:"id"`
	Type                  string `json:"type"`
	Title                 string `json:"title"`
	URL                   string `json:"url"`
	WebSocketDebuggerURL  string `json:"webSocketDebuggerUrl"`
	DevtoolsFrontendURL   string `json:"devtoolsFrontendUrl"`
}

// GetChromeWebSocketURL gets the WebSocket URL for the first page target
func GetChromeWebSocketURL(host, port string) (string, error) {
	// Get list of targets from Chrome
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/json", host, port))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Chrome DevTools: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	var targets []ChromeTarget
	if err := json.Unmarshal(body, &targets); err != nil {
		return "", fmt.Errorf("failed to parse targets: %w", err)
	}
	
	// Find the first page target
	for _, target := range targets {
		if target.Type == "page" && target.WebSocketDebuggerURL != "" {
			return target.WebSocketDebuggerURL, nil
		}
	}
	
	return "", fmt.Errorf("no page targets found")
}

// GetAllChromeTargets gets all Chrome DevTools targets
func GetAllChromeTargets(host, port string) ([]ChromeTarget, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:%s/json", host, port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Chrome DevTools: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var targets []ChromeTarget
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, fmt.Errorf("failed to parse targets: %w", err)
	}
	
	return targets, nil
}