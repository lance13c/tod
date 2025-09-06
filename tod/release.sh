#!/bin/bash

# Tod Release Automation Script
# Automates the entire release process but pauses before submitting PR for review

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
RELEASES_REPO="lance13c/tod-releases"
HOMEBREW_CASK_REPO="homebrew/homebrew-cask"
FORKED_CASK_REPO="lance13c/homebrew-cask"
TOD_HOMEPAGE="https://tod.dev/"

# Global variables
CURRENT_VERSION=""
NEW_VERSION=""
NEW_SHA256=""
BRANCH_NAME=""
DRY_RUN=false

# Utility functions
log() {
    echo -e "${GREEN}[Tod Release]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[Warning]${NC} $1"
}

error() {
    echo -e "${RED}[Error]${NC} $1"
    exit 1
}

info() {
    echo -e "${CYAN}[Info]${NC} $1"
}

tod_says() {
    echo -e "${PURPLE}[Tod]${NC} $1"
}

# Helper functions
confirm() {
    local prompt="$1"
    local response
    while true; do
        read -p "$prompt [y/N]: " response
        case $response in
            [Yy]* ) return 0;;
            [Nn]* | "" ) return 1;;
            * ) echo "Please answer yes or no.";;
        esac
    done
}

check_dependencies() {
    log "Checking dependencies..."
    
    local deps=("git" "gh" "go" "make" "shasum" "hdiutil" "lipo")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "Required dependency '$dep' is not installed"
        fi
    done
    
    # Check if we're in the tod directory
    if [[ ! -f "main.go" ]] || [[ ! -f "Makefile" ]]; then
        error "Please run this script from the tod project root directory"
    fi
    
    # Check if git is clean
    if [[ -n "$(git status --porcelain)" ]]; then
        warn "Git working directory is not clean"
        if ! confirm "Continue anyway?"; then
            exit 1
        fi
    fi
    
    log "Dependencies check passed ✅"
}

get_current_version() {
    # Get current version from homebrew/tod.rb
    if [[ -f "homebrew/tod.rb" ]]; then
        CURRENT_VERSION=$(grep 'version "' homebrew/tod.rb | sed 's/.*version "\([^"]*\)".*/\1/')
        log "Current version: $CURRENT_VERSION"
    else
        warn "No homebrew/tod.rb found, assuming this is the first release"
        CURRENT_VERSION="0.0.0"
    fi
}

increment_version() {
    local version="$1"
    local part="$2"  # major, minor, patch
    
    IFS='.' read -ra ADDR <<< "$version"
    local major="${ADDR[0]}"
    local minor="${ADDR[1]}"
    local patch="${ADDR[2]}"
    
    case "$part" in
        "major")
            echo "$((major + 1)).0.0"
            ;;
        "minor")
            echo "${major}.$((minor + 1)).0"
            ;;
        "patch")
            echo "${major}.${minor}.$((patch + 1))"
            ;;
        *)
            error "Invalid version part: $part"
            ;;
    esac
}

select_version() {
    log "Current version: $CURRENT_VERSION"
    echo ""
    echo "Select version increment:"
    echo "1) Patch (${CURRENT_VERSION} -> $(increment_version "$CURRENT_VERSION" "patch"))"
    echo "2) Minor (${CURRENT_VERSION} -> $(increment_version "$CURRENT_VERSION" "minor"))"
    echo "3) Major (${CURRENT_VERSION} -> $(increment_version "$CURRENT_VERSION" "major"))"
    echo "4) Custom version"
    echo ""
    
    while true; do
        read -p "Choose option [1-4]: " choice
        case $choice in
            1)
                NEW_VERSION=$(increment_version "$CURRENT_VERSION" "patch")
                break
                ;;
            2)
                NEW_VERSION=$(increment_version "$CURRENT_VERSION" "minor")
                break
                ;;
            3)
                NEW_VERSION=$(increment_version "$CURRENT_VERSION" "major")
                break
                ;;
            4)
                read -p "Enter custom version (e.g., 1.0.0): " NEW_VERSION
                if [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                    break
                else
                    error "Invalid version format. Please use semantic versioning (e.g., 1.0.0)"
                fi
                ;;
            *)
                echo "Invalid option. Please choose 1-4."
                ;;
        esac
    done
    
    log "Selected version: $NEW_VERSION"
}

build_release() {
    log "Building Tod v$NEW_VERSION..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would build version $NEW_VERSION"
        return 0
    fi
    
    # Clean previous builds
    make clean || true
    
    # Build the release
    log "Building universal binary..."
    GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$NEW_VERSION" -o dist/tod-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$NEW_VERSION" -o dist/tod-darwin-arm64 .
    lipo -create -output dist/tod dist/tod-darwin-amd64 dist/tod-darwin-arm64
    rm dist/tod-darwin-amd64 dist/tod-darwin-arm64
    
    # Create app bundle
    log "Creating app bundle..."
    mkdir -p dist/Tod.app/Contents/MacOS
    mkdir -p dist/Tod.app/Contents/Resources
    cp dist/tod dist/Tod.app/Contents/MacOS/tod
    chmod +x dist/Tod.app/Contents/MacOS/tod
    
    # Create Info.plist
    cat > dist/Tod.app/Contents/Info.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDisplayName</key>
	<string>Tod</string>
	<key>CFBundleExecutable</key>
	<string>tod</string>
	<key>CFBundleIdentifier</key>
	<string>com.ciciliostudio.tod</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>Tod</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$NEW_VERSION</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.developer-tools</string>
	<key>LSMinimumSystemVersion</key>
	<string>10.15</string>
</dict>
</plist>
EOF
    
    # Create DMG
    log "Creating DMG..."
    rm -f "dist/tod-$NEW_VERSION.dmg"
    hdiutil create -volname "Tod $NEW_VERSION" -srcfolder dist/Tod.app -ov -format UDZO "dist/tod-$NEW_VERSION.dmg"
    
    # Calculate SHA256
    NEW_SHA256=$(shasum -a 256 "dist/tod-$NEW_VERSION.dmg" | cut -d' ' -f1)
    log "SHA256: $NEW_SHA256"
    
    log "Build completed ✅"
}

create_github_release() {
    log "Creating GitHub release..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would create release v$NEW_VERSION on $RELEASES_REPO"
        return 0
    fi
    
    # Create release notes
    local release_notes="Tod v$NEW_VERSION

## Installation

\`\`\`bash
brew install --cask tod
\`\`\`

Or download the DMG directly and install manually.

---
Copyright (c) 2025 Ciciliostudio LLC. All rights reserved."
    
    # Create the release
    gh release create "v$NEW_VERSION" "dist/tod-$NEW_VERSION.dmg" \
        --repo "$RELEASES_REPO" \
        --title "Tod v$NEW_VERSION" \
        --notes "$release_notes"
    
    log "GitHub release created ✅"
}

update_local_formula() {
    log "Updating local homebrew formula..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would update homebrew/tod.rb"
        return 0
    fi
    
    # Update homebrew/tod.rb
    sed -i '' "s/version \".*\"/version \"$NEW_VERSION\"/" homebrew/tod.rb
    sed -i '' "s/sha256 \".*\"/sha256 \"$NEW_SHA256\"/" homebrew/tod.rb
    
    log "Local formula updated ✅"
}

update_homebrew_cask() {
    log "Preparing homebrew-cask update..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would update homebrew-cask formula"
        return 0
    fi
    
    # Create branch name
    BRANCH_NAME="update-tod-$NEW_VERSION"
    
    # Clone or update forked homebrew-cask
    if [[ ! -d "homebrew-cask" ]]; then
        log "Cloning forked homebrew-cask..."
        gh repo fork "$HOMEBREW_CASK_REPO" --clone=true || true
        cd homebrew-cask
    else
        log "Updating existing homebrew-cask clone..."
        cd homebrew-cask
        git fetch upstream
        git checkout master
        git merge upstream/master
    fi
    
    # Create new branch
    git checkout -b "$BRANCH_NAME"
    
    # Update the cask file
    log "Updating cask file..."
    sed -i '' "s/version \".*\"/version \"$NEW_VERSION\"/" Casks/t/tod.rb
    sed -i '' "s/sha256 \".*\"/sha256 \"$NEW_SHA256\"/" Casks/t/tod.rb
    
    # Commit changes
    git add Casks/t/tod.rb
    git commit -m "Update Tod to $NEW_VERSION

- Tod $NEW_VERSION with latest improvements
- Updated SHA256: $NEW_SHA256
- Release available at: https://github.com/$RELEASES_REPO/releases/tag/v$NEW_VERSION"
    
    # Push branch
    git push origin "$BRANCH_NAME"
    
    cd ..
    log "Homebrew-cask updated ✅"
}

show_review_summary() {
    echo ""
    echo "==============================================="
    tod_says "The Test God has prepared your release, nephew!"
    echo "==============================================="
    echo ""
    echo -e "${BLUE}Release Summary:${NC}"
    echo "  Version: $CURRENT_VERSION → $NEW_VERSION"
    echo "  SHA256: $NEW_SHA256"
    echo "  DMG: dist/tod-$NEW_VERSION.dmg"
    echo "  GitHub Release: https://github.com/$RELEASES_REPO/releases/tag/v$NEW_VERSION"
    echo "  Branch: $BRANCH_NAME"
    echo ""
    echo -e "${BLUE}Files Updated:${NC}"
    echo "  ✅ Local homebrew/tod.rb"
    echo "  ✅ Forked homebrew-cask Casks/t/tod.rb"
    echo "  ✅ GitHub release created"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "  1. Review the changes above"
    echo "  2. Test the DMG installation manually"
    echo "  3. Confirm you want to submit the PR"
    echo ""
}

submit_pr() {
    log "Submitting PR to homebrew-cask..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would submit PR to $HOMEBREW_CASK_REPO"
        return 0
    fi
    
    local pr_body="## Update: Tod to $NEW_VERSION

Tod is an agentic TUI manual tester - a text-adventure interface for E2E testing.

### Changes
- Updated to version $NEW_VERSION
- Updated SHA256 checksum
- Release available at: https://github.com/$RELEASES_REPO/releases/tag/v$NEW_VERSION

### Pre-submission checklist
- [x] The cask is for a stable version ($NEW_VERSION)
- [x] The cask file follows the template structure
- [x] The download URL is publicly accessible
- [x] The SHA256 checksum is correct ($NEW_SHA256)
- [x] The application installs and runs correctly

### Additional Notes
- Proprietary software from Ciciliostudio LLC
- Homepage: $TOD_HOMEPAGE
- Public releases hosted at: https://github.com/$RELEASES_REPO

Thank you for reviewing this update!"
    
    cd homebrew-cask
    local pr_url=$(gh pr create \
        --repo "$HOMEBREW_CASK_REPO" \
        --head "$FORKED_CASK_REPO:$BRANCH_NAME" \
        --title "Update Tod to $NEW_VERSION" \
        --body "$pr_body" \
        --draft)
    
    cd ..
    
    echo ""
    log "PR submitted as draft: $pr_url"
    echo ""
    tod_says "Your divine release is now in the hands of mortal reviewers!"
    echo ""
    echo -e "${GREEN}Next steps:${NC}"
    echo "1. Review the draft PR: $pr_url"
    echo "2. Mark as ready for review when satisfied"
    echo "3. Wait for homebrew maintainers to review"
    echo ""
}

cleanup() {
    log "Cleaning up..."
    # Add any cleanup logic here if needed
}

show_usage() {
    echo "Tod Release Automation Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v VERSION    Specify version explicitly (e.g., -v 1.0.0)"
    echo "  -d, --dry-run Perform a dry run without making changes"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                 # Interactive version selection"
    echo "  $0 -v 1.2.3       # Use specific version"
    echo "  $0 --dry-run       # Test the script without changes"
    echo ""
}

# Main execution
main() {
    local specified_version=""
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v)
                specified_version="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    # Banner
    echo ""
    echo "==============================================="
    tod_says "Welcome to Tod's divine release automation!"
    echo "==============================================="
    echo ""
    
    if [[ "$DRY_RUN" == "true" ]]; then
        warn "DRY RUN MODE - No changes will be made"
        echo ""
    fi
    
    # Execute release steps
    check_dependencies
    get_current_version
    
    if [[ -n "$specified_version" ]]; then
        NEW_VERSION="$specified_version"
        log "Using specified version: $NEW_VERSION"
    else
        select_version
    fi
    
    echo ""
    if confirm "Proceed with release v$NEW_VERSION?"; then
        build_release
        create_github_release
        update_local_formula
        update_homebrew_cask
        
        echo ""
        show_review_summary
        
        if confirm "Submit PR to homebrew-cask?"; then
            submit_pr
        else
            warn "PR not submitted. You can submit manually later:"
            echo "  cd homebrew-cask"
            echo "  gh pr create --repo $HOMEBREW_CASK_REPO --head $FORKED_CASK_REPO:$BRANCH_NAME"
        fi
    else
        log "Release cancelled"
        exit 0
    fi
    
    cleanup
    
    echo ""
    tod_says "Release process complete! May your software be bug-free and your tests divine!"
    echo ""
}

# Trap for cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"