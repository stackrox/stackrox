#!/usr/bin/env bash
# Generates full bundle from required sources (unfiltered)
set -euo pipefail

cd "$(dirname "$0")/../../.."

SOURCES="alpine,debian,osv,rhel-vex,manual"
OUTPUT_DIR="scanner/updater/ci/bundles/full"

echo "=== Generating Full Bundle from Sources: $SOURCES ==="

# Build updater binary if it doesn't exist
if [ ! -f scanner/bin/updater ]; then
    echo "Building updater binary..."
    make -C scanner bin/updater
fi

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Generate bundle with required sources
echo "Exporting vulnerability bundles..."
STACKROX_SCANNER_V4_UPDATER_SOURCES="$SOURCES" \
  scanner/bin/updater export \
  --manual-url "https://raw.githubusercontent.com/stackrox/stackrox/master/scanner/updater/manual/vulns.yaml" \
  "$OUTPUT_DIR"

echo ""
echo "✓ Generated full bundle at $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR"/*.json.zst
