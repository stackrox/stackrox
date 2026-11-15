#!/usr/bin/env bash
# fast-path.sh — the content-addressed build system's fast path.
#
# Traces backward from the output: "does the image I need already exist
# for this exact source state?" If yes, re-tag it (metadata-only, ~2s).
# If no, signals the caller to take the build path.
#
# Usage: fast-path.sh <branding> <arch> <tag>
# Outputs:
#   skip=true|false   (whether to skip the build)
#   Exits 0 either way (the caller checks the output)

set -euo pipefail

BRANDING="${1:?branding required}"
ARCH="${2:?arch required}"
TAG="${3:?tag required}"
GCS="${GOCACHEPROG:?GOCACHEPROG must be set}"

# The source tree hash: deterministic, fast, invariant under empty commits.
# Same tree = same source = same expected build output.
TREE_HASH=$(git rev-parse HEAD^{tree} 2>/dev/null | cut -c1-16)
if [ -z "$TREE_HASH" ]; then
  echo "skip=false" >> "$GITHUB_OUTPUT"
  echo "fast-path: no git tree hash (not a git repo?) — building"
  exit 0
fi

RECEIPT_KEY="build-receipt-${BRANDING}-${ARCH}-${TREE_HASH}"
RECEIPT_FILE="/tmp/build-receipt.json"

if "$GCS" -fetch-any "$RECEIPT_KEY" "$RECEIPT_FILE" 2>/dev/null && [ -s "$RECEIPT_FILE" ]; then
  echo "fast-path: receipt found for tree ${TREE_HASH}"

  # Parse the receipt
  DIGEST=$(jq -r '.digest // empty' "$RECEIPT_FILE" 2>/dev/null)
  REGISTRY=$(jq -r '.registry // empty' "$RECEIPT_FILE" 2>/dev/null)
  RECEIPT_TAG=$(jq -r '.tag // empty' "$RECEIPT_FILE" 2>/dev/null)

  if [ -n "$DIGEST" ] && [ -n "$REGISTRY" ]; then
    # Re-tag: metadata-only operation via imagetools (no layer transfer).
    # ~2-3s per image.
    if docker buildx imagetools create \
      -t "${REGISTRY}/main:${TAG}-${ARCH}" \
      "${REGISTRY}/main@${DIGEST}" 2>/dev/null; then
      echo "skip=true" >> "$GITHUB_OUTPUT"
      echo "fast-path: re-tagged ${REGISTRY}/main:${TAG}-${ARCH} from ${DIGEST:0:20}..."
      exit 0
    else
      echo "fast-path: re-tag failed (digest may be GC'd) — falling through to build"
    fi
  fi
fi

echo "skip=false" >> "$GITHUB_OUTPUT"
echo "fast-path: no receipt for tree ${TREE_HASH} — building"
