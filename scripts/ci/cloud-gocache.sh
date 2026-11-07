#!/usr/bin/env bash
# PROTOTYPE (do not merge): cloud-backed GOCACHE for CI jobs.
# Supports three backends for comparison: azure (blobfuse2), gcs (gcsfuse),
# s3 (Mountpoint for S3). All use the same overlayfs pattern:
# read-only cloud mount (lazy, on-demand) + local writable upper layer.
#
# Usage: cloud-gocache.sh <azure|gcs|s3> <mount|sync|unmount>
#
# Required secrets (per backend):
#   azure: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_STORAGE_ACCOUNT
#   gcs:   GCP_SERVICE_ACCOUNT_STACKROX_CI (existing, via job-preamble)
#   s3:    AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (existing)

set -euo pipefail

BACKEND="${1:?usage: cloud-gocache.sh <azure|gcs|s3> <mount|sync|unmount>}"
MODE="${2:?usage: cloud-gocache.sh <azure|gcs|s3> <mount|sync|unmount>}"

AZ_ROOT="/tmp/cloud-gocache"
RO_DIR="$AZ_ROOT/ro"
UPPER_DIR="$AZ_ROOT/rw"
WORK_DIR="$AZ_ROOT/work"
OVERLAY_DIR="$AZ_ROOT/gocache"
GCS_BUCKET="ci-gocache-proto"
S3_BUCKET="ci-gocache-proto"
AZURE_CONTAINER="ci-gocache"

BLOBFUSE2_VERSION="2.5.6"
GCSFUSE_VERSION="3.13.0"

step() { echo "=== cloud-gocache [$BACKEND] $1 at $(date -u +%H:%M:%S) ==="; }

# ---------------------------------------------------------------- install

install_blobfuse2() {
    command -v blobfuse2 >/dev/null 2>&1 && return
    step "installing blobfuse2"
    # Debian 12 build is compatible with Ubuntu 24.04
    curl -fsSL -o /tmp/blobfuse2.deb \
        "https://github.com/Azure/azure-storage-fuse/releases/download/blobfuse2-${BLOBFUSE2_VERSION}/blobfuse2-${BLOBFUSE2_VERSION}-Debian-12.0.x86_64.deb"
    sudo dpkg -i /tmp/blobfuse2.deb
}

install_gcsfuse() {
    command -v gcsfuse >/dev/null 2>&1 && return
    step "installing gcsfuse"
    curl -fsSL -o /tmp/gcsfuse.deb \
        "https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v${GCSFUSE_VERSION}/gcsfuse_${GCSFUSE_VERSION}_amd64.deb"
    sudo dpkg -i /tmp/gcsfuse.deb
}

install_mountpoint_s3() {
    command -v mount-s3 >/dev/null 2>&1 && return
    step "installing mountpoint-s3"
    # Distributed from AWS's own S3 release bucket, not GitHub releases
    curl -fsSL -o /tmp/mount-s3.deb \
        "https://s3.amazonaws.com/mountpoint-s3-release/latest/x86_64/mount-s3.deb"
    sudo apt-get install -y /tmp/mount-s3.deb
}

install_azcopy() {
    command -v azcopy >/dev/null 2>&1 && return
    curl -fsSL -o /tmp/azcopy.tar.gz \
        "https://azcopyvnext.azureedge.net/release20250123/azcopy_linux_amd64.tar.gz"
    tar -xzf /tmp/azcopy.tar.gz -C /tmp
    sudo install /tmp/azcopy_linux_amd64/azcopy /usr/local/bin/azcopy
}

# ---------------------------------------------------------------- mount

# Prefetch backend: no FUSE, no overlayfs. Bulk-parallel download the
# entire bucket to local disk before the job, run at native filesystem
# speed, then sync new entries back after. The read tool is the cloud
# CLI (azcopy/gcloud/aws), not a filesystem layer.
do_prefetch() {
    mkdir -p "$OVERLAY_DIR"

    case "$BACKEND" in
        azure)
            install_azcopy
            export AZURE_SPA_CLIENT_ID="${AZURE_CLIENT_ID:?}"
            export AZURE_SPA_CLIENT_SECRET="${AZURE_CLIENT_SECRET:?}"
            export AZURE_TENANT_ID="${AZURE_TENANT_ID:?}"
            azcopy login --service-principal > /dev/null 2>&1 || true
            step "prefetching from azure blob (parallel bulk download)"
            azcopy copy \
                "https://${AZURE_STORAGE_ACCOUNT:?}.blob.core.windows.net/$AZURE_CONTAINER/*" \
                "$OVERLAY_DIR" \
                --recursive --parallel-level 32 --output-level quiet
            ;;
        gcs)
            step "prefetching from gcs (parallel bulk download)"
            gcloud storage cp -r "gs://$GCS_BUCKET/*" "$OVERLAY_DIR" \
                --no-user-output-enabled 2>/dev/null || true
            ;;
        s3)
            step "prefetching from s3 (parallel bulk download)"
            aws s3 cp "s3://$S3_BUCKET" "$OVERLAY_DIR" \
                --recursive --only-show-errors --no-progress
            ;;
    esac

    local entry_count=$(find "$OVERLAY_DIR" -type f 2>/dev/null | wc -l)
    local total_bytes=$(du -sb "$OVERLAY_DIR" 2>/dev/null | cut -f1 || echo 0)
    step "prefetch complete: $entry_count entries, $(( ${total_bytes:-0} / 1048576 )) MB"

    echo "GOCACHE=${OVERLAY_DIR}"
}

do_prefetch_sync() {
    # Same sync logic as FUSE backends, but the "upper layer" is the
    # entire local directory (everything not already in the cloud is new).
    do_sync
}

do_mount() {
    if [ "$BACKEND" = "prefetch" ]; then
        do_prefetch
        return
    fi
    mkdir -p "$RO_DIR" "$UPPER_DIR" "$WORK_DIR" "$OVERLAY_DIR"

    case "$BACKEND" in
        azure)
            install_blobfuse2
            local acct="${AZURE_STORAGE_ACCOUNT:?AZURE_STORAGE_ACCOUNT must be set}"
            cat > "$AZ_ROOT/fuse-config.yaml" <<EOF
allow-other: false
read-only: true
logging:
  type: base
  level: log_warning
  file-path: $AZ_ROOT/fuse.log
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
  container: $AZURE_CONTAINER
  mode: spn
  tenantid: ${AZURE_TENANT_ID:?}
  clientid: ${AZURE_CLIENT_ID:?}
  clientsecret: ${AZURE_CLIENT_SECRET:?}
  block-size-mb: 8
  max-concurrency: 16
EOF
            step "mounting azure blob (blobfuse2 SPN)"
            blobfuse2 mount "$RO_DIR" --config-file="$AZ_ROOT/fuse-config.yaml"
            ;;
        gcs)
            install_gcsfuse
            # gcsfuse auth: uses GOOGLE_APPLICATION_CREDENTIALS (set by job-preamble's gcloud auth)
            step "mounting gcs bucket (gcsfuse)"
            gcsfuse --implicit-dirs --only-dir= "$GCS_BUCKET" "$RO_DIR" \
                --o ro \
                --metadata-cache-ttl-secs 7200 \
                --metadata-cache-stat-cache-capacity 4096 \
                --metadata-cache-type-cache-capacity 4096 \
                --client-protocol http2
            ;;
        s3)
            install_mountpoint_s3
            # mountpoint-s3 auth: uses AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY env
            step "mounting s3 bucket (mountpoint-s3)"
            mount-s3 "$S3_BUCKET" "$RO_DIR" --read-only \
                --metadata-ttl 120 \
                --max-parallel-requests 16
            ;;
    esac

    timeout 30 bash -c "until mountpoint -q '$RO_DIR'; do sleep 1; done" \
        || { echo "ERROR: $BACKEND mount did not become ready"; exit 1; }
    step "cloud mount ready at $RO_DIR"

    step "creating overlayfs"
    sudo mount -t overlay overlay \
        -o "lowerdir=$RO_DIR,upperdir=$UPPER_DIR,workdir=$WORK_DIR" \
        "$OVERLAY_DIR"

    echo "GOCACHE=${OVERLAY_DIR}"
}

# ---------------------------------------------------------------- sync

do_sync() {
    step "syncing new entries"
    local sync_dir="$UPPER_DIR"
    if [ "$BACKEND" = "prefetch" ] || [ ! -d "$sync_dir" ]; then
        sync_dir="$OVERLAY_DIR"
    fi
    if [ ! -d "$sync_dir" ]; then
        step "no sync directory found, skipping"
        return
    fi
    local entry_count
    entry_count=$(find "$sync_dir" -type f 2>/dev/null | wc -l)
    if [ "$entry_count" -eq 0 ]; then
        step "0 entries to sync, skipping"
        return
    fi
    case "$BACKEND" in
        azure|prefetch)
            install_azcopy
            export AZURE_SPA_CLIENT_ID="${AZURE_CLIENT_ID:?}"
            export AZURE_SPA_CLIENT_SECRET="${AZURE_CLIENT_SECRET:?}"
            export AZURE_TENANT_ID="${AZURE_TENANT_ID:?}"
            azcopy login --service-principal 2>&1 | tail -1
            azcopy sync "$sync_dir" \
                "https://${AZURE_STORAGE_ACCOUNT:?}.blob.core.windows.net/$AZURE_CONTAINER" \
                --recursive --delete-destination=false \
                --output-level quiet 2>&1 | tail -3
            azcopy logout > /dev/null 2>&1 || true
            ;;
        gcs)
            gcloud storage rsync --recursive --no-clobber \
                "$sync_dir" "gs://$GCS_BUCKET" 2>&1 | tail -5
            ;;
        s3)
            aws s3 sync "$sync_dir" "s3://$S3_BUCKET" \
                --only-show-errors --no-progress 2>&1 | tail -3
            ;;
    esac
    step "sync complete ($entry_count entries)"
}

# ---------------------------------------------------------------- unmount

do_unmount() {
    if [ "$BACKEND" = "prefetch" ]; then
        step "prefetch mode: nothing to unmount"
        return
    fi
    sudo umount "$OVERLAY_DIR" 2>/dev/null || true
    case "$BACKEND" in
        azure) blobfuse2 unmount all 2>/dev/null || true ;;
        gcs)   fusermount -u "$RO_DIR" 2>/dev/null || true ;;
        s3)    sudo umount "$RO_DIR" 2>/dev/null || true ;;
    esac
    step "unmounted"
}

case "$MODE" in
    mount)   do_mount ;;
    sync)    do_sync ;;
    unmount) do_unmount ;;
    *)       echo "unknown mode: $MODE" >&2; exit 2 ;;
esac
