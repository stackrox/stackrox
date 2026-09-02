#!/usr/bin/env bash
# Generates ci-minimal bundle WITHOUT compression for review
set -euo pipefail

cd "$(dirname "$0")/../../.."

FULL_BUNDLE_DIR="scanner/updater/ci/bundles/full"
REVIEW_DIR="scanner/updater/ci/bundles/review"
ALLOWLIST="scanner/updater/ci/test-cves-allowlist.txt"

echo "=== Generating Uncompressed Bundle for Review ==="
echo ""

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
rm -rf "$REVIEW_DIR"
mkdir -p "$REVIEW_DIR"

# Process each source bundle
total_kept=0
total_original=0

for bundle in "$FULL_BUNDLE_DIR"/*.json.zst; do
  if [ ! -f "$bundle" ]; then
    echo "Warning: No bundles found in $FULL_BUNDLE_DIR"
    echo "Run generate-full-bundle.sh first"
    exit 1
  fi

  source=$(basename "$bundle" .json.zst)
  echo "Processing $source..."

  # Create jq filter expression for CVE matching
  cve_array=$(printf '%s\n' "${allowed_cves[@]}" | jq -R . | jq -s .)

  # Decompress and filter by CVE allowlist
  # Output as uncompressed JSON (one record per line)
  filtered_content=$(zstd -dc "$bundle" | \
    jq -c --argjson cves "$cve_array" '
      select(
        (.Vuln.name // .Vuln.Name // "") as $cve |
        $cves | index($cve)
      )
    ' 2>/dev/null || true)

  if [ -n "$filtered_content" ]; then
    # Save as uncompressed JSON for review
    echo "$filtered_content" > "$REVIEW_DIR/${source}.json"

    # Also create a pretty-printed sample (first 3 records)
    echo "$filtered_content" | head -3 | jq '.' > "$REVIEW_DIR/${source}.sample.json" 2>/dev/null || true

    # Count records
    filtered_count=$(echo "$filtered_content" | wc -l)
    original_count=$(zstd -dc "$bundle" | wc -l)

    echo "  Kept $filtered_count/$original_count records"
    echo "  Saved to: $REVIEW_DIR/${source}.json"

    total_kept=$((total_kept + filtered_count))
    total_original=$((total_original + original_count))
  else
    echo "  No matching CVEs found in this source"
  fi
done

echo ""
echo "Total: Kept $total_kept/$total_original records across all sources"

# Create summary file
cat > "$REVIEW_DIR/SUMMARY.txt" <<EOF
CI-Minimal Bundle - Review Summary
===================================

CVE Allowlist: ${#allowed_cves[@]} CVEs
Total Records: $total_kept (filtered from $total_original)

Files Generated:
EOF

for file in "$REVIEW_DIR"/*.json; do
  if [ -f "$file" ]; then
    count=$(wc -l < "$file")
    size=$(du -h "$file" | cut -f1)
    basename=$(basename "$file")
    echo "  - $basename: $count records, $size" >> "$REVIEW_DIR/SUMMARY.txt"
  fi
done

echo ""
echo "✓ Generated uncompressed bundles in $REVIEW_DIR"
echo ""
cat "$REVIEW_DIR/SUMMARY.txt"
