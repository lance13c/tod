# Tod Development Setup

Quick setup guide for Tod development environment with hot reload.

## One-Line Setup

```bash
./dev-setup.sh
```

That's it! The script handles everything:
- ✅ Installs Go 1.24+ (required for air)
- ✅ Installs air hot reload tool  
- ✅ Sets up global `toddev` with auto-update
- ✅ Configures shell PATH automatically
- ✅ Creates development workflow

## What You Get

### Hot Reload Development
```bash
make hotdev    # Start hot reload - keep this running
# Now edit code and see instant updates!
```

### Global Access
```bash
cd ~/any/project
toddev init --non-interactive
toddev         # Always uses your latest development code!
```

### Development Commands
```bash
make hotdev          # Hot reload development (primary workflow)
make dev             # Quick run (go run .)
make build-dev       # Build toddev binary
make test-repo REPO=../my-project  # Test in specific repo
make help            # Show all commands
```

## How It Works

1. **`make hotdev`** runs air which watches files and rebuilds `./tmp/toddev`
2. **Global `toddev`** is symlinked to `./tmp/toddev` 
3. **Every code change** automatically updates the global `toddev`
4. **Use from anywhere** - `toddev` always has your latest changes

## Manual Setup (if needed)

If the script doesn't work, here's the manual process:

```bash
# 1. Install Go 1.24+
brew install go@1.24

# 2. Update PATH (add to ~/.zshrc or ~/.bashrc)
export PATH="/opt/homebrew/opt/go@1.24/bin:$PATH"
export PATH="$HOME/bin:$PATH"

# 3. Install air
go install github.com/air-verse/air@latest

# 4. Setup global toddev
make install-dev-link
```

## Troubleshooting

**Air not found?**
- Ensure Go 1.24+ is installed and in PATH
- Run: `go install github.com/air-verse/air@latest`

**toddev not global?**
- Check `~/bin` is in PATH: `echo $PATH | grep "$HOME/bin"`
- Restart terminal or run: `source ~/.zshrc`

**Hot reload not working?**
- Check if air is running: `make hotdev` should show file watching
- Verify `.air.toml` configuration exists

## File Structure

```
tod/
├── dev-setup.sh      # One-line setup script
├── .air.toml         # Air configuration  
├── Makefile          # Development commands
├── tmp/toddev        # Hot reload binary (created by air)
└── ~/bin/toddev      # Global symlink (created by setup)
```

## Development Workflow

1. **Start Hot Reload**: `make hotdev` (keep running)
2. **Edit Code**: Make changes to any `.go` file
3. **Test Globally**: Run `toddev` from any directory
4. **Instant Updates**: Changes apply immediately

Perfect for developing and testing Tod across different projects!