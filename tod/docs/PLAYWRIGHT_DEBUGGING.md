# Playwright with Chrome DevTools Protocol (CDP) Debugging

## Overview
When Playwright launches a browser, you can configure it to enable the Chrome DevTools Protocol debugging port. This allows you to:
- Connect Chrome DevTools to inspect the browser
- Use the TOD Chrome Debugger Scanner to view page HTML
- Connect other automation tools to the same browser instance
- Debug your tests while they're running

## Configuration Options

### 1. Using Chromedp (Current Implementation)
The existing `browser.Client` uses chromedp. To enable debugging port:

```go
opts := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.Flag("remote-debugging-port", "9222"),
    chromedp.Flag("remote-debugging-address", "127.0.0.1"),
    // other flags...
)
```

### 2. Using Playwright (New Option)
Configure Playwright to always enable debugging:

```go
// In your test setup
config := browser.DefaultPlaywrightConfig()
config.DebugPort = 9222
config.Headless = false // Usually want headed mode for debugging

browser, err := browser.LaunchBrowserWithDebugger(config)
```

### 3. Environment-based Configuration
Set debugging port via environment variable:

```bash
# Enable debugging on port 9222
export BROWSER_DEBUG_PORT=9222

# Or in your .env file
BROWSER_DEBUG_PORT=9222
```

Then in your code:
```go
debugPort := os.Getenv("BROWSER_DEBUG_PORT")
if debugPort != "" {
    port, _ := strconv.Atoi(debugPort)
    config.DebugPort = port
}
```

## Using with TOD Chrome Debugger Scanner

1. **Start your Playwright test** with debugging enabled:
   ```go
   browser, _ := LaunchBrowserWithDebugger(&PlaywrightConfig{
       DebugPort: 9222,
       Headless:  false,
   })
   ```

2. **Run TOD** and select "Chrome Debugger Scanner"

3. **TOD will find** your Playwright browser on port 9222

4. **Select the page** and view its HTML, inspect elements, etc.

## Multiple Browser Instances

If running multiple Playwright instances, use different ports:

```go
// Test instance 1
config1 := &PlaywrightConfig{DebugPort: 9222}

// Test instance 2  
config2 := &PlaywrightConfig{DebugPort: 9223}

// Test instance 3
config3 := &PlaywrightConfig{DebugPort: 9224}
```

## Security Considerations

⚠️ **Warning**: Enabling debugging port makes the browser accessible over network.

For security:
- Always bind to `127.0.0.1` (localhost only)
- Don't expose debugging ports in production
- Use firewall rules to block external access
- Consider using random ports in CI/CD

## Connecting Multiple Tools

Once debugging is enabled, you can connect multiple tools simultaneously:

1. **Chrome DevTools**: Navigate to `chrome://inspect`
2. **TOD Scanner**: Use the Chrome Debugger Scanner feature
3. **Another Playwright**: Connect via CDP:
   ```go
   browser, _ := ConnectOverCDP("ws://localhost:9222")
   ```
4. **Puppeteer/Other tools**: Most automation tools support CDP connection

## Troubleshooting

### Port Already in Use
If you get "address already in use" error:
```bash
# Find process using port 9222
lsof -i :9222

# Kill the process
kill -9 <PID>
```

### Can't Connect to Debugging Port
- Ensure browser is running with `--remote-debugging-port`
- Check firewall settings
- Verify the port number matches
- Try `127.0.0.1` instead of `localhost`

### Browser Closes Immediately
- Set `Headless: false` for debugging
- Add `SlowMo` to slow down actions
- Use `page.Pause()` to keep browser open

## Example: Full Test with Debugging

```go
func TestWithDebugging(t *testing.T) {
    // Setup browser with debugging
    config := &PlaywrightConfig{
        DebugPort: 9222,
        Headless:  false,
        SlowMo:    100, // Slow down by 100ms
    }
    
    browser, err := LaunchBrowserWithDebugger(config)
    require.NoError(t, err)
    defer browser.Close()
    
    // Create page
    page, err := browser.NewPage()
    require.NoError(t, err)
    
    // Your test actions
    page.Goto("https://example.com")
    
    // Now you can:
    // 1. Open Chrome DevTools at chrome://inspect
    // 2. Use TOD Chrome Debugger Scanner
    // 3. Inspect the page while test is running
    
    // Keep browser open for inspection
    time.Sleep(30 * time.Second)
}
```