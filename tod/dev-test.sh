#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default to ciciliostudio or accept path as argument
TEST_REPO=${1:-"../ciciliostudio"}

echo -e "${BLUE}🏗️  Building toddev...${NC}"
go build -o ./tmp/toddev .

echo -e "${BLUE}📁 Testing in: $TEST_REPO${NC}"

# Check if repo exists
if [ ! -d "$TEST_REPO" ]; then
    echo -e "${RED}❌ Repository not found: $TEST_REPO${NC}"
    echo -e "${YELLOW}Usage: ./dev-test.sh [path-to-repo]${NC}"
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "  ./dev-test.sh                    # Test in ../ciciliostudio"
    echo -e "  ./dev-test.sh ../my-other-repo   # Test in specific repo"
    echo -e "  ./dev-test.sh ~/projects/myapp   # Test with absolute path"
    exit 1
fi

# Get absolute paths
TODDEV_PATH=$(pwd)/tmp/toddev
cd "$TEST_REPO"

echo -e "${GREEN}⚙️  Initializing toddev in $(pwd)...${NC}"
echo -e "${YELLOW}Running: $TODDEV_PATH init --non-interactive${NC}"
$TODDEV_PATH init --non-interactive

echo ""
echo -e "${GREEN}🎯 Starting toddev adventure mode...${NC}"
echo -e "${YELLOW}Press Ctrl+C to exit${NC}"
echo ""
$TODDEV_PATH