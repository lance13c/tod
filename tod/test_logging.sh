#!/bin/bash

# Create .tod directory if it doesn't exist
mkdir -p .tod

# Clear the log file
> .tod/api_calls.log

echo "Log file cleared. Ready to test."
echo "Monitoring .tod/api_calls.log for API calls..."
echo "Press Ctrl+C to stop"
echo ""

# Tail the log file
tail -f .tod/api_calls.log