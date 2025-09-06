# Tod Release Process

This document describes the automated release process for Tod using the `release.sh` script.

## Overview

The release script automates the entire release workflow but pauses before submitting the PR to allow for review. It handles:

- Version management (automatic increment or custom)
- Building universal macOS binaries
- Creating DMG installers
- Uploading to GitHub releases
- Updating homebrew formulas
- Preparing homebrew-cask PR

## Prerequisites

### Required Tools
- `git` - Version control
- `gh` - GitHub CLI (authenticated)
- `go` - Go compiler 
- `make` - Build system
- `shasum` - SHA256 calculation
- `hdiutil` - macOS DMG creation
- `lipo` - Universal binary creation

### Setup
1. Ensure you're in the tod project root directory
2. Have a clean git working directory
3. GitHub CLI must be authenticated (`gh auth login`)
4. The `lance13c/tod-releases` repository must exist
5. The `lance13c/homebrew-cask` fork must exist

## Usage

### Basic Usage
```bash
# Interactive version selection
./release.sh

# Use specific version
./release.sh -v 1.2.3

# Dry run (test without making changes)
./release.sh --dry-run

# Show help
./release.sh --help
```

### Version Selection Options
When run interactively, you can choose:
1. **Patch increment** - `0.0.2` → `0.0.3` (bug fixes)
2. **Minor increment** - `0.0.2` → `0.1.0` (new features)
3. **Major increment** - `0.0.2` → `1.0.0` (breaking changes)
4. **Custom version** - Specify any valid semver

## Release Workflow

### 1. Pre-flight Checks
- ✅ Verifies all dependencies are installed
- ✅ Confirms running in tod project directory
- ✅ Checks git working directory is clean
- ✅ Gets current version from `homebrew/tod.rb`

### 2. Version Selection
- Interactive menu for version increment
- Or specify version via `-v` flag
- Validates semantic versioning format

### 3. Build Process
- Cleans previous build artifacts
- Builds AMD64 and ARM64 binaries
- Creates universal binary with `lipo`
- Builds macOS app bundle (`Tod.app`)
- Generates `Info.plist` with correct version
- Creates DMG installer
- Calculates SHA256 hash

### 4. GitHub Release
- Uploads DMG to `lance13c/tod-releases`
- Creates release with version tag
- Includes installation instructions

### 5. Formula Updates
- Updates local `homebrew/tod.rb` with new version/hash
- Clones/updates forked `homebrew-cask`
- Creates new branch `update-tod-VERSION`
- Updates `Casks/t/tod.rb` with new version/hash
- Commits and pushes changes

### 6. Review Checkpoint ⚠️
**The script PAUSES here for manual review:**
- Shows summary of all changes
- Lists updated files
- Prompts for confirmation before PR submission
- **This is your chance to test the DMG manually**

### 7. PR Submission
- Creates draft PR to `homebrew/homebrew-cask`
- Includes detailed description and checklist
- Provides PR URL for further review

## Manual Testing

Before confirming PR submission, you should:

1. **Test DMG Installation**
   ```bash
   # Download and test the DMG
   open dist/tod-0.x.x.dmg
   # Install Tod.app and verify it works
   ```

2. **Test Binary Access**
   ```bash
   # Verify binary is accessible after installation
   /Applications/Tod.app/Contents/MacOS/tod --version
   ```

3. **Test Homebrew Formula**
   ```bash
   # Test local formula (optional)
   brew install --cask ./homebrew/tod.rb
   ```

## File Structure

The script creates/modifies these files:
```
tod/
├── release.sh              # Main release script
├── RELEASE.md              # This documentation
├── dist/
│   ├── tod-VERSION.dmg     # DMG installer
│   ├── Tod.app/            # macOS app bundle
│   └── tod                 # Universal binary
├── homebrew/
│   └── tod.rb              # Local homebrew formula
└── homebrew-cask/          # Forked homebrew-cask
    └── Casks/t/tod.rb      # Official cask formula
```

## Troubleshooting

### Common Issues

**"No such file or directory: tod"**
- Run script from tod project root directory

**"Required dependency 'X' is not installed"**
- Install missing dependency (e.g., `brew install gh`)

**"Git working directory is not clean"**
- Commit or stash changes before running
- Or use `-f` to force (if added to script)

**"Release already exists"**
- Delete existing release: `gh release delete vX.X.X --repo lance13c/tod-releases`
- Or increment version number

**"Authentication failed"**
- Run `gh auth login` to authenticate GitHub CLI

### Recovery

If the script fails partway through:

1. **Build failure**: Fix build issues and re-run
2. **GitHub release failure**: Delete partial release and retry
3. **PR already exists**: Close existing PR or use different version

### Manual Override

To manually complete a failed release:

```bash
# Manual GitHub release
gh release create v0.0.3 dist/tod-0.0.3.dmg --repo lance13c/tod-releases

# Manual PR creation
cd homebrew-cask
gh pr create --repo homebrew/homebrew-cask --head lance13c:update-tod-0.0.3
```

## Examples

### Patch Release (Bug Fix)
```bash
./release.sh
# Select option 1 (patch increment)
# 0.0.2 → 0.0.3
```

### Minor Release (New Features)
```bash
./release.sh -v 0.1.0
# Direct version specification
```

### Major Release (Breaking Changes)
```bash
./release.sh -v 1.0.0
# Major version bump
```

### Test Release Process
```bash
./release.sh --dry-run
# Test without making actual changes
```

## Security Notes

- The script requires GitHub CLI authentication
- DMG files are signed by the build process (if certificates available)
- All releases are public in `lance13c/tod-releases`
- Source code remains private in `lance13c/tod`

## Contributing

To modify the release process:
1. Edit `release.sh`
2. Test with `--dry-run` flag
3. Update this documentation
4. Test with a patch version first

---

Tod says: "The Test God has blessed you with divine automation, nephew. Use it wisely!"