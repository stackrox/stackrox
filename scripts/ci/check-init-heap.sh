#!/usr/bin/env bash
# Checks that the total init() heap allocation stays below a threshold.
# The threshold ratchets down as lazy-init conversions land.
#
# Usage: scripts/ci/check-init-heap.sh [threshold_bytes]
# Default threshold is read from scripts/ci/init-heap-threshold.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
THRESHOLD="${1:-$(cat "$SCRIPT_DIR/init-heap-threshold")}"
BINARY="$(mktemp)"
trap 'rm -f "$BINARY"' EXIT

echo "Building central binary..."
CGO_ENABLED=0 go build -o "$BINARY" ./central/

echo "Measuring init() heap allocation..."
TRACE=$(GODEBUG=inittrace=1 "$BINARY" 2>&1 || true)
heap=$(echo "$TRACE" | grep '^init ' | \
  awk '{for(i=1;i<=NF;i++) if($(i+1)=="bytes,"){gsub(/,/,"",$i);t+=$i}} END{print t+0}')

heap_mb=$(echo "scale=1; $heap / 1048576" | bc)
threshold_mb=$(echo "scale=1; $THRESHOLD / 1048576" | bc)

if [ "$heap" -gt "$THRESHOLD" ]; then
  echo "FAIL: init heap ${heap_mb} MB exceeds threshold ${threshold_mb} MB"
  echo ""
  echo "Top 10 init() allocators:"
  echo "$TRACE" | grep '^init ' | \
    awk '{pkg=$2; for(i=1;i<=NF;i++) if($(i+1)=="bytes,"){gsub(/,/,"",$i); print $i, pkg}}' | \
    sort -rn | head -10
  echo ""
  echo "To fix: convert heavy init() allocations to sync.OnceValue."
  echo "To update threshold: edit scripts/ci/init-heap-threshold"
  exit 1
fi

echo "PASS: init heap ${heap_mb} MB within threshold ${threshold_mb} MB"
