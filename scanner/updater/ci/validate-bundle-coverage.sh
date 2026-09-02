#!/usr/bin/env bash
# Validates that ci-minimal bundle contains all CVEs tested in CI
set -euo pipefail

cd "$(dirname "$0")/../../.."

BUNDLE_PATH="scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip"

echo "=== Validating CI-Minimal Bundle Coverage ==="
echo ""

# Check bundle exists
if [ ! -f "$BUNDLE_PATH" ]; then
    echo "❌ ERROR: ci-minimal bundle not found at $BUNDLE_PATH"
    echo "Run generate-ci-bundle.sh first"
    exit 1
fi

# Extract CVEs from test files
echo "Extracting CVEs from test files..."
test_cves=$(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' \
  scanner/e2etests/testdata/ \
  qa-tests-backend/src/test/groovy/ 2>/dev/null | sort -u)

test_count=$(echo "$test_cves" | wc -l)
echo "Found $test_count unique CVEs in tests"

# Extract CVEs from ci-minimal bundle
echo "Extracting CVEs from ci-minimal bundle..."
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

unzip -q "$BUNDLE_PATH" -d "$tmp_dir"

# Extract CVE IDs from all bundle files
# The bundle format uses .Vuln.name or .Vuln.Name
bundle_cves=$(find "$tmp_dir" -name "*.json.zst" -exec \
  zstd -dc {} \; 2>/dev/null | \
  jq -r '.Vuln.name // .Vuln.Name // empty' 2>/dev/null | \
  sort -u)

bundle_count=$(echo "$bundle_cves" | grep -c . || echo 0)
echo "Found $bundle_count unique CVEs in bundle"

echo ""

# Compare and report missing CVEs
missing=$(comm -23 <(echo "$test_cves") <(echo "$bundle_cves"))
if [ -n "$missing" ]; then
    missing_count=$(echo "$missing" | wc -l)
    echo "❌ ERROR: ci-minimal bundle missing $missing_count CVEs required by tests:"
    echo "$missing" | head -20
    if [ "$missing_count" -gt 20 ]; then
        echo "... and $((missing_count - 20)) more"
    fi
    echo ""
    echo "Action required: Regenerate bundle or check source selection"
    exit 1
fi

# Report extra CVEs (informational only)
extra=$(comm -13 <(echo "$test_cves") <(echo "$bundle_cves"))
if [ -n "$extra" ]; then
    extra_count=$(echo "$extra" | wc -l)
    echo "ℹ️  Note: Bundle contains $extra_count additional CVEs not explicitly tested"
    echo "   (This is OK - dependencies and related vulnerabilities)"
fi

echo ""
echo "✓ All $test_count test CVEs present in ci-minimal bundle"

# Show bundle size
size=$(stat --format=%s "$BUNDLE_PATH" | numfmt --to=iec-i --suffix=B)
echo "✓ Bundle size: $size"
