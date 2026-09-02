#!/usr/bin/env bash
# Extracts CVE allowlist from test files
set -euo pipefail

cd "$(dirname "$0")/../../.."

OUTPUT="scanner/updater/ci/test-cves-allowlist.txt"

# Extract all CVEs from test files
grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' \
  scanner/e2etests/testdata/ \
  qa-tests-backend/src/test/groovy/ 2>/dev/null | \
  sort -u > "$OUTPUT"

CVE_COUNT=$(wc -l < "$OUTPUT")
echo "✓ Extracted $CVE_COUNT unique CVEs to $OUTPUT"

# Show breakdown
SCANNER_COUNT=$(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' scanner/e2etests/testdata/ | sort -u | wc -l)
QA_COUNT=$(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' qa-tests-backend/src/test/groovy/ 2>/dev/null | sort -u | wc -l || echo 0)
echo "  - scanner/e2etests: $SCANNER_COUNT CVEs"
echo "  - qa-tests-backend: $QA_COUNT CVEs"
echo "  - Combined unique: $CVE_COUNT CVEs"
