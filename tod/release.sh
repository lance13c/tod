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
RELEASES_REPO="lance13c/tod"
PUBLIC_REPO="lance13c/tod"
TOD_HOMEPAGE="https://tod.dev/"

# Global variables
CURRENT_VERSION=""
NEW_VERSION=""
NEW_SHA256=""
DRY_RUN=false
SIGNING_IDENTITY=""
NOTARIZATION_PROFILE=""

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

check_signing_identity() {
    log "Checking for code signing identity..."
    
    # Check for signing identity
    SIGNING_IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | awk -F'"' '{print $2}')
    if [[ -z "$SIGNING_IDENTITY" ]]; then
        warn "No Developer ID Application certificate found. App will not be signed."
        warn "Users will see a security warning when opening the app."
        if ! confirm "Continue without signing?"; then
            error "Aborting release. Please install a Developer ID Application certificate or continue without signing."
        fi
        SIGNING_IDENTITY=""
    else
        log "Using signing identity: $SIGNING_IDENTITY"
        
        # Check for notarization profile
        if security find-generic-password -s "notarytool-profile" &> /dev/null; then
            NOTARIZATION_PROFILE="notarytool-profile"
            log "Notarization profile found: $NOTARIZATION_PROFILE"
        else
            warn "No notarization profile found. App will be signed but not notarized."
            warn "Users may still see security warnings when opening the app."
        fi
    fi
}

notarize_app() {
    if [[ -z "$NOTARIZATION_PROFILE" ]]; then
        warn "Skipping notarization - no profile configured"
        return 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would notarize app bundle"
        return 0
    fi
    
    log "Submitting app for notarization..."
    
    # Create a zip file for notarization (required for app bundles)
    local zip_path="dist/tod-notarization.zip"
    (cd dist && zip -r "$(basename "$zip_path")" Tod.app)
    
    # Submit for notarization
    local submission_id=$(xcrun notarytool submit "$zip_path" \
        --keychain-profile "$NOTARIZATION_PROFILE" \
        --wait \
        --output-format json | jq -r '.id')
    
    if [[ -z "$submission_id" || "$submission_id" == "null" ]]; then
        warn "Failed to get submission ID from notarization service"
        return 1
    fi
    
    log "Notarization submission ID: $submission_id"
    
    # Check status (the --wait flag should handle this, but let's be safe)
    log "Waiting for notarization to complete..."
    local status=""
    local attempts=0
    local max_attempts=30  # 15 minutes max (30 * 30 seconds)
    
    while [[ "$attempts" -lt "$max_attempts" ]]; do
        status=$(xcrun notarytool info "$submission_id" \
            --keychain-profile "$NOTARIZATION_PROFILE" \
            --output-format json | jq -r '.status')
        
        case "$status" in
            "Accepted")
                log "Notarization successful! ✅"
                break
                ;;
            "Rejected")
                error "Notarization was rejected. Check the logs with: xcrun notarytool log $submission_id --keychain-profile $NOTARIZATION_PROFILE"
                ;;
            "Invalid")
                error "Notarization submission was invalid. Check the logs with: xcrun notarytool log $submission_id --keychain-profile $NOTARIZATION_PROFILE"
                ;;
            "In Progress")
                info "Notarization in progress... (attempt $((attempts + 1))/$max_attempts)"
                sleep 30
                ((attempts++))
                ;;
            *)
                warn "Unknown notarization status: $status"
                sleep 30
                ((attempts++))
                ;;
        esac
    done
    
    if [[ "$status" != "Accepted" ]]; then
        error "Notarization did not complete successfully within the timeout period"
    fi
    
    # Staple the notarization ticket
    log "Stapling notarization ticket..."
    if xcrun stapler staple dist/Tod.app; then
        log "Notarization ticket stapled successfully ✅"
    else
        warn "Failed to staple notarization ticket, but notarization was successful"
    fi
    
    # Clean up temporary zip file
    rm -f "$zip_path"
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
    
    # Sign the app bundle if we have a signing identity
    if [[ -n "$SIGNING_IDENTITY" ]]; then
        log "Signing app bundle..."
        codesign --force --deep --sign "$SIGNING_IDENTITY" --options runtime dist/Tod.app
        
        # Verify the signature
        log "Verifying signature..."
        if codesign --verify --verbose dist/Tod.app; then
            log "App bundle signed successfully ✅"
            
            # Notarize the app
            notarize_app
        else
            warn "Failed to verify signature. App may still work but will show security warnings."
        fi
    else
        warn "Skipping code signing - no identity available"
    fi
    
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

push_to_public_repo() {
    log "Pushing release to public repository..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        info "DRY RUN: Would push to public repository $PUBLIC_REPO"
        return 0
    fi
    
    # Check if we have the public repo as a remote
    if ! git remote | grep -q "public"; then
        log "Adding public repository as remote..."
        git remote add public "https://github.com/$PUBLIC_REPO.git"
    fi
    
    # Fetch latest from public repo
    git fetch public
    
    # Create a release branch based on current state
    local release_branch="release-v$NEW_VERSION"
    git checkout -b "$release_branch"
    
    # Add the updated homebrew formula
    git add homebrew/tod.rb
    
    # Commit the release
    git commit -m "Release Tod v$NEW_VERSION

- Updated version to $NEW_VERSION
- Updated homebrew formula with new SHA256: $NEW_SHA256
- Built and signed app bundle
- Ready for distribution"
    
    # Create and push tag
    git tag -a "v$NEW_VERSION" -m "Tod v$NEW_VERSION"
    
    # Push to public repository
    git push public "$release_branch"
    git push public "v$NEW_VERSION"
    
    # Switch back to original branch
    git checkout -
    
    log "Pushed to public repository ✅"
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
    echo "  Public Repository: $PUBLIC_REPO"
    echo ""
    echo -e "${BLUE}Files Updated:${NC}"
    echo "  ✅ Local homebrew/tod.rb"
    echo "  ✅ GitHub release created"
    echo "  ✅ Pushed to public repository"
    echo ""
    echo -e "${BLUE}Installation:${NC}"
    echo "  brew tap lance13c/tod"
    echo "  brew install --cask tod"
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
    check_signing_identity
    
    if [[ -n "$specified_version" ]]; then
        NEW_VERSION="$specified_version"
        log "Using specified version: $NEW_VERSION"
    else
        select_version
    fi
    
    echo ""
    if confirm "Proceed with release v$NEW_VERSION?"; then
        build_release
        update_local_formula
        push_to_public_repo
        create_github_release
        
        echo ""
        show_review_summary
        
        echo ""
        tod_says "Release complete! Your homebrew tap is ready for divine consumption!"
        echo ""
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