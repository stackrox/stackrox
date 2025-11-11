#!/usr/bin/env bash

# Compare binary artifacts built by Make vs Bazel
# Usage: ./compare-binaries.sh <make_binary> <bazel_binary>

set -euo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <make_binary> <bazel_binary>"
    echo "Example: $0 bin/linux_amd64/central bazel-bin/central/central_/central"
    exit 1
fi

MAKE_BIN="$1"
BAZEL_BIN="$2"

echo "======================================================================"
echo "Binary Comparison: Make vs Bazel"
echo "======================================================================"
echo ""

# Check if files exist
if [ ! -f "$MAKE_BIN" ]; then
    echo "ERROR: Make binary not found: $MAKE_BIN"
    exit 1
fi

if [ ! -f "$BAZEL_BIN" ]; then
    echo "ERROR: Bazel binary not found: $BAZEL_BIN"
    exit 1
fi

# File sizes
MAKE_SIZE=$(ls -lh "$MAKE_BIN" | awk '{print $5}')
BAZEL_SIZE=$(ls -lh "$BAZEL_BIN" | awk '{print $5}')
MAKE_SIZE_BYTES=$(stat -f%z "$MAKE_BIN" 2>/dev/null || stat -c%s "$MAKE_BIN")
BAZEL_SIZE_BYTES=$(stat -f%z "$BAZEL_BIN" 2>/dev/null || stat -c%s "$BAZEL_BIN")

echo "📊 File Sizes:"
echo "  Make:  $MAKE_SIZE ($MAKE_SIZE_BYTES bytes)"
echo "  Bazel: $BAZEL_SIZE ($BAZEL_SIZE_BYTES bytes)"

# Calculate size difference percentage
SIZE_DIFF=$((BAZEL_SIZE_BYTES - MAKE_SIZE_BYTES))
SIZE_DIFF_PCT=$(awk "BEGIN {printf \"%.2f\", ($SIZE_DIFF / $MAKE_SIZE_BYTES) * 100}")

if [ "$SIZE_DIFF" -gt 0 ]; then
    echo "  Difference: +$SIZE_DIFF_PCT% (Bazel is larger)"
elif [ "$SIZE_DIFF" -lt 0 ]; then
    echo "  Difference: $SIZE_DIFF_PCT% (Bazel is smaller)"
else
    echo "  Difference: Identical sizes ✅"
fi

# Check size difference threshold (±5%)
SIZE_DIFF_ABS=${SIZE_DIFF#-}
SIZE_THRESHOLD=$((MAKE_SIZE_BYTES * 5 / 100))
if [ "$SIZE_DIFF_ABS" -gt "$SIZE_THRESHOLD" ]; then
    echo "  ⚠️  WARNING: Size difference >5%, investigate further"
fi

echo ""

# Checksums
MAKE_SHA256=$(shasum -a 256 "$MAKE_BIN" | awk '{print $1}')
BAZEL_SHA256=$(shasum -a 256 "$BAZEL_BIN" | awk '{print $1}')

echo "🔐 SHA256 Checksums:"
echo "  Make:  $MAKE_SHA256"
echo "  Bazel: $BAZEL_SHA256"

if [ "$MAKE_SHA256" = "$BAZEL_SHA256" ]; then
    echo "  ✅ Binaries are IDENTICAL"
    echo ""
    echo "======================================================================"
    echo "✅ PASS: Binaries are byte-for-byte identical"
    echo "======================================================================"
    exit 0
else
    echo "  ℹ️  Binaries differ (expected due to build paths/timestamps)"
fi

echo ""

# File type
echo "📝 File Type:"
file "$MAKE_BIN" | sed 's/^/  Make:  /'
file "$BAZEL_BIN" | sed 's/^/  Bazel: /'
echo ""

# Dynamic libraries (if applicable)
if command -v ldd >/dev/null 2>&1; then
    echo "📚 Dynamic Libraries (ldd):"
    echo "  Make:"
    ldd "$MAKE_BIN" 2>/dev/null | head -10 | sed 's/^/    /' || echo "    (static binary or ldd not applicable)"
    echo "  Bazel:"
    ldd "$BAZEL_BIN" 2>/dev/null | head -10 | sed 's/^/    /' || echo "    (static binary or ldd not applicable)"
elif command -v otool >/dev/null 2>&1; then
    echo "📚 Dynamic Libraries (otool -L):"
    echo "  Make:"
    otool -L "$MAKE_BIN" 2>/dev/null | tail -n +2 | sed 's/^/    /' || echo "    (static binary)"
    echo "  Bazel:"
    otool -L "$BAZEL_BIN" 2>/dev/null | tail -n +2 | sed 's/^/    /' || echo "    (static binary)"
fi

echo ""

# Try to run version command if binary supports it
echo "🔧 Version Check (if supported):"
if "$MAKE_BIN" --version >/dev/null 2>&1; then
    MAKE_VERSION=$("$MAKE_BIN" --version 2>&1 | head -5)
    echo "  Make binary version:"
    echo "$MAKE_VERSION" | sed 's/^/    /'
else
    echo "  Make binary: No --version flag"
fi

if "$BAZEL_BIN" --version >/dev/null 2>&1; then
    BAZEL_VERSION=$("$BAZEL_BIN" --version 2>&1 | head -5)
    echo "  Bazel binary version:"
    echo "$BAZEL_VERSION" | sed 's/^/    /'
else
    echo "  Bazel binary: No --version flag"
fi

echo ""

# Strip and compare (removes timestamps and build paths)
if command -v strip >/dev/null 2>&1; then
    echo "🔪 Comparing stripped binaries (removes debug info/paths)..."
    TEMP_DIR=$(mktemp -d)
    cp "$MAKE_BIN" "$TEMP_DIR/make_stripped"
    cp "$BAZEL_BIN" "$TEMP_DIR/bazel_stripped"
    
    strip "$TEMP_DIR/make_stripped" 2>/dev/null || true
    strip "$TEMP_DIR/bazel_stripped" 2>/dev/null || true
    
    MAKE_STRIPPED_SHA=$(shasum -a 256 "$TEMP_DIR/make_stripped" | awk '{print $1}')
    BAZEL_STRIPPED_SHA=$(shasum -a 256 "$TEMP_DIR/bazel_stripped" | awk '{print $1}')
    
    echo "  Stripped Make:  $MAKE_STRIPPED_SHA"
    echo "  Stripped Bazel: $BAZEL_STRIPPED_SHA"
    
    if [ "$MAKE_STRIPPED_SHA" = "$BAZEL_STRIPPED_SHA" ]; then
        echo "  ✅ Stripped binaries are IDENTICAL"
        rm -rf "$TEMP_DIR"
        echo ""
        echo "======================================================================"
        echo "✅ PASS: Binaries are functionally equivalent (stripped comparison)"
        echo "======================================================================"
        exit 0
    else
        echo "  ℹ️  Stripped binaries still differ (may have different compiler flags)"
    fi
    
    rm -rf "$TEMP_DIR"
fi

echo ""
echo "======================================================================"
echo "⚠️  REVIEW NEEDED: Binaries differ"
echo "======================================================================"
echo ""
echo "This is often expected due to:"
echo "  - Build timestamps"
echo "  - Build paths embedded in binaries"
echo "  - Different compiler optimization flags"
echo "  - Different Go build cache states"
echo ""
echo "Next steps:"
echo "  1. Check binary sizes (should be within ±5%)"
echo "  2. Run functional tests on both binaries"
echo "  3. Compare runtime behavior"
echo ""

exit 0

