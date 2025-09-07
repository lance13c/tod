#!/bin/bash

# Start Chrome with remote debugging enabled
# This script ensures Chrome starts with the correct flags for debugging

PORT=${1:-9222}
echo "Starting Chrome with remote debugging on port $PORT..."

# Kill any existing Chrome instances on this port (optional)
# lsof -ti:$PORT | xargs kill -9 2>/dev/null

# Create a temporary user data directory
USER_DATA_DIR="/tmp/chrome-debug-$PORT"
mkdir -p "$USER_DATA_DIR"

# Start Chrome with all necessary flags
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=$PORT \
  --remote-debugging-address=127.0.0.1 \
  --user-data-dir="$USER_DATA_DIR" \
  --no-first-run \
  --no-default-browser-check \
  --disable-background-timer-throttling \
  --disable-backgrounding-occluded-windows \
  --disable-renderer-backgrounding \
  --disable-features=TranslateUI \
  --disable-ipc-flooding-protection \
  --enable-features=NetworkService,NetworkServiceInProcess &

# Give Chrome a moment to start
sleep 2

# Test the connection
echo "Testing connection to Chrome DevTools Protocol..."
if curl -s http://127.0.0.1:$PORT/json/version > /dev/null 2>&1; then
    echo "✓ Chrome is running with debugging enabled on port $PORT"
    echo "✓ You can now use TOD's Chrome Debugger Scanner"
    echo ""
    echo "API endpoints available:"
    echo "  http://127.0.0.1:$PORT/json/version - Version info"
    echo "  http://127.0.0.1:$PORT/json/list    - List of targets"
    echo ""
    echo "User data directory: $USER_DATA_DIR"
else
    echo "✗ Failed to connect to Chrome DevTools Protocol"
    echo "Check if Chrome started correctly"
fi
