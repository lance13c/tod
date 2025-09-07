# Test God - Tod - Agentic Manual Tester

## Year
We are in the year 2025.
Never say best practices 2024, always say best practices 2025 if you this comes up.

## Truth Seeking
You actually check what I say and find the truth! No sycophantic behavior. You must assume I might be wrong. I might have a good point. You need to think about it.

## Overview

An agentic TUI that act as a developer's personal agent of beautiful software where UX and art come first. We prioritize emotion, attention to detail, and user experience over pure functionality. This monorepo contains multiple applications that share a unified backend and database.

## Claude
- When you are conducting reviews, there might be nothing wrong and that is okay, there also might be improvements you see. The user (me), is not always right and you might be wrong. Take everything with a grain of salt, be curious, looks up docs, and ask questions. We both might be wrong.
- Avoid writing technical implementation into the specs, and instead keep it in english if possible.

## Development Server
- NEVER run dev server, always assume it is running

## Development Workflow
- After ui edits to the desktop app check the tarui app console logs to see if there are any errors, if so fix them please

## URL and Environment Configuration
- **NEVER use fallback URLs** - always require proper environment variables
- **Bad practice**: `const url = process.env.API_URL || 'http://localhost:3000'`
- **Good practice**: `const url = process.env.API_URL` with proper error handling if missing
- Environment variables should be explicitly set and validated, not assumed with defaults

## Deployment Rules
- NEVER deploy to production unless explicitly instructed
- Always ask before deploying to production
- Staging deployments can be done without asking

## Philosophy

**Core Values:**
- Beautiful software where UX and art come first
- Attention to detail is paramount

## Icon Management
- We often have issues with icons, ensure we add icons to the icon map and relevant files when we add or update them

## Circular Dependency Detection

### Tool: dpdm (Detect Project Dependencies Map)
We use `dpdm` to detect circular dependencies in our TypeScript/JavaScript codebase. This tool is preferred over alternatives like `madge` because:
- Uses TypeScript AST for accurate parsing
- Better handles mixed JS/TS codebases  
- Provides clear circular dependency chain output
- Fast performance even on large projects

### Usage
After making code changes and running lint, check for circular dependencies:
```bash
# Check entire monorepo for circular dependencies
pnpm circular:check

# Run lint and circular dependency check together
pnpm lint:all
```

### Understanding the Output
When circular dependencies are found, dpdm will show:
- The files involved in the circular dependency chain
- The import path that creates the cycle
- Exit with code 1 to fail CI/CD pipelines

Example output:
```
✖ Circular dependency found: 
  src/services/auth.ts → src/contexts/AuthContext.tsx → src/services/auth.ts
```

### Fixing Circular Dependencies
Common strategies:
1. **Extract shared types/interfaces** to a separate file
2. **Use dependency injection** instead of direct imports
3. **Lazy load modules** with dynamic imports
4. **Refactor to remove bidirectional dependencies**

### Important
- Always run `pnpm circular:check` after `pnpm lint`
- The check covers both `/apps` and `/packages` directories
- Circular dependencies will cause build failures and runtime errors

### NextJS
- Avoid use nextjs API, always use the apps/api

## Blog Writing Guidelines

### Numbered List Formatting
When writing blog content with numbered lists, avoid using standard markdown numbered lists (`1. **Item**`) as they can have rendering issues. Instead, use this format:

**✅ Correct format:**
```markdown
**1. Item title**  
Description text goes here.

**2. Another item**  
More description text.
```

**❌ Avoid:**
```markdown
1. **Item title**
   Description text goes here.

2. **Another item**
   More description text.
```

**Why:** Standard markdown numbered lists can cause rendering issues where numbers appear on separate lines from headings, or text runs together on one line. The bold paragraph format ensures consistent display with numbers inline with headings and proper line breaks.

### General Writing Style
- Use "collabs" for informal references to collaborations (industry standard)
- Avoid excessive hyphenation that sounds AI-generated
- Keep bullet points concise with consistent formatting
- Use tables with proper alignment (left for text, right for numbers)
- avoid using emojis, you can use cool ascii codes, no emojis please

## Tod Release Process

### Prerequisites (One-time Setup)

**Apple Developer Certificates:**
- Developer ID Application certificate installed in Keychain
- Team ID: `745D23AJ53`

**Notarization Setup:**
```bash
# Store notarization credentials (already configured)
xcrun notarytool store-credentials "notarytool-profile" \
  --apple-id "dominic@ciciliostudio.com" \
  --team-id "745D23AJ53" \
  --password "[app-specific-password]"
```

**Credential Storage:**
- Notarization credentials are stored securely in macOS Keychain
- Keychain profile name: `notarytool-profile`
- Never commit raw credentials to git - they should only exist in keychain and environment variables

### Notarization Setup (One-time)

**Required Credentials:**
- **Apple ID**: dominic@ciciliostudio.com
- **Team ID**: 745D23AJ53 (Ciciliostudio LLC)
- **App-Specific Password**: Generate from [appleid.apple.com](https://appleid.apple.com) → Sign-In and Security → App-Specific Passwords

**Store Credentials in Keychain:**
```bash
xcrun notarytool store-credentials "notarytool-profile" \
  --apple-id "dominic@ciciliostudio.com" \
  --team-id "745D23AJ53" \
  --password "[app-specific-password]"
```

**Verify Setup:**
```bash
# Check if profile exists
security find-generic-password -s "notarytool-profile"

# Verify credentials work
xcrun notarytool history --keychain-profile "notarytool-profile"
```

### Release Workflow

**1. Enhanced Release Script**
The `tod/release.sh` script has been updated with:
- Apple notarization support via `xcrun notarytool`
- Hardened runtime code signing (`--options runtime`)
- Public repository sync (lance13c/tod)
- Homebrew tap updates

**2. Execute Release**
```bash
cd tod/
./release.sh
# Select version increment or provide custom version
# Script handles: build → sign → notarize → DMG creation → GitHub release
```

**3. Repository Structure**
- **Private repo**: Contains source code, stays private
- **Public repo** (`lance13c/tod`): Contains only release artifacts:
  - `README.md`
  - `LICENSE`
  - `HOMEBREW_TAP.md`
  - `Casks/tod.rb` (homebrew formula)

**4. What the Script Does**
- Builds universal binary (Intel + Apple Silicon)
- Code signs with Developer ID + hardened runtime
- Submits to Apple for notarization
- Waits for notarization approval
- Staples notarization ticket to app
- Creates DMG with notarized app
- Creates GitHub release on public repo
- Updates homebrew formula with new SHA256

### Notarization Process

**How It Works:**
1. App signed with `--options runtime` (hardened runtime)
2. Zip created for submission: `tod-notarization.zip`
3. Submitted via `xcrun notarytool submit`
4. Apple processes (usually 1-5 minutes)
5. Ticket stapled to app: `xcrun stapler staple`
6. Result: "Notarized Developer ID" status

**Verification:**
```bash
# Check notarization status
spctl -a -vvv -t install dist/Tod.app
# Should show: "source=Notarized Developer ID"
```

### User Installation

**Via Homebrew Tap:**
```bash
brew tap lance13c/tod
brew install --cask tod
```

**Manual Download:**
- Users download DMG from GitHub releases
- No security warnings due to proper notarization

### Common Issues

**"Unnotarized Developer ID" Error:**
- Missing hardened runtime flag
- Solution: Re-sign with `--options runtime`

**"Archive contains critical validation errors":**
- Usually hardened runtime not enabled
- Check logs: `xcrun notarytool log [submission-id]`

**Release Branch Contains Private Code:**
- Never push `release-v*` branches to public repo
- Only update homebrew formula and create GitHub releases

**Notarization Timeout:**
- Apple service can be slow
- Script waits up to 15 minutes automatically

### Manual Notarization Process

**If automatic notarization fails during release:**

1. **Create zip for notarization:**
   ```bash
   cd dist && zip -r tod-notarization.zip Tod.app
   ```

2. **Submit to Apple:**
   ```bash
   xcrun notarytool submit tod-notarization.zip \
     --keychain-profile "notarytool-profile" \
     --wait
   ```

3. **Staple notarization ticket:**
   ```bash
   xcrun stapler staple Tod.app
   ```

4. **Verify notarization:**
   ```bash
   spctl -a -vvv -t install Tod.app
   # Should show: "source=Notarized Developer ID"
   ```

5. **Recreate DMG with notarized app:**
   ```bash
   rm tod-X.X.X.dmg
   hdiutil create -volname "Tod X.X.X" -srcfolder Tod.app -ov -format UDZO tod-X.X.X.dmg
   ```

6. **Calculate new SHA256:**
   ```bash
   shasum -a 256 tod-X.X.X.dmg
   ```

7. **Update homebrew formula:**
   - Edit `homebrew/tod.rb` with new SHA256
   - Commit and push to public repo

8. **Re-upload to GitHub release:**
   ```bash
   gh release upload vX.X.X dist/tod-X.X.X.dmg --repo lance13c/tod --clobber
   ```

### Common Issues

**"Unnotarized Developer ID" Error:**
- Missing notarization profile in keychain
- Run setup commands above to store credentials

**"Archive contains critical validation errors":**
- Check hardened runtime is enabled: `--options runtime`
- View detailed logs: `xcrun notarytool log [submission-id] --keychain-profile "notarytool-profile"`

**Homebrew Formula SHA256 Mismatch:**
- Re-notarization changes the DMG SHA256
- Always recalculate and update formula after notarization

### Security Notes

- Private source code never exposed to public
- Release artifacts are minimal and safe
- Notarization ensures no malware false positives
- Code signing prevents tampering