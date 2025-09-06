# Meet Tod Your Webdev Test God

Tod is a text-adventure style E2E testing tool that makes web testing interactive and fun. Navigate your web applications like a text adventure game, with natural language commands and AI-powered assistance.

## 🚀 Quick Start

### Prerequisites
- Go 1.24 (exact version required)
- For hot reload development: `go install github.com/air-verse/air@latest`

### Development Setup

```bash
cd tod

# Start hot reload development (auto-rebuilds as toddev)
make hotdev

# In another terminal, test in any repo
make test-repo REPO=../my-project
```

### Production Build

```bash
# Build production binary (tod)
make build

# Install globally
make install

# Build for distribution
make release
```

## 🎯 Usage

### Development Workflow
1. **Start hot reload**: `make hotdev` - builds `toddev` automatically on file changes
2. **Test anywhere**: `./dev-test.sh ../my-project` - test toddev in any repo
3. **Quick commands**: Use the Makefile targets for common tasks

### Initialize a Project
```bash
# Interactive setup (recommended)
toddev init

# Non-interactive (for CI/CD)  
toddev init --non-interactive
```

### Adventure Mode
```bash
# Start the text adventure interface
toddev
```

Navigate with natural language:
- `click login button`
- `fill email with user@example.com`
- `go to /dashboard`
- `help` for full command reference

## 🛠️ Development Commands

```bash
# 🔥 Hot reload development
make hotdev                    # Start air with auto-rebuild

# 🧪 Testing  
make test-repo REPO=../myapp  # Test in specific repo
./dev-test.sh ../my-project   # Direct script usage

# 📦 Building
make build-dev                # Build toddev binary
make build                    # Build production tod binary
make install-dev              # Install toddev globally

# 🧹 Maintenance
make clean                    # Clean all artifacts
make help                     # Show all targets
```

## 📁 Project Structure

```
tod/
├── .air.toml          # Hot reload config
├── dev-test.sh        # Development testing script
├── Makefile           # Build and development targets
├── cmd/               # CLI commands
├── internal/
│   ├── config/        # Configuration system
│   ├── ui/            # Terminal interface
│   │   └── views/     # Adventure mode views
│   ├── discovery/     # Code scanning
│   └── testing/       # Framework integration
└── tmp/               # Development builds (toddev)
```

## ⚙️ Configuration

Tod uses `.tod/config.yaml` with upward directory search:

```yaml
ai:
  provider: openai
  model: gpt-4-turbo
  api_key: ${TOD_AI_KEY}

testing:
  framework: playwright
  version: 1.40.0
  language: typescript
  test_dir: tests/e2e

environments:
  development:
    base_url: http://localhost:3000
  staging:
    base_url: https://staging.example.com

current_env: development
```

## 🎮 Features

- **Text Adventure Interface**: Navigate web apps like a game
- **Natural Language Commands**: Type conversational commands
- **Framework Agnostic**: Works with any E2E testing framework  
- **AI-Powered**: Multiple AI provider support
- **Session Recording**: All interactions saved for test generation
- **Hot Reload Development**: Instant feedback during development

## TODO
* General Navigation Between Pages ✅
* Scanning of code for actions ✅
* AI Provider Integration
* Session Recording & Test Generation
