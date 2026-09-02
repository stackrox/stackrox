#!/usr/bin/env bash
# Adds synthetic vulnerability records for missing test CVEs
set -euo pipefail

cd "$(dirname "$0")/../../.."

REVIEW_DIR="scanner/updater/ci/bundles/review"
ALLOWLIST="scanner/updater/ci/test-cves-allowlist.txt"
SYNTHETIC_FILE="$REVIEW_DIR/synthetic.json"

echo "=== Adding Synthetic Records for Missing CVEs ==="
echo ""

# Get CVEs already in bundle
existing_cves=$(find "$REVIEW_DIR" -name "*.json" -not -name "*.sample.json" -not -name "synthetic.json" -exec \
  jq -r '.Vuln.name // empty' {} \; 2>/dev/null | sort -u)

# Find missing CVEs
missing_cves=$(comm -23 <(sort "$ALLOWLIST") <(echo "$existing_cves"))

if [ -z "$missing_cves" ]; then
    echo "✓ All CVEs present in bundle - no synthetic records needed"
    exit 0
fi

missing_count=$(echo "$missing_cves" | wc -l)
echo "Found $missing_count missing CVEs - generating synthetic records..."
echo ""

# Generate synthetic records
: > "$SYNTHETIC_FILE"

while IFS= read -r cve; do
    echo "Adding $cve..."
    cat >> "$SYNTHETIC_FILE" <<EOF
{"Updater":"synthetic-ci-test","Fingerprint":"\"ci-test-data\"","Date":"$(date -Iseconds)","Ref":"00000000-0000-0000-0000-000000000000","Vuln":{"id":"","updater":"synthetic-ci-test","name":"$cve","description":"Synthetic vulnerability record for CI testing - contains no real data","issued":"2026-01-01T00:00:00Z","links":"https://nvd.nist.gov/vuln/detail/$cve","severity":"Unknown","normalized_severity":"Unknown","package":{"id":"","name":"synthetic-package","version":"","kind":"binary","normalized_version":"","cpe":"","detector":null},"distribution":{"id":"","did":"synthetic","name":"Synthetic CI Test Distribution","version":"","version_code_name":"","version_id":"1.0","arch":"","cpe":"","pretty_name":"Synthetic CI Test Distribution v1.0"},"fixed_in_version":"999.999.999","Self":{"space":"","name":""},"Aliases":null,"Invert":false},"Kind":"vulnerability"}
EOF
done <<< "$missing_cves"

echo ""
echo "✓ Generated $missing_count synthetic records in $SYNTHETIC_FILE"

# Show sample
echo ""
echo "Sample synthetic record:"
head -1 "$SYNTHETIC_FILE" | jq '.'
