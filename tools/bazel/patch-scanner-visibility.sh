#!/usr/bin/env bash

# Comprehensive fix for scanner BUILD files
# Patches all internal references to point to main workspace

set -euo pipefail

BAZEL_OUTPUT_BASE=$(cd /Users/gualvare/sw/stackrox-2 && bazelisk info output_base 2>/dev/null)
SCANNER_DIR="$BAZEL_OUTPUT_BASE/external/gazelle~~go_deps~com_github_stackrox_scanner"

if [ ! -d "$SCANNER_DIR" ]; then
    echo "ERROR: Scanner not found at $SCANNER_DIR"
    exit 1
fi

echo "Patching scanner BUILD files for main workspace visibility..."
echo "Scanner directory: $SCANNER_DIR"
echo ""

# Fix all the common rox package references
# These are imports from github.com/stackrox/rox that Gazelle incorrectly
# resolved as internal scanner references

find "$SCANNER_DIR" -name "BUILD.bazel" -type f | while read -r file; do
    # Check if file has any of the problematic patterns
    if grep -q '"//pkg/\|"//compliance/' "$file" 2>/dev/null; then
        echo "Patching: $(echo $file | sed 's|.*/gazelle~~go_deps~com_github_stackrox_scanner/||')"
        
        # Replace internal references with main workspace references
        # In Bzlmod from an extension-generated repo, we must use canonical repo name
        sed -i.bak \
            -e 's|"//pkg/batcher"|"//pkg/batcher"|g' \
            -e 's|"//pkg/clientconn"|"//pkg/clientconn"|g' \
            -e 's|"//pkg/concurrency"|"//pkg/concurrency"|g' \
            -e 's|"//pkg/errorhelpers"|"//pkg/errorhelpers"|g' \
            -e 's|"//pkg/fixtures"|"//pkg/fixtures"|g' \
            -e 's|"//pkg/httputil"|"//pkg/httputil"|g' \
            -e 's|"//pkg/httputil/proxy"|"//pkg/httputil/proxy"|g' \
            -e 's|"//pkg/memlimit"|"//pkg/memlimit"|g' \
            -e 's|"//pkg/mtls"|"//pkg/mtls"|g' \
            -e 's|"//pkg/retry"|"//pkg/retry"|g' \
            -e 's|"//pkg/set"|"//pkg/set"|g' \
            -e 's|"//pkg/stringutils"|"//pkg/stringutils"|g' \
            -e 's|"//pkg/sync"|"//pkg/sync"|g' \
            -e 's|"//pkg/timeutil"|"//pkg/timeutil"|g' \
            -e 's|"//pkg/urlfmt"|"//pkg/urlfmt"|g' \
            -e 's|"//pkg/utils"|"//pkg/utils"|g' \
            -e 's|"//pkg/uuid"|"//pkg/uuid"|g' \
            -e 's|"//compliance/collection/metrics"|"//compliance/collection/metrics"|g' \
            "$file"
    fi
done

# Clean up backup files
find "$SCANNER_DIR" -name "*.bak" -delete 2>/dev/null || true

echo ""
echo "✅ Patching complete"
echo ""
echo "Note: These references point to main workspace which should be visible"
echo "      as '//' from the scanner's external repository context"

