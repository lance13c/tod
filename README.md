# Tod aka Test gOD

A delightful CLI headless browsing and E2E test generation tool.

## Key Features

* Walk through webapp with only the cli
* Authenticate test users easily (WIP)
* Usable by agentic development tools like claude code (WIP)
* Test Generation (WIP)

## Installation

### For Users

#### Homebrew Cask (macOS) - Recommended

```bash
# Install via Homebrew Cask
brew tap lance13c/tod https://github.com/lance13c/tod && brew install --cask tod

# Run tod from anywhere
tod
```

#### Requirements

- macOS 10.15 (Catalina) or later
- Intel or Apple Silicon Mac

### For Developers

#### One-Line Development Setup

```bash
git clone https://github.com/lance13c/tod.git
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
git clone https://github.com/lance13c/tod.git
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

## Support

- Homepage: [https://tod.dev/](https://tod.dev/)
- Issues: [https://github.com/lance13c/tod/issues](https://github.com/lance13c/tod/issues)
