#!/usr/bin/env bash
# Filters vulnerability bundles to include ONLY CVEs in allowlist
set -euo pipefail

cd "$(dirname "$0")/../../.."

FULL_BUNDLE_DIR="scanner/updater/ci/bundles/full"
FILTERED_DIR="scanner/updater/ci/bundles/ci-minimal"
ALLOWLIST="scanner/updater/ci/test-cves-allowlist.txt"

echo "=== Filtering Bundles to Test CVEs Only ==="

# Verify allowlist exists
if [ ! -f "$ALLOWLIST" ]; then
    echo "❌ Error: Allowlist not found at $ALLOWLIST"
    echo "Run extract-test-cves.sh first"
    exit 1
fi

# Load allowlist
mapfile -t allowed_cves < "$ALLOWLIST"
echo "Filtering to ${#allowed_cves[@]} allowed CVEs..."

# Clean and create output directory
rm -rf "$FILTERED_DIR"
mkdir -p "$FILTERED_DIR"

# Process each source bundle
total_kept=0
total_original=0

for bundle in "$FULL_BUNDLE_DIR"/*.json.zst; do
  if [ ! -f "$bundle" ]; then
    echo "Warning: No bundles found in $FULL_BUNDLE_DIR"
    continue
  fi

  source=$(basename "$bundle")
  echo "Processing $source..."

  # Create jq filter expression for CVE matching
  # The bundle format uses .Vuln.name or .Vuln.Name for the CVE ID
  cve_array=$(printf '%s\n' "${allowed_cves[@]}" | jq -R . | jq -s .)

  # Decompress, filter by CVE allowlist, recompress
  # Keep entries where the CVE name matches the allowlist
  filtered_content=$(zstd -dc "$bundle" | \
    jq -c --argjson cves "$cve_array" '
      select(
        (.Vuln.name // .Vuln.Name // "") as $cve |
        $cves | index($cve)
      )
    ' 2>/dev/null || true)

  if [ -n "$filtered_content" ]; then
    echo "$filtered_content" | zstd -o "$FILTERED_DIR/$source"

    # Count records
    filtered_count=$(echo "$filtered_content" | wc -l)
    original_count=$(zstd -dc "$bundle" | wc -l)

    echo "  Kept $filtered_count/$original_count records"

    total_kept=$((total_kept + filtered_count))
    total_original=$((total_original + original_count))
  else
    echo "  No matching CVEs found in this source"
  fi
done

echo ""
echo "Total: Kept $total_kept/$total_original records across all sources"

# Package into zip
cd "$FILTERED_DIR"
if compgen -G "*.json.zst" > /dev/null; then
  zip -q vulnerabilities.zip *.json.zst
  cd -

  echo ""
  echo "✓ Generated ci-minimal bundle at $FILTERED_DIR/vulnerabilities.zip"
  ls -lh "$FILTERED_DIR/vulnerabilities.zip"
else
  cd -
  echo "❌ Error: No filtered bundles generated"
  exit 1
fi
