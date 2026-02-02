#!/usr/bin/env bash
set -euo pipefail

# This script validates that Go and Python CSV patcher implementations produce equivalent output

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building Go tools..."
make csv-patcher fix-spec-descriptors >/dev/null

echo "Building bundle with Python..."
CSV_PATCHER_IMPL=python make bundle bundle-post-process >/dev/null
cp -r build/bundle build/bundle-python

echo "Building bundle with Go..."
rm -rf bundle build/bundle
CSV_PATCHER_IMPL=go make bundle bundle-post-process >/dev/null
cp -r build/bundle build/bundle-go

echo ""
echo "Comparing outputs..."

# Compare CSV files
PYTHON_CSV="build/bundle-python/manifests/rhacs-operator.clusterserviceversion.yaml"
GO_CSV="build/bundle-go/manifests/rhacs-operator.clusterserviceversion.yaml"

if ! diff -u "$PYTHON_CSV" "$GO_CSV"; then
    echo ""
    echo "ERROR: CSV files differ between Python and Go implementations"
    echo "Python: $PYTHON_CSV"
    echo "Go: $GO_CSV"
    echo ""
    echo "Validating both with operator-sdk..."

    # Validate both bundles
    echo "Validating Python bundle..."
    if .gotools/bin/operator-sdk bundle validate ./build/bundle-python --select-optional suite=operatorframework; then
        echo "✓ Python bundle is valid"
    else
        echo "✗ Python bundle validation failed"
    fi

    echo ""
    echo "Validating Go bundle..."
    if .gotools/bin/operator-sdk bundle validate ./build/bundle-go --select-optional suite=operatorframework; then
        echo "✓ Go bundle is valid"
    else
        echo "✗ Go bundle validation failed"
    fi

    exit 1
fi

echo "✓ CSV files are identical"

# Validate Go bundle
echo ""
echo "Validating Go bundle with operator-sdk..."
if .gotools/bin/operator-sdk bundle validate ./build/bundle-go --select-optional suite=operatorframework; then
    echo "✓ Go bundle is valid"
else
    echo "✗ Go bundle validation failed"
    exit 1
fi

echo ""
echo "SUCCESS: Go and Python implementations produce identical, valid output"

# Cleanup
rm -rf build/bundle-python build/bundle-go
