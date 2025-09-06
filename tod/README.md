# TIF - Text-adventure Interface Framework

A delightful CLI testing tool that presents E2E testing as interactive text-adventure journeys, with a focus on magic link authentication flows.

## Features

🎮 **Text Adventure Interface** - Testing as an interactive story
🔗 **Magic Link Support** - Specialized magic link authentication testing  
📧 **Email Integration** - Mailhog, Mailtrap, and other providers
🎨 **Beautiful TUI** - Built with Bubble Tea for rich terminal UI
📝 **Session Recording** - Capture and replay test sessions
🧪 **Test Generation** - Convert sessions to Playwright/Cypress tests

## Installation

### For Users

#### Homebrew Cask (macOS) - Recommended

```bash
# Install via Homebrew Cask
brew install --cask tod

# Run tod from anywhere
tod
```

#### From Release

Download the latest DMG from [Releases](https://github.com/ciciliostudio/tod/releases) and drag Tod.app to your Applications folder.

### For Developers

#### One-Line Development Setup

```bash
git clone https://github.com/ciciliostudio/tod.git
cd tod
./dev-setup.sh
```

This sets up:
- ✅ Go 1.24+ with air hot reload
- ✅ Global `toddev` with auto-update  
- ✅ Complete development workflow

See [DEVELOPMENT.md](./DEVELOPMENT.md) for details.

#### Manual Installation

```bash
# Clone and build
git clone https://github.com/ciciliostudio/tod.git
cd tod
make build

# Or install to $GOPATH/bin
make install

# Or run directly
go run .
```

## Quick Start

### For Users
```bash
# If installed via Homebrew Cask
tod

# Initialize in your project
tod init
```

### For Developers
```bash
# After running ./dev-setup.sh:
make hotdev    # Start hot reload (keep running)
toddev         # Use from anywhere - auto-updates!
```

## Usage

Launch `tif` and you'll enter an interactive text adventure:

```
╔══════════════════════════════════════════╗
║     _______  ___   _______              ║
║    |       ||   | |       |             ║
║    |_     _||   | |    ___|             ║
║      |   |  |   | |   |___              ║
║      |   |  |   | |    ___|             ║
║      |   |  |   | |   |                 ║
║      |___|  |___| |___|                 ║
║                                          ║
║   Text-adventure Interface Framework    ║
╚══════════════════════════════════════════╝

Welcome, brave tester! Choose your path:

  > Start New Journey
    Continue Journey
    Review Past Adventures
    Generate Test Scroll
    Configure Your Realm
    Exit
```

Navigate with ↑/↓, select with Enter, and quit with 'q'.

## Development

```bash
# Run in development mode
make dev

# Run tests
make test

# Build binary
make build

# Install globally
make install

# Clean up
make clean
```

### Release Process

```bash
# Build universal binary for macOS
make build-universal

# Create macOS application bundle
make build-app

# Create DMG installer
make build-dmg

# Generate SHA256 for Homebrew Cask
make sha256

# Full release process (all of the above)
make release
```

The release process creates:
- Universal binary supporting both Intel and Apple Silicon Macs
- Proper macOS application bundle (Tod.app)
- DMG installer for distribution
- SHA256 hash for Homebrew Cask formula

### Homebrew Cask Distribution

The Homebrew Cask formula is located at `homebrew/tod.rb`. To submit to Homebrew Cask:

1. Create a GitHub release with the DMG file
2. Update the SHA256 in `homebrew/tod.rb` (automatically done by `make release`)
3. Fork the [homebrew-cask](https://github.com/Homebrew/homebrew-cask) repository
4. Copy `homebrew/tod.rb` to `Casks/tod.rb`
5. Submit a pull request

## Architecture

- **TUI Models** (`internal/ui/`) - Bubble Tea interface components
- **Flow Engine** (`internal/engine/`) - Test flow state machine  
- **Email Integration** (`internal/email/`) - Magic link providers
- **Test Generation** (`internal/generator/`) - Session to test conversion

## Roadmap

- [ ] Magic link email polling
- [ ] Flow definition system
- [ ] Session recording/replay
- [ ] Test generation (Playwright/Cypress)
- [ ] Multiple email providers
- [ ] Code discovery and analysis

## License

MIT