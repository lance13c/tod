#!/bin/bash

# Example usage of the Tod release script

echo "Tod Release Script Examples"
echo "=========================="
echo ""

echo "1. Interactive release (most common):"
echo "   ./release.sh"
echo ""

echo "2. Patch release (bug fix):"
echo "   ./release.sh -v 0.0.3"
echo ""

echo "3. Minor release (new features):"
echo "   ./release.sh -v 0.1.0"
echo ""

echo "4. Major release (breaking changes):"
echo "   ./release.sh -v 1.0.0"
echo ""

echo "5. Test the process (dry run):"
echo "   ./release.sh --dry-run"
echo ""

echo "6. Test with specific version:"
echo "   ./release.sh --dry-run -v 0.0.4"
echo ""

echo "7. Show help:"
echo "   ./release.sh --help"
echo ""

echo "Workflow Summary:"
echo "=================="
echo "1. Script checks dependencies and git status"
echo "2. You select version (or specify with -v)"
echo "3. Script builds DMG and uploads to GitHub"
echo "4. Script updates homebrew formulas"
echo "5. ⚠️  PAUSE - You review and test the release"
echo "6. You confirm PR submission"
echo "7. Draft PR created for homebrew-cask"
echo ""

echo "Tod says: 'Use the automation wisely, nephew!'"