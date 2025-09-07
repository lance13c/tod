package browser

import (
	"fmt"
)

// PlaywrightConfig contains configuration for Playwright browser instances
type PlaywrightConfig struct {
	DebugPort int
	Headless  bool
	SlowMo    float64
}

// DefaultPlaywrightConfig returns a default configuration with debugging enabled
func DefaultPlaywrightConfig() *PlaywrightConfig {
	return &PlaywrightConfig{
		DebugPort: 9222,
		Headless:  false,
		SlowMo:    0,
	}
}

// GetPlaywrightLaunchArgs returns the command line arguments for enabling debugging
func GetPlaywrightLaunchArgs(debugPort int) []string {
	return []string{
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
		"--remote-debugging-address=127.0.0.1",
	}
}

// GetChromeDebugURL returns the URL to connect to Chrome DevTools
func GetChromeDebugURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// GetWebSocketDebugURL returns the WebSocket URL for CDP connection
func GetWebSocketDebugURL(port int) string {
	return fmt.Sprintf("ws://127.0.0.1:%d", port)
}