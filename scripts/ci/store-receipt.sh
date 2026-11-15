#!/usr/bin/env bash
# store-receipt.sh — record the build output for the fast path.
#
# After a successful image build+push, stores a receipt in the blob store
# keyed by the source tree hash. The next run with the same source can
# re-tag from this receipt instead of rebuilding (~2s vs minutes).
#
# Usage: store-receipt.sh <branding> <arch> <registry> <tag> <digest>

set -euo pipefail

BRANDING="${1:?branding required}"
ARCH="${2:?arch required}"
REGISTRY="${3:?registry required}"
TAG="${4:?tag required}"
DIGEST="${5:?digest required}"
GCS="${GOCACHEPROG:?GOCACHEPROG must be set}"

TREE_HASH=$(git rev-parse HEAD^{tree} 2>/dev/null | cut -c1-16)
[ -z "$TREE_HASH" ] && { echo "store-receipt: no tree hash"; exit 1; }

RECEIPT_KEY="build-receipt-${BRANDING}-${ARCH}-${TREE_HASH}"

cat > /tmp/build-receipt-out.json <<EOF
{
  "tree_hash": "${TREE_HASH}",
  "branding": "${BRANDING}",
  "arch": "${ARCH}",
  "registry": "${REGISTRY}",
  "tag": "${TAG}",
  "digest": "${DIGEST}",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

"$GCS" -store-all "$RECEIPT_KEY" /tmp/build-receipt-out.json
echo "store-receipt: ${RECEIPT_KEY} -> ${DIGEST:0:20}..."
