#!/usr/bin/env bash
# Downloads public vulnerability bundle from GCS
set -euo pipefail

cd "$(dirname "$0")/../../.."

BUNDLE_URL="https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip"
OUTPUT_DIR="scanner/updater/ci/bundles/full"

echo "=== Downloading Public Vulnerability Bundle ==="
echo "Source: $BUNDLE_URL"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Download bundle
echo "Downloading bundle..."
curl --fail --silent --show-error \
  --retry 3 --retry-delay 5 \
  --connect-timeout 10 --max-time 300 \
  -o "$OUTPUT_DIR/vulnerabilities.zip" \
  "$BUNDLE_URL"

# Extract bundle
echo "Extracting bundle..."
cd "$OUTPUT_DIR"
unzip -q vulnerabilities.zip

echo ""
echo "✓ Downloaded and extracted bundle to $OUTPUT_DIR"
ls -lh *.json.zst
