# Chrome Remote Debugging Guide

## The Problem

When you run Chrome with just `--remote-debugging-port=9229`, it may not properly expose the debugging interface. This is because:

1. Chrome needs a separate user data directory to avoid conflicts with existing instances
2. The debugging address needs to be explicitly set
3. Some additional flags help ensure the debugging interface works correctly

## Solution

### Method 1: Use the provided script

```bash
./start_chrome_debug.sh 9229
```

### Method 2: Manual command with all required flags

```bash
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9229 \
  --remote-debugging-address=127.0.0.1 \
  --user-data-dir=/tmp/chrome-debug-9229 \
  --no-first-run \
  --no-default-browser-check
```

### Method 3: Use Chrome Canary (if installed)

```bash
/Applications/Google\ Chrome\ Canary.app/Contents/MacOS/Google\ Chrome\ Canary \
  --remote-debugging-port=9229 \
  --user-data-dir=/tmp/chrome-debug-canary
```

## Key Flags Explained

- `--remote-debugging-port=9229`: Enables the DevTools Protocol on port 9229
- `--remote-debugging-address=127.0.0.1`: Binds to localhost (some versions need this)
- `--user-data-dir=/tmp/chrome-debug-9229`: Uses a separate profile to avoid conflicts
- `--no-first-run`: Skips first-run wizards
- `--no-default-browser-check`: Prevents default browser prompts

## Testing the Connection

After starting Chrome, test that the debugging port is working:

```bash
# Check version endpoint
curl http://127.0.0.1:9229/json/version

# List all targets
curl http://127.0.0.1:9229/json/list

# Or use the TOD test tool
go run tools/test_chrome_debug.go
```

## Common Issues

### Issue: "Connection refused" when accessing the debugging port

**Solution**: Make sure to include `--user-data-dir` flag. Chrome won't enable remote debugging if it's using your default profile.

### Issue: Chrome opens but debugging port doesn't work

**Solution**: Close ALL Chrome instances first, then start with the debugging flags. Use:
```bash
killall "Google Chrome"
```

### Issue: Port already in use

**Solution**: Either kill the process using the port or choose a different port:
```bash
lsof -ti:9229 | xargs kill -9
```

## Using with TOD

Once Chrome is running with debugging enabled, you can use TOD's Chrome Debugger Scanner:

1. Start Chrome with debugging: `./start_chrome_debug.sh`
2. Run TOD: `./toddev` or `go run .`
3. Select "Chrome Debugger Scanner" from the menu
4. TOD will automatically detect and connect to Chrome

## Playwright Integration

TOD's browser client automatically enables debugging when starting browsers. The debugging port can be configured via environment variables:

```bash
# Set custom debugging port (default: 9222)
export BROWSER_DEBUG_PORT=9229

# Disable debugging completely
export BROWSER_DISABLE_DEBUG=true
```