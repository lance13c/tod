#!/bin/bash

# Tod Development Environment Setup Script
# Sets up Go 1.24+, air hot reload, and global toddev access

set -e

echo "🎯 Tod Development Environment Setup"
echo "======================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    print_error "This script is designed for macOS. Please install manually."
    exit 1
fi

# Check if Homebrew is installed
if ! command -v brew >/dev/null 2>&1; then
    print_error "Homebrew not found. Please install Homebrew first:"
    echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    exit 1
fi

print_status "Checking current Go version..."
CURRENT_GO=$(go version 2>/dev/null | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "none")
print_status "Current Go: $CURRENT_GO"

# Check Go version and install 1.24+ if needed
GO_124_PATH="/opt/homebrew/opt/go@1.24/bin"
if [[ "$CURRENT_GO" < "go1.24" ]] || ! command -v go >/dev/null 2>&1; then
    print_status "Installing Go 1.24..."
    brew install go@1.24
    print_success "Go 1.24 installed"
else
    # Check if Go 1.24 is available via Homebrew
    if [[ ! -d "/opt/homebrew/opt/go@1.24" ]]; then
        print_status "Installing Go 1.24 (air requirement)..."
        brew install go@1.24
        print_success "Go 1.24 installed"
    else
        print_success "Go 1.24 already available"
    fi
fi

# Setup shell configuration
SHELL_CONFIG=""
if [[ "$SHELL" == *"zsh"* ]] || [[ -n "$ZSH_VERSION" ]]; then
    SHELL_CONFIG="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    SHELL_CONFIG="$HOME/.bashrc"
    [[ -f "$HOME/.bash_profile" ]] && SHELL_CONFIG="$HOME/.bash_profile"
else
    print_warning "Unknown shell. Please manually add paths to your shell config."
    SHELL_CONFIG=""
fi

if [[ -n "$SHELL_CONFIG" ]]; then
    print_status "Updating $SHELL_CONFIG..."
    
    # Backup shell config
    cp "$SHELL_CONFIG" "$SHELL_CONFIG.backup.$(date +%Y%m%d_%H%M%S)"
    print_status "Backed up $SHELL_CONFIG"
    
    # Add Go 1.24 to PATH if not already present
    if ! grep -q 'go@1.24/bin' "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Tod Development - Go 1.24" >> "$SHELL_CONFIG"
        echo 'export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"' >> "$SHELL_CONFIG"
        print_success "Added Go 1.24 to PATH in $SHELL_CONFIG"
    else
        print_success "Go 1.24 already in PATH"
    fi
    
    # Add ~/bin to PATH if not already present
    if ! grep -q 'HOME/bin' "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Tod Development - User binaries" >> "$SHELL_CONFIG"
        echo 'export PATH="$HOME/bin:$PATH"' >> "$SHELL_CONFIG"
        print_success "Added ~/bin to PATH in $SHELL_CONFIG"
    else
        print_success "~/bin already in PATH"
    fi
fi

# Export paths for current session
export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"
export PATH="$HOME/bin:$PATH"

print_status "Current Go version: $(go version | cut -d' ' -f3)"

# Install air for hot reload
print_status "Installing air for hot reload..."
if ! command -v air >/dev/null 2>&1; then
    go install github.com/air-verse/air@latest
    print_success "Air installed: $(air -v | head -1)"
else
    print_success "Air already installed: $(air -v | head -1)"
fi

# Setup Tod development
print_status "Setting up Tod development environment..."

# Build initial toddev
make build-dev
print_success "Built toddev development binary"

# Create global toddev with auto-update symlink
make install-dev-link
print_success "Created global toddev with auto-update symlink"

# Create ~/bin directory if it doesn't exist
mkdir -p "$HOME/bin"

# Test the setup
print_status "Testing setup..."
if command -v toddev >/dev/null 2>&1; then
    print_success "toddev is accessible globally"
else
    print_warning "toddev not found in PATH. You may need to restart your shell."
fi

echo ""
echo "🎉 Setup Complete!"
echo "=================="
echo ""
echo "📋 What was installed:"
echo "  • Go 1.24.7 (required for air)"
echo "  • Air hot reload tool"
echo "  • Global toddev binary with auto-update"
echo "  • Updated shell configuration ($SHELL_CONFIG)"
echo ""
echo "🚀 Quick Start:"
echo "  1. Start hot reload:  make hotdev"
echo "  2. Use globally:      toddev (from any directory)"
echo "  3. Get help:          make help"
echo ""
echo "💡 Development Workflow:"
echo "  • Run 'make hotdev' in one terminal (keeps running)"
echo "  • Use 'toddev' from anywhere - it auto-updates!"
echo "  • Make code changes - they instantly apply to global toddev"
echo ""
echo "🔧 Useful Commands:"
echo "  make hotdev          # Start hot reload development"
echo "  make dev             # Quick run without reload"
echo "  make test-cicilio    # Test in ciciliostudio repo"
echo "  make help            # Show all commands"
echo ""

# Check if we need to restart shell
if [[ -n "$SHELL_CONFIG" ]]; then
    print_warning "Shell configuration updated. Run one of:"
    echo "  source $SHELL_CONFIG"
    echo "  # OR restart your terminal"
    echo ""
fi

print_success "Tod development environment ready!"
echo ""
echo "🎯 Try it now:"
echo "  make hotdev    # Start developing with hot reload"