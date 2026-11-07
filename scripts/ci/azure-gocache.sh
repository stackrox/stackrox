#!/usr/bin/env bash
# PROTOTYPE (do not merge): Azure Blob-backed GOCACHE for CI jobs.
#
# Architecture (the "read-only cloud lower + local writable upper" design):
#   - The storage container mirrors GOCACHE's content-addressed layout
#     (entries under 00-FF/ hash dirs). One container serves ALL jobs —
#     entries are shared globally, deduplicated by content.
#   - blobfuse2 mounts the container READ-ONLY (lazy, parallel, on-demand
#     fetch — a job only downloads the entries it actually hits).
#   - overlayfs puts a local writable layer on top; the Go toolchain reads
#     through to the cloud on miss-local/hit-cloud and writes locally at
#     full speed.
#   - After the job, new local entries are uploaded with azcopy sync
#     (content-addressed keys make concurrent uploads idempotent).
#
# Required GitHub secrets:
#   AZURE_STORAGE_CONNECTION_STRING  - connection string for the cache
#                                      storage account
#   AZURE_GOCACHE_CONTAINER          - container name (e.g. ci-gocache)
#
# Usage: azure-gocache.sh <mount|sync|unmount>
#   mount    - install blobfuse2, mount RO, set up overlay, export GOCACHE
#              (sourced: sets GOCACHE env var)
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

CONNECTION_STRING="${AZURE_STORAGE_CONNECTION_STRING:?secret AZURE_STORAGE_CONNECTION_STRING must be set}"
CONTAINER="${AZURE_GOCACHE_CONTAINER:-ci-gocache}"

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

account_and_key() {
    # Extract account name and key from the connection string for blobfuse2 config.
    local acct="" key=""
    acct=$(echo "$CONNECTION_STRING" | tr ';' '\n' | grep '^AccountName=' | cut -d= -f2)
    key=$(echo "$CONNECTION_STRING" | tr ';' '\n' | grep '^AccountKey=' | cut -d= -f2)
    echo "$acct $key"
}

do_mount() {
    install_blobfuse2
    mkdir -p "$RO_DIR" "$UPPER_DIR" "$WORK_DIR" "${OVERLAY_DIR:?OVERLAY_DIR must be set}"

    read -r acct key <<<"$(account_and_key)"

    # blobfuse2 config: read-only mount, lazy on-demand reads, streaming.
    cat > "$AZ_ROOT/blobfuse2-config.yaml" <<EOF
allow-other: false
read-only: true
logging:
  type: base
  level: log_warning
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
  account-name: $acct
  account-key: $key
  endpoint: https://$acct.blob.core.windows.net
  mode: key
  container: $CONTAINER
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
    local acct
    acct=$(echo "$CONNECTION_STRING" | tr ';' '\n' | grep '^AccountName=' | cut -d= -f2)
    echo "Syncing new cache entries (upper layer) to the container..."
    azcopy sync "$UPPER_DIR" "https://$acct.blob.core.windows.net/$CONTAINER" \
        --recursive --delete-destination=false \
        --from-to Local Blob \
        --output-level quiet
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
