#!/usr/bin/env bash

# Fix scanner external repository BUILD files to reference main workspace correctly
# This resolves the circular dependency issue with scanner importing rox packages

set -euo pipefail

BAZEL_OUTPUT_BASE=$(cd /Users/gualvare/sw/stackrox-2 && bazelisk info output_base 2>/dev/null)
SCANNER_EXTERNAL="$BAZEL_OUTPUT_BASE/external/gazelle~~go_deps~com_github_stackrox_scanner"

if [ ! -d "$SCANNER_EXTERNAL" ]; then
    echo "Scanner external repository not found at: $SCANNER_EXTERNAL"
    echo "Run a build first to fetch the scanner repository"
    exit 1
fi

echo "======================================================================"
echo "Fixing Scanner BUILD Files"
echo "======================================================================"
echo "Scanner repo: $SCANNER_EXTERNAL"
echo ""

# Find all BUILD files that reference stackrox_rox
BUILD_FILES=$(find "$SCANNER_EXTERNAL" -name "BUILD.bazel" -exec grep -l "stackrox_rox\|github.com/stackrox/rox" {} \;)

if [ -z "$BUILD_FILES" ]; then
    echo "No BUILD files found with rox references"
    exit 0
fi

echo "Found BUILD files with rox references:"
echo "$BUILD_FILES" | sed 's|.*/external/gazelle~~go_deps~com_github_stackrox_scanner/||'
echo ""

# Replace @stackrox_rox// with @// (references main workspace from extension)
for file in $BUILD_FILES; do
    echo "Fixing: $(basename $(dirname $file))/$(basename $file)"
    
    # In Bzlmod, @ from an extension repo should reference main workspace
    sed -i.bak 's|@stackrox_rox//|@//|g' "$file"
    sed -i.bak 's|@@stackrox_rox//|@//|g' "$file"
    
    # Alternative: use canonical name
    # sed -i.bak 's|@stackrox_rox//|@@//|g' "$file"
done

echo ""
echo "✅ Fixed all scanner BUILD files"
echo ""
echo "Removed backup files..."
find "$SCANNER_EXTERNAL" -name "*.bak" -delete

echo "Done!"

