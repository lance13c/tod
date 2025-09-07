//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// This is a standalone tool to test Chrome debugger connectivity
// Run with: go run tools/test_chrome_debug.go

func main() {
	fmt.Println("Testing Chrome Debugger connectivity...")
	fmt.Println("=====================================")
	
	testPorts()
}

func testPorts() {
	ports := []int{9222, 9223, 9224, 9225, 9222}
	hosts := []string{"localhost", "127.0.0.1", "0.0.0.0"}
	
	foundAny := false
	
	for _, port := range ports {
		for _, host := range hosts {
			if testConnection(host, port) {
				foundAny = true
			}
		}
	}
	
	if !foundAny {
		fmt.Println("\n✗ No Chrome Debugger instances found")
		fmt.Println("\nTo enable Chrome debugging, try one of these commands:")
		fmt.Println("\n1. With a temporary profile (recommended):")
		fmt.Println("   /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome \\")
		fmt.Println("     --remote-debugging-port=9222 \\")
		fmt.Println("     --user-data-dir=/tmp/chrome-debug")
		
		fmt.Println("\n2. Create a new Chrome instance:")
		fmt.Println("   open -na 'Google Chrome' --args \\")
		fmt.Println("     --remote-debugging-port=9222 \\")
		fmt.Println("     --user-data-dir=/tmp/chrome-debug")
		
		fmt.Println("\n3. If Chrome is already running, close ALL instances first, then:")
		fmt.Println("   killall 'Google Chrome'")
		fmt.Println("   Then run one of the commands above")
		
		fmt.Println("\nNote: The --user-data-dir flag is important to avoid conflicts with existing Chrome sessions")
	}
}

func testConnection(host string, port int) bool {
	url := fmt.Sprintf("http://%s:%d/json/version", host, port)
	
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false
	}
	
	body, _ := io.ReadAll(resp.Body)
	
	var version map[string]interface{}
	if err := json.Unmarshal(body, &version); err != nil {
		return false
	}
	
	fmt.Printf("\n✓ Found Chrome Debugger at %s:%d\n", host, port)
	fmt.Printf("  Browser: %v\n", version["Browser"])
	fmt.Printf("  Protocol: %v\n", version["Protocol-Version"])
	fmt.Printf("  WebSocket URL: %v\n", version["webSocketDebuggerUrl"])
	
	// Now try to get targets
	listURL := fmt.Sprintf("http://%s:%d/json/list", host, port)
	listResp, err := client.Get(listURL)
	if err == nil && listResp.StatusCode == http.StatusOK {
		defer listResp.Body.Close()
		listBody, _ := io.ReadAll(listResp.Body)
		
		var targets []map[string]interface{}
		if err := json.Unmarshal(listBody, &targets); err == nil {
			fmt.Printf("  Found %d targets:\n", len(targets))
			for i, target := range targets {
				if i < 5 { // Only show first 5 targets
					fmt.Printf("    %d. %s (%s)\n", i+1, target["title"], target["type"])
					if url, ok := target["url"].(string); ok && url != "" {
						fmt.Printf("       URL: %s\n", url)
					}
				}
			}
			if len(targets) > 5 {
				fmt.Printf("    ... and %d more\n", len(targets)-5)
			}
		}
	}
	
	return true
}
