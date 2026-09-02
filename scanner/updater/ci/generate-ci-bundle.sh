#!/usr/bin/env bash
# Orchestrates ci-minimal bundle generation
set -euo pipefail

cd "$(dirname "$0")/../../.."

echo "=========================================="
echo "CI-Minimal Vulnerability Bundle Generator"
echo "==========================================="
echo ""

echo "Step 1: Extract test CVE allowlist..."
./scanner/updater/ci/extract-test-cves.sh

echo ""
echo "Step 2: Generate full bundle from required sources..."
./scanner/updater/ci/generate-full-bundle.sh

echo ""
echo "Step 3: Filter to only test CVEs..."
./scanner/updater/ci/filter-bundle-by-cves.sh

echo ""
echo "=========================================="
echo "✓ CI-minimal bundle generation complete"
echo "=========================================="
ls -lh scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip

# Show size comparison
FULL_SIZE=$(du -sh scanner/updater/ci/bundles/full 2>/dev/null | cut -f1 || echo "N/A")
MINIMAL_SIZE=$(stat --format=%s scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip 2>/dev/null | numfmt --to=iec-i --suffix=B || echo "N/A")

echo ""
echo "Bundle comparison:"
echo "  Full sources (uncompressed): $FULL_SIZE"
echo "  CI-minimal (compressed): $MINIMAL_SIZE"
