package browser

import "fmt"

// This file shows how to configure Playwright with debugging port enabled
// To use this, first install playwright-go:
// go get github.com/playwright-community/playwright-go

/*
Example configuration for Playwright with debugging:

import (
    "github.com/playwright-community/playwright-go"
)

func LaunchWithDebugging() {
    // Initialize Playwright
    pw, err := playwright.Run()
    if err != nil {
        log.Fatal(err)
    }
    
    // Launch Chrome with debugging port enabled
    browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
        Headless: playwright.Bool(false),
        Args: []string{
            "--remote-debugging-port=9222",
            "--remote-debugging-address=127.0.0.1",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()
    
    // Create a new page
    page, err := browser.NewPage()
    if err != nil {
        log.Fatal(err)
    }
    
    // Navigate to a website
    _, err = page.Goto("https://example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    // Now you can:
    // 1. Connect Chrome DevTools to localhost:9222
    // 2. Use TOD Chrome Debugger Scanner to inspect the page
    // 3. Connect another automation tool via CDP
}

// For Puppeteer (Node.js), the equivalent would be:
//
// const browser = await puppeteer.launch({
//   headless: false,
//   args: [
//     '--remote-debugging-port=9222',
//     '--remote-debugging-address=127.0.0.1'
//   ]
// });

// For Selenium WebDriver with Chrome:
//
// ChromeOptions options = new ChromeOptions();
// options.addArguments("--remote-debugging-port=9222");
// options.addArguments("--remote-debugging-address=127.0.0.1");
// WebDriver driver = new ChromeDriver(options);

*/

// GetDebuggerURL returns the WebSocket URL for connecting to the debugger
func GetDebuggerURL(port int) string {
	return fmt.Sprintf("ws://127.0.0.1:%d", port)
}

// GetDebuggerHTTPURL returns the HTTP URL for the debugger JSON API
func GetDebuggerHTTPURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/json", port)
}