#!/usr/bin/env bash
# Analyzes test images to determine required vulnerability sources
set -euo pipefail

cd "$(dirname "$0")/../../.."

# Extract image OS distributions from test data
echo "=== Test Image Distribution ==="
jq -r '.[].image' scanner/e2etests/testdata/image_tests.json | \
  sed 's/:.*$//' | sort | uniq -c | sort -rn

# Extract all CVEs tested
echo ""
echo "=== CVE Coverage ==="
echo "scanner/e2etests: $(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' scanner/e2etests/testdata/ | sort -u | wc -l) CVEs"
echo "qa-tests-backend: $(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' qa-tests-backend/src/test/groovy/ 2>/dev/null | sort -u | wc -l || echo 0) CVEs"
echo "Total unique: $(grep -roh 'CVE-[0-9]\{4\}-[0-9]\+' scanner/e2etests/testdata/ qa-tests-backend/src/test/groovy/ 2>/dev/null | sort -u | wc -l || echo 0) CVEs"

# Recommend minimal sources
echo ""
echo "=== Recommended Minimal Sources ==="
echo "- alpine    (Alpine Linux: 3.13-3.18, nginx:alpine)"
echo "- debian    (Ubuntu/Debian: ubuntu:16.04-23.10, debian:12.0)"
echo "- osv       (Java/Node.js: Jackson, Log4j, Spring, npm packages)"
echo "- rhel-vex  (RHEL/UBI: ubi9, rhacs-collector, jenkins-agent)"
echo "- manual    (Urgent vulnerability fixes)"
