#!/bin/bash
# Test script for Terminal Intelligence
# Usage:
#   ./test.sh           - Run all tests (fast mode)
#   ./test.sh --full    - Run all tests including slow property-based tests
#   ./test.sh <package> - Run tests for specific package

set -e

PACKAGE="${1:-./...}"
FULL=false

# Parse arguments
for arg in "$@"; do
    case $arg in
        --full)
            FULL=true
            shift
            ;;
        *)
            PACKAGE="$arg"
            ;;
    esac
done

echo -e "\033[36mRunning Terminal Intelligence Tests\033[0m"
echo -e "\033[36m=====================================\033[0m"
echo ""

if [ "$FULL" = true ]; then
    echo -e "\033[33mRunning FULL test suite (including slow property-based tests)...\033[0m"
    go test "$PACKAGE" -v -timeout 5m
else
    echo -e "\033[32mRunning FAST test suite (skipping slow property-based tests)...\033[0m"
    echo -e "\033[90mUse --full flag to run complete test suite\033[0m"
    echo ""
    go test "$PACKAGE" -short -v
fi

if [ $? -eq 0 ]; then
    echo ""
    echo -e "\033[32m✓ All tests passed!\033[0m"
else
    echo ""
    echo -e "\033[31m✗ Some tests failed\033[0m"
    exit 1
fi
