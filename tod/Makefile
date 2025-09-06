.PHONY: dev hotdev build build-dev test install clean deps help test-repo test-cicilio install-dev

# Development with hot reload (builds as toddev)
hotdev: deps
	@echo "🔥 Starting hot reload development (toddev)..."
	@echo "📁 Builds to ./tmp/toddev"
	@if ! command -v air >/dev/null 2>&1; then \
		echo "❌ air not found."; \
		echo "💡 Install with: go install github.com/air-verse/air@latest"; \
		echo "📋 Alternatives:"; \
		echo "   • Use 'make watch' for simple file watching"; \
		echo "   • Use 'make dev' for manual reloads"; \
		exit 1; \
	fi
	air

# Simple file watching alternative to air (no external dependencies)
watch: deps
	@echo "👁️  Simple file watcher (manual restart on changes)"
	@echo "📁 Builds to ./toddev"
	@echo "🔄 Press Ctrl+C to stop"
	@while true; do \
		echo "🔨 Building toddev..."; \
		go build -o toddev . && echo "✅ Built toddev - running..."; \
		./toddev & \
		PID=$$!; \
		inotifywait -q -r -e modify --exclude='\.git|\.tod|dist|tmp|toddev$$' . 2>/dev/null || fswatch -o --exclude='\.git|\.tod|dist|tmp|toddev$$' . | head -1 >/dev/null 2>&1; \
		kill $$PID 2>/dev/null || true; \
		wait $$PID 2>/dev/null || true; \
		echo "🔄 File changed, restarting..."; \
		sleep 1; \
	done

# Default target - quick development run
dev: deps
	go run .

# Build the production binary
build: deps
	go build -o tod .

# Build development binary (toddev)
build-dev: deps
	go build -o toddev .

# Run tests
test:
	go test ./...

# Install production version to $GOPATH/bin
install: deps
	go install

# Install development version globally
install-dev: build-dev
	@echo "📦 Installing toddev globally..."
	mkdir -p ~/bin
	cp toddev ~/bin/toddev
	@if echo $$PATH | grep -q "$$HOME/bin"; then \
		echo "✅ toddev installed to ~/bin/toddev"; \
	else \
		echo "✅ toddev installed to ~/bin/toddev"; \
		echo "⚠️  Add ~/bin to PATH: echo 'export PATH=\"$$HOME/bin:$$PATH\"' >> ~/.zshrc"; \
	fi
	@echo "🔄 Run 'make update-dev' to update the global toddev with latest changes"

# Update the globally installed toddev with hot-reload changes
update-dev:
	@if [ -f ~/bin/toddev ]; then \
		echo "🔄 Updating global toddev..."; \
		cp ./tmp/toddev ~/bin/toddev 2>/dev/null || cp toddev ~/bin/toddev; \
		echo "✅ Global toddev updated"; \
	else \
		echo "❌ toddev not installed globally. Run 'make install-dev' first"; \
	fi

# Install development version with auto-update symlink
install-dev-link: build-dev
	@echo "📦 Creating symlink for toddev..."
	mkdir -p ~/bin
	ln -sf $(PWD)/tmp/toddev ~/bin/toddev
	@if echo $$PATH | grep -q "$$HOME/bin"; then \
		echo "✅ toddev symlinked to ~/bin/toddev (auto-updates with hotdev)"; \
	else \
		echo "✅ toddev symlinked to ~/bin/toddev (auto-updates with hotdev)"; \
		echo "⚠️  Add ~/bin to PATH: echo 'export PATH=\"$$HOME/bin:$$PATH\"' >> ~/.zshrc"; \
	fi

# Test in another repo with toddev (usage: make test-repo REPO=/path/to/repo)
test-repo: build-dev
	@if [ -z "$(REPO)" ]; then \
		echo "Usage: make test-repo REPO=/path/to/repo"; \
		echo "Example: make test-repo REPO=../ciciliostudio"; \
		exit 1; \
	fi
	@echo "🧪 Testing toddev in $(REPO)..."
	@mkdir -p tmp
	@cp toddev tmp/
	cd $(REPO) && $(PWD)/tmp/toddev init --non-interactive
	cd $(REPO) && $(PWD)/tmp/toddev

# Quick test in ciciliostudio
test-cicilio: build-dev
	@echo "🧪 Testing toddev in ../ciciliostudio..."
	@mkdir -p tmp
	@cp toddev tmp/
	@if [ ! -d "../ciciliostudio" ]; then \
		echo "❌ ../ciciliostudio directory not found"; \
		exit 1; \
	fi
	cd ../ciciliostudio && ../test-god/tod/tmp/toddev init --non-interactive
	cd ../ciciliostudio && ../test-god/tod/tmp/toddev

# Clean build artifacts
clean:
	rm -f ./tod toddev
	rm -rf tmp/
	go clean

# Download and tidy dependencies
deps:
	go mod download
	go mod tidy

# Run with race detection
dev-race: deps
	go run -race .

# Build for multiple platforms
build-all: deps
	GOOS=darwin GOARCH=amd64 go build -o dist/tod-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/tod-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -o dist/tod-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o dist/tod-windows-amd64.exe .

# Build universal binary for macOS
build-universal: deps dist
	GOOS=darwin GOARCH=amd64 go build -o dist/tod-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/tod-darwin-arm64 .
	lipo -create -output dist/tod dist/tod-darwin-amd64 dist/tod-darwin-arm64
	rm dist/tod-darwin-amd64 dist/tod-darwin-arm64

# Create macOS application bundle
build-app: build-universal
	mkdir -p dist/Tod.app/Contents/MacOS
	mkdir -p dist/Tod.app/Contents/Resources
	cp dist/tod dist/Tod.app/Contents/MacOS/tod
	chmod +x dist/Tod.app/Contents/MacOS/tod
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > dist/Tod.app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> dist/Tod.app/Contents/Info.plist
	@echo '<plist version="1.0">' >> dist/Tod.app/Contents/Info.plist
	@echo '<dict>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleDisplayName</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>Tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleExecutable</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleIdentifier</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>com.ciciliostudio.tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleInfoDictionaryVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>6.0</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleName</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>Tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundlePackageType</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>APPL</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleShortVersionString</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>0.0.1</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>1</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>LSApplicationCategoryType</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>public.app-category.developer-tools</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>LSMinimumSystemVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>10.15</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '</dict>' >> dist/Tod.app/Contents/Info.plist
	@echo '</plist>' >> dist/Tod.app/Contents/Info.plist

# Create DMG installer
build-dmg: build-app
	@if [ ! -f dist/Tod.app/Contents/MacOS/tod ]; then echo "Error: Tod.app not found"; exit 1; fi
	rm -f dist/tod-0.0.2.dmg
	hdiutil create -volname "Tod 0.0.2" -srcfolder dist/Tod.app -ov -format UDZO dist/tod-0.0.2.dmg

# Generate SHA256 for DMG
sha256: build-dmg
	shasum -a 256 dist/tod-0.0.2.dmg

# Full Homebrew Cask release process
release: build-dmg sha256
	@echo "Release artifacts created:"
	@echo "  DMG: dist/tod-0.0.2.dmg"
	@echo "  SHA256: $$(shasum -a 256 dist/tod-0.0.2.dmg | cut -d' ' -f1)"
	@echo ""
	@echo "Next steps for Homebrew Cask:"
	@echo "1. Upload dist/tod-0.0.2.dmg to GitHub Releases"
	@echo "2. Copy the SHA256 hash above"
	@echo "3. Update homebrew/tod.rb if needed"
	@echo "4. Submit PR to homebrew-cask repository"

# Create versioned release (VERSION=x.y.z make release-version)
release-version: VERSION ?= 0.0.1
release-version: clean deps dist
	@echo "Building release $(VERSION)..."
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/tod-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/tod-darwin-arm64 .
	lipo -create -output dist/tod dist/tod-darwin-amd64 dist/tod-darwin-arm64
	rm dist/tod-darwin-amd64 dist/tod-darwin-arm64
	mkdir -p dist/Tod.app/Contents/MacOS
	mkdir -p dist/Tod.app/Contents/Resources
	cp dist/tod dist/Tod.app/Contents/MacOS/tod
	chmod +x dist/Tod.app/Contents/MacOS/tod
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > dist/Tod.app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> dist/Tod.app/Contents/Info.plist
	@echo '<plist version="1.0">' >> dist/Tod.app/Contents/Info.plist
	@echo '<dict>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleDisplayName</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>Tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleExecutable</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleIdentifier</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>com.ciciliostudio.tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleInfoDictionaryVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>6.0</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleName</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>Tod</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundlePackageType</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>APPL</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleShortVersionString</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>$(VERSION)</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>CFBundleVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>1</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>LSApplicationCategoryType</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>public.app-category.developer-tools</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<key>LSMinimumSystemVersion</key>' >> dist/Tod.app/Contents/Info.plist
	@echo '	<string>10.15</string>' >> dist/Tod.app/Contents/Info.plist
	@echo '</dict>' >> dist/Tod.app/Contents/Info.plist
	@echo '</plist>' >> dist/Tod.app/Contents/Info.plist
	rm -f dist/tod-$(VERSION).dmg
	hdiutil create -volname "Tod $(VERSION)" -srcfolder dist/Tod.app -ov -format UDZO dist/tod-$(VERSION).dmg
	@echo "Release $(VERSION) created:"
	@echo "  DMG: dist/tod-$(VERSION).dmg"
	@echo "  SHA256: $$(shasum -a 256 dist/tod-$(VERSION).dmg | cut -d' ' -f1)"

# Create dist directory
dist:
	mkdir -p dist

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "🚀 Development:"
	@echo "  dev             - Run in development mode (go run)"
	@echo "  hotdev          - Hot reload development with air (builds toddev)"
	@echo "  build-dev       - Build development binary (toddev)"
	@echo "  install-dev     - Install toddev globally to ~/bin"
	@echo "  install-dev-link - Create symlink that auto-updates with hotdev"
	@echo "  update-dev      - Update global toddev with latest changes"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  test-cicilio    - Test toddev in ../ciciliostudio"
	@echo "  test-repo       - Test in specific repo (REPO=path)"
	@echo "  test            - Run Go tests"
	@echo ""
	@echo "📦 Production:"
	@echo "  build           - Build production binary (tod)"
	@echo "  install         - Install to \$$GOPATH/bin"
	@echo "  build-all       - Build for multiple platforms"
	@echo "  build-universal - Build universal binary for macOS"
	@echo "  build-app       - Create macOS application bundle"
	@echo "  build-dmg       - Create DMG installer"
	@echo "  release         - Full Homebrew Cask release process"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  dev-race        - Run with race detection"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "💡 Examples:"
	@echo "  make hotdev                    # Start hot reload dev"
	@echo "  make test-cicilio             # Test in ciciliostudio repo"
	@echo "  make test-repo REPO=../myapp  # Test in specific repo"