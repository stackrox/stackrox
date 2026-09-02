#!/usr/bin/env bash
# Trims unnecessary fields from vulnerability bundles to reduce size
set -euo pipefail

cd "$(dirname "$0")/../../.."

REVIEW_DIR="scanner/updater/ci/bundles/review"
TRIMMED_DIR="scanner/updater/ci/bundles/trimmed"

echo "=== Trimming Bundle Fields ==="
echo ""

mkdir -p "$TRIMMED_DIR"

# Define which fields to keep (minimal set for CI testing)
# We keep only fields needed for CVE matching and basic display
JQ_FILTER='
{
  Updater: .Updater,
  Fingerprint: .Fingerprint,
  Date: .Date,
  Ref: .Ref,
  Vuln: {
    name: .Vuln.name,
    description: (.Vuln.description | split("\n")[0] | .[0:200]),  # First line only, max 200 chars
    severity: .Vuln.severity,
    normalized_severity: .Vuln.normalized_severity,
    package: {
      name: .Vuln.package.name,
      kind: .Vuln.package.kind
    },
    distribution: {
      did: .Vuln.distribution.did,
      name: .Vuln.distribution.name,
      version_id: .Vuln.distribution.version_id,
      pretty_name: .Vuln.distribution.pretty_name
    },
    fixed_in_version: .Vuln.fixed_in_version
  },
  Kind: .Kind
}
'

for bundle in "$REVIEW_DIR"/*.json; do
  if [[ "$bundle" == *.sample.json ]] || [[ "$bundle" == */SUMMARY* ]]; then
    continue
  fi

  basename=$(basename "$bundle")
  echo "Trimming $basename..."

  # Apply filter to remove verbose fields
  original_size=$(stat --format=%s "$bundle")
  jq -c "$JQ_FILTER" "$bundle" > "$TRIMMED_DIR/$basename"
  trimmed_size=$(stat --format=%s "$TRIMMED_DIR/$basename")

  reduction=$((100 - (trimmed_size * 100 / original_size)))
  echo "  Original: $(numfmt --to=iec-i --suffix=B $original_size)"
  echo "  Trimmed:  $(numfmt --to=iec-i --suffix=B $trimmed_size)"
  echo "  Reduction: ${reduction}%"
  echo ""
done

# Create pretty-printed samples
for f in "$TRIMMED_DIR"/*.json; do
  basename=$(basename "$f" .json)
  head -3 "$f" | jq '.' > "$TRIMMED_DIR/${basename}.sample.json" 2>/dev/null || true
done

echo "✓ Trimmed bundles saved to $TRIMMED_DIR"
