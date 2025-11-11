#!/usr/bin/env bash

# Basic smoke test for a binary
# Usage: ./smoke-test.sh <binary_path> [test_type]

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <binary_path> [test_type]"
    echo "  test_type: basic (default), version, help"
    exit 1
fi

BINARY="$1"
TEST_TYPE="${2:-basic}"

if [ ! -f "$BINARY" ]; then
    echo "ERROR: Binary not found: $BINARY"
    exit 1
fi

if [ ! -x "$BINARY" ]; then
    echo "ERROR: Binary is not executable: $BINARY"
    exit 1
fi

BINARY_NAME=$(basename "$BINARY")
echo "======================================================================"
echo "Smoke Test: $BINARY_NAME"
echo "======================================================================"
echo ""

case "$TEST_TYPE" in
    basic)
        echo "✅ Binary exists and is executable"
        
        # Try to get version
        if "$BINARY" --version >/dev/null 2>&1; then
            echo "✅ Binary responds to --version flag"
            VERSION=$("$BINARY" --version 2>&1 | head -1)
            echo "   Version: $VERSION"
        elif "$BINARY" version >/dev/null 2>&1; then
            echo "✅ Binary responds to 'version' subcommand"
            VERSION=$("$BINARY" version 2>&1 | head -1)
            echo "   Version: $VERSION"
        else
            echo "ℹ️  Binary doesn't support version command"
        fi
        
        # Try to get help
        if "$BINARY" --help >/dev/null 2>&1; then
            echo "✅ Binary responds to --help flag"
        elif "$BINARY" -h >/dev/null 2>&1; then
            echo "✅ Binary responds to -h flag"
        else
            echo "ℹ️  Binary doesn't support help flags"
        fi
        ;;
        
    version)
        echo "Testing version command..."
        if "$BINARY" --version 2>&1; then
            echo "✅ PASS"
        elif "$BINARY" version 2>&1; then
            echo "✅ PASS"
        else
            echo "❌ FAIL: Version command not supported"
            exit 1
        fi
        ;;
        
    help)
        echo "Testing help command..."
        if "$BINARY" --help 2>&1 | head -20; then
            echo "✅ PASS"
        elif "$BINARY" -h 2>&1 | head -20; then
            echo "✅ PASS"
        else
            echo "❌ FAIL: Help command not supported"
            exit 1
        fi
        ;;
        
    *)
        echo "ERROR: Unknown test type: $TEST_TYPE"
        exit 1
        ;;
esac

echo ""
echo "======================================================================"
echo "✅ Smoke test passed"
echo "======================================================================"

