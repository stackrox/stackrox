#!/usr/bin/env bash
# PROTOTYPE (do not merge): Azure Blob-backed GOCACHE for CI jobs.
#
# Architecture (the "read-only cloud lower + local writable upper" design):
#   - The storage container mirrors GOCACHE's content-addressed layout
#     (entries under 00-FF/ hash dirs). One container serves ALL jobs —
#     entries are shared globally, deduplicated by content.
#   - blobfuse2 mounts the container READ-ONLY via service principal auth
#     (lazy, parallel, on-demand fetch — a job only downloads the entries
#     it actually hits).
#   - overlayfs puts a local writable layer on top; the Go toolchain reads
#     through to the cloud on miss-local/hit-cloud and writes locally at
#     full speed.
#   - After the job, new local entries are uploaded with azcopy sync
#     (content-addressed keys make concurrent uploads idempotent).
#
# Required GitHub secrets:
#   AZURE_TENANT_ID       - Entra ID tenant
#   AZURE_CLIENT_ID       - service principal client ID
#   AZURE_CLIENT_SECRET   - service principal client secret
#   AZURE_STORAGE_ACCOUNT - storage account name (e.g. ci-gocache storage)
#
# Usage: azure-gocache.sh <mount|sync|unmount>
#   mount    - install blobfuse2, mount RO via SPN, set up overlay
#   sync     - upload new entries from the local upper layer
#   unmount  - unmount everything

set -euo pipefail

MODE="${1:?usage: azure-gocache.sh <mount|sync|unmount>}"
AZ_ROOT="${AZ_GOCACHE_ROOT:-/tmp/az-gocache}"
RO_DIR="$AZ_ROOT/ro"
UPPER_DIR="$AZ_ROOT/rw"
WORK_DIR="$AZ_ROOT/work"
OVERLAY_DIR="${AZ_CACHE_DIR:-$AZ_ROOT/gocache}"
BLOBFUSE2_VERSION="2.4.2"  # ratchet: check https://github.com/Azure/azure-storage-fuse/releases
CONTAINER="${AZURE_GOCACHE_CONTAINER:-ci-gocache}"

TENANT_ID="${AZURE_TENANT_ID:?secret AZURE_TENANT_ID must be set}"
CLIENT_ID="${AZURE_CLIENT_ID:?secret AZURE_CLIENT_ID must be set}"
CLIENT_SECRET="${AZURE_CLIENT_SECRET:?secret AZURE_CLIENT_SECRET must be set}"
STORAGE_ACCOUNT="${AZURE_STORAGE_ACCOUNT:?secret AZURE_STORAGE_ACCOUNT must be set}"

install_blobfuse2() {
    if command -v blobfuse2 >/dev/null 2>&1; then
        return
    fi
    echo "Installing blobfuse2 $BLOBFUSE2_VERSION..."
    local url="https://github.com/Azure/azure-storage-fuse/releases/download/blobfuse2-${BLOBFUSE2_VERSION}/blobfuse2-${BLOBFUSE2_VERSION}-ubuntu-24.04-x86_64.deb"
    curl -fsSL -o /tmp/blobfuse2.deb "$url"
    sudo dpkg -i /tmp/blobfuse2.deb
    blobfuse2 --version
}

install_azcopy() {
    if command -v azcopy >/dev/null 2>&1; then
        return
    fi
    echo "Installing azcopy..."
    curl -fsSL -o /tmp/azcopy.tar.gz "https://azcopyvnext.azureedge.net/release20250123/azcopy_linux_amd64.tar.gz"
    tar -xzf /tmp/azcopy.tar.gz -C /tmp
    sudo install /tmp/azcopy_linux_amd64/azcopy /usr/local/bin/azcopy
    azcopy --version
}

do_mount() {
    install_blobfuse2
    mkdir -p "$RO_DIR" "$UPPER_DIR" "$WORK_DIR" "${OVERLAY_DIR:?OVERLAY_DIR must be set}"

    # blobfuse2 config: SPN auth, read-only mount, lazy on-demand reads, streaming.
    cat > "$AZ_ROOT/blobfuse2-config.yaml" <<EOF
allow-other: false
read-only: true
logging:
  type: base
  level: log_warning
  file-path: $AZ_ROOT/blobfuse2.log
components:
  - libfuse
  - attr_cache
  - azstorage
libfuse:
  attribute-expiration-sec: 120
  entry-expiration-sec: 120
  negative-entry-expiration-sec: 120
attr_cache:
  timeout-sec: 7200
azstorage:
  type: block
  account-name: $STORAGE_ACCOUNT
  container: $CONTAINER
  mode: spn
  tenantid: $TENANT_ID
  clientid: $CLIENT_ID
  clientsecret: $CLIENT_SECRET
  block-size-mb: 8
  max-concurrency: 16
EOF

    echo "Mounting blob container read-only at $RO_DIR..."
    blobfuse2 mount "$RO_DIR" --config-file="$AZ_ROOT/blobfuse2-config.yaml"
    # Wait for the mount to be ready.
    timeout 30 bash -c "until mountpoint -q '$RO_DIR'; do sleep 1; done"

    echo "Creating overlayfs at ${OVERLAY_DIR}..."
    sudo mount -t overlay overlay \
        -o "lowerdir=$RO_DIR,upperdir=$UPPER_DIR,workdir=$WORK_DIR" \
        "${OVERLAY_DIR:?}"

    echo "GOCACHE overlay ready: ${OVERLAY_DIR}"
    echo "Lower (cloud RO): $RO_DIR"
    echo "Upper (local RW): $UPPER_DIR"
    # Emit for the caller to export.
    echo "GOCACHE=${OVERLAY_DIR}"
}

do_sync() {
    install_azcopy
    echo "Syncing new cache entries (upper layer) to the container..."
    # azcopy SPN auth via env vars
    export AZURE_SPA_CLIENT_ID="$CLIENT_ID"
    export AZURE_SPA_CLIENT_SECRET="$CLIENT_SECRET"
    export AZURE_TENANT_ID="$TENANT_ID"
    azcopy login --service-principal > /dev/null 2>&1 || true
    azcopy sync "$UPPER_DIR" "https://$STORAGE_ACCOUNT.blob.core.windows.net/$CONTAINER" \
        --recursive --delete-destination=false \
        --from-to Local Blob \
        --output-level quiet
    azcopy logout > /dev/null 2>&1 || true
    echo "Sync complete."
}

do_unmount() {
    sudo umount "${OVERLAY_DIR:?}" 2>/dev/null || true
    blobfuse2 unmount all 2>/dev/null || true
    echo "Unmounted."
}

case "$MODE" in
    mount)   do_mount ;;
    sync)    do_sync ;;
    unmount) do_unmount ;;
    *)       echo "unknown mode: $MODE" >&2; exit 2 ;;
esac
