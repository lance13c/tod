# Tod Quick Start

## New Developer Setup (30 seconds)

```bash
git clone https://github.com/ciciliostudio/tod.git
cd tod
./dev-setup.sh
```

## Development Workflow

```bash
# Terminal 1: Keep this running
make hotdev

# Terminal 2: Use toddev anywhere
cd ~/any/project
toddev init --non-interactive
toddev
```

## Key Commands

| Command | Purpose |
|---------|---------|
| `./dev-setup.sh` | One-time setup for new developers |
| `make hotdev` | Start hot reload (keep running) |
| `toddev` | Run Tod from anywhere (auto-updates) |
| `make help` | Show all available commands |

## How It Works

1. **`make hotdev`** watches your code and rebuilds `./tmp/toddev`
2. **Global `toddev`** symlinks to `./tmp/toddev` 
3. **Every change** automatically updates the global binary
4. **Test instantly** from any directory

## Troubleshooting

**Setup fails?** Check you have Homebrew installed
**toddev not found?** Restart terminal or run `source ~/.zshrc`  
**Hot reload not working?** Ensure `make hotdev` is running

---

**That's it!** You now have a complete Tod development environment with hot reload and global access.