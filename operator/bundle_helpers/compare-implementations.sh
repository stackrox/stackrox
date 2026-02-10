#!/usr/bin/env bash
# Compare Python vs Go bundle helper outputs locally
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPERATOR_DIR="$(dirname "$SCRIPT_DIR")"

RELATED_IMAGES_MODE="${RELATED_IMAGES_MODE:-omit}"
WORK_DIR="${WORK_DIR:-/tmp/bundle-compare-$$}"

echo "=== Comparing bundle implementations ==="
echo "Related images mode: $RELATED_IMAGES_MODE"
echo "Work directory: $WORK_DIR"

cd "$OPERATOR_DIR"

# Use python3.10 which is known to work
export PYTHON=python3.10

# Build with Python
echo ""
echo "=== Building with Python implementation..."
USE_GO_BUNDLE_HELPER=false RELATED_IMAGES_MODE="$RELATED_IMAGES_MODE" make bundle bundle-post-process
mkdir -p "$WORK_DIR/python" "$WORK_DIR/python-build"
cp -r bundle "$WORK_DIR/python/"
[ -d build/bundle ] && cp -r build/bundle "$WORK_DIR/python-build/"

# Clean
echo ""
echo "=== Cleaning..."
git restore bundle/ build/ || true
git clean -fd bundle/ build/ || true

# Build with Go
echo ""
echo "=== Building with Go implementation..."
USE_GO_BUNDLE_HELPER=true RELATED_IMAGES_MODE="$RELATED_IMAGES_MODE" make bundle bundle-post-process
mkdir -p "$WORK_DIR/go" "$WORK_DIR/go-build"
cp -r bundle "$WORK_DIR/go/"
[ -d build/bundle ] && cp -r build/bundle "$WORK_DIR/go-build/"

# Compare
echo ""
echo "=== Comparing bundle/ outputs..."
if diff -ruN "$WORK_DIR/python/bundle" "$WORK_DIR/go/bundle"; then
  echo "✓ bundle/ outputs are identical"
  BUNDLE_MATCH=true
else
  echo "✗ bundle/ outputs differ"
  BUNDLE_MATCH=false
fi

echo ""
if [ -d "$WORK_DIR/python-build/build" ] || [ -d "$WORK_DIR/go-build/build" ]; then
  echo "=== Pruning createdAt timestamps from build/bundle..."
  find "$WORK_DIR" -name "*.clusterserviceversion.yaml" -exec sed -i.bak '/^    createdAt:/d' {} \;

  echo ""
  echo "=== Comparing build/bundle outputs..."
  if diff -ruN "$WORK_DIR/python-build/build" "$WORK_DIR/go-build/build"; then
    echo "✓ build/bundle outputs are identical"
    BUILD_MATCH=true
  else
    echo "✗ build/bundle outputs differ"
    BUILD_MATCH=false
  fi
else
  echo "=== Skipping build/bundle comparison (neither implementation generated it)..."
  BUILD_MATCH=true
fi

# Cleanup
echo ""
echo "=== Cleaning up..."
git restore bundle/ build/ || true
git clean -fd bundle/ build/ || true

# Final result
echo ""
if [ "$BUNDLE_MATCH" = true ] && [ "$BUILD_MATCH" = true ]; then
  echo "✓✓✓ SUCCESS: All outputs are identical ✓✓✓"
  rm -rf "$WORK_DIR"
  exit 0
else
  echo "✗✗✗ FAILURE: Outputs differ ✗✗✗"
  echo "Artifacts saved in: $WORK_DIR"
  exit 1
fi
