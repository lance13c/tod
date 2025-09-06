#!/bin/bash

# Script to help submit Tod to Homebrew Cask
# Usage: ./scripts/submit-to-homebrew.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🍺 Tod Homebrew Cask Submission Helper"
echo "======================================"

# Check if we have the latest release
if [ ! -f "$PROJECT_DIR/dist/tod-1.0.0.dmg" ]; then
    echo "❌ DMG file not found. Run 'make release' first."
    exit 1
fi

# Get the SHA256
SHA256=$(shasum -a 256 "$PROJECT_DIR/dist/tod-1.0.0.dmg" | cut -d' ' -f1)
echo "✅ SHA256: $SHA256"

# Check if homebrew-cask fork exists
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Upload dist/tod-0.0.1.dmg to GitHub Releases:"
echo "   https://github.com/ciciliostudio/tod/releases/new"
echo ""
echo "2. Fork the homebrew-cask repository:"
echo "   https://github.com/Homebrew/homebrew-cask/fork"
echo ""
echo "3. Clone your fork and add the cask:"
echo "   git clone https://github.com/YOUR_USERNAME/homebrew-cask.git"
echo "   cd homebrew-cask"
echo "   cp $PROJECT_DIR/homebrew/tod.rb Casks/t/tod.rb"
echo ""
echo "4. Verify the cask works:"
echo "   brew install --cask ./Casks/t/tod.rb"
echo ""
echo "5. Submit a pull request with title:"
echo "   'tod 1.0.0 (new cask)'"
echo ""
echo "💡 The SHA256 in homebrew/tod.rb should already be correct: $SHA256"

# Verify the SHA256 in our cask file
CASK_SHA256=$(grep -o 'sha256 "[^"]*"' "$PROJECT_DIR/homebrew/tod.rb" | cut -d'"' -f2)
if [ "$SHA256" = "$CASK_SHA256" ]; then
    echo "✅ SHA256 in cask file is correct"
else
    echo "⚠️  SHA256 mismatch! Update homebrew/tod.rb with: $SHA256"
fi

echo ""
echo "🚀 Ready for submission!"
