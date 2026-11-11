#!/bin/bash
# Bootstrap: download gocache-serve binary from blob store.
# 647 bytes in the repo vs 7MB committed binary.
# Downloads the zstd-compressed stripped binary (~2.4MB), decompresses (~35ms).
# Uses our SPN env vars + curl. No az CLI, no managed identity.
#
# Usage: bootstrap-gocache-serve.sh <source-hash> <output-path>
# Env: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET
set -euo pipefail

SRC_HASH="${1:?source hash required}"
OUT="${2:?output path required}"
ACCOUNT="${AZURE_STORAGE_ACCOUNT:-cigocache}"
CONTAINER="${GOCACHE_SERVE_CONTAINER:-ci-gocache}"
KEY="gocache-serve-${SRC_HASH}"

# Get OAuth token from our SPN credentials
TOKEN=$(curl -s -X POST \
  "https://login.microsoftonline.com/${AZURE_TENANT_ID}/oauth2/v2.0/token" \
  -d "grant_type=client_credentials&client_id=${AZURE_CLIENT_ID}&client_secret=${AZURE_CLIENT_SECRET}&scope=https://storage.azure.com/.default" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])')

# Download the zstd-compressed binary
curl -sL -H "Authorization: Bearer $TOKEN" \
  "https://${ACCOUNT}.blob.core.windows.net/${CONTAINER}/${KEY}.zst" \
  -o "${OUT}.zst"

# Decompress and make executable (zstd is on runners; fallback to gzip)
if command -v zstd >/dev/null 2>&1; then
  zstd -d -c "${OUT}.zst" > "$OUT"
else
  # Fallback: try gzip-named blob
  curl -sL -H "Authorization: Bearer $TOKEN" \
    "https://${ACCOUNT}.blob.core.windows.net/${CONTAINER}/${KEY}.gz" \
    | gunzip > "$OUT"
fi
rm -f "${OUT}.zst"
chmod +x "$OUT"

# Verify it's a valid executable
if ! file "$OUT" | grep -q "executable"; then
  echo "ERROR: downloaded file is not an executable" >&2
  rm -f "$OUT"
  exit 1
fi
