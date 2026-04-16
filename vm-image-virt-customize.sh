#!/usr/bin/env bash
#
# Build a customized RHEL ContainerDisk image for KubeVirt VM scanning tests.
#
# APPROACH: libguestfs (`virt-customize`).
# Extracts a qcow2 from the source ContainerDisk, then runs `virt-customize`
# which boots a tiny qemu appliance, executes commands inside the guest's
# own package manager, and writes the result back. No privileged container,
# no losetup, no chroot — libguestfs handles mount/unmount/selinux-relabel.
#
# Depends on libguestfs-tools (virt-customize) + qemu-utils + podman.
# Works unaccelerated but wants /dev/kvm for tolerable speed. GitHub Actions
# `ubuntu-latest` runners expose /dev/kvm.
#
# See vm-image-chroot.sh for the alternative approach (privileged container
# with losetup+chroot). The two scripts are side-by-side for evaluation;
# one will be kept.
#
# Required env vars:
#   RHEL_ACTIVATION_ORG   - Red Hat subscription org ID
#   RHEL_ACTIVATION_KEY   - Red Hat activation key
#   TARGET_REGISTRY       - Container registry to push the image to
#   TARGET_TAG            - Image tag (e.g. vm-images/rhel9-custom:latest)
#
# Optional env vars:
#   SRC_IMAGE               - Source ContainerDisk image (default: RHEL 9 guest)
#   SOURCE_DIGEST           - Upstream source-image digest to record as an OCI
#                             annotation on the built image. Used by CI to
#                             detect when the upstream image has changed.
#                             Defaults to the @sha256:... suffix of SRC_IMAGE
#                             when present.
#   RHEL_ACTIVATION_ENDPOINT - Custom subscription server URL
#   PLATFORM                - Target platform (default: linux/amd64)
#   EXTRA_PACKAGES          - Space-separated list of extra packages to install
#                             (default: "bc")
#
# Requires a Linux host with `podman`, `qemu-img`, and `virt-customize`
# available on $PATH. Apple Silicon / macOS hosts are not supported.
#
# Example:
#   RHEL_ACTIVATION_ORG=12345 \
#   RHEL_ACTIVATION_KEY=my-key \
#   TARGET_REGISTRY=quay.io/my-org \
#   TARGET_TAG=vm-images/rhel9-custom:latest \
#     ./vm-image-virt-customize.sh
#
set -euo pipefail

RHEL_ACTIVATION_ORG="${RHEL_ACTIVATION_ORG:?}"
RHEL_ACTIVATION_KEY="${RHEL_ACTIVATION_KEY:?}"
RHEL_ACTIVATION_ENDPOINT="${RHEL_ACTIVATION_ENDPOINT:-}"
TARGET_REGISTRY="${TARGET_REGISTRY:?}"

SRC_IMAGE="${SRC_IMAGE:-registry.redhat.io/rhel9/rhel-guest-image@sha256:ab4ec16077fe00e3c7efd0b2f6a77571f3645f5c95befc4d917757dc88b2f423}"
TARGET_TAG="${TARGET_TAG:?}"
PLATFORM="${PLATFORM:-linux/amd64}"
EXTRA_PACKAGES="${EXTRA_PACKAGES:-bc}"

if [[ -z "${SOURCE_DIGEST:-}" && "$SRC_IMAGE" == *"@sha256:"* ]]; then
  SOURCE_DIGEST="${SRC_IMAGE##*@}"
fi
SOURCE_DIGEST="${SOURCE_DIGEST:-}"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "vm-image-virt-customize.sh only supports Linux hosts (detected $(uname -s))." >&2
  exit 1
fi

for tool in podman qemu-img virt-customize; do
  command -v "$tool" >/dev/null || {
    echo "Missing required tool: $tool" >&2
    echo "Install with: sudo apt-get install -y podman qemu-utils libguestfs-tools" >&2
    exit 1
  }
done

# libguestfs needs a world-readable kernel on some distros to build its
# appliance. `ubuntu-latest` runners have this correctly set already; the
# chmod below is a no-op there and a one-liner fix on restrictive hosts.
sudo chmod 0644 /boot/vmlinuz-* 2>/dev/null || true

if [[ ! -e /dev/kvm ]]; then
  echo "WARNING: /dev/kvm not present — virt-customize will run without" >&2
  echo "acceleration and take several minutes." >&2
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# --- Step 1: Extract qcow2 from ContainerDisk image ---
echo "==> Extracting qcow2 from container image (platform=$PLATFORM)..."
# Pass a placeholder command: ContainerDisk images carry no CMD/ENTRYPOINT,
# and `podman create` refuses without one even though we never start the
# container (we only `podman cp` out of it).
CID=$(podman create --platform "$PLATFORM" "$SRC_IMAGE" true)
podman cp "$CID:/disk/." "$WORKDIR/"
podman rm "$CID"
DISK_NAME="$(basename "$(find "$WORKDIR" -maxdepth 1 -type f \( -name '*.qcow2' -o -name '*.img' \) | head -1)")"
DISK_PATH="$WORKDIR/$DISK_NAME"
echo "==> Disk: $DISK_NAME"

# --- Step 2: Customize with virt-customize ---
# Activation keys aren't a first-class selector for --sm-credentials, so
# register/unregister via --run-command which accepts the same CLI flags
# that `subscription-manager` takes inside the guest.
REGISTER_CMD="subscription-manager register --org=$RHEL_ACTIVATION_ORG --activationkey=$RHEL_ACTIVATION_KEY"
[[ -n "$RHEL_ACTIVATION_ENDPOINT" ]] && REGISTER_CMD+=" --serverurl=$RHEL_ACTIVATION_ENDPOINT"

echo "==> Running virt-customize (installing: $EXTRA_PACKAGES)..."
# TEMPORARY: -v -x + LIBGUESTFS_DEBUG/TRACE until the appliance launches
# cleanly in CI. Strip once the Ubuntu 24.04 runner is fully sorted.
export LIBGUESTFS_DEBUG=1 LIBGUESTFS_TRACE=1
# shellcheck disable=SC2086 # EXTRA_PACKAGES is intentionally word-split.
virt-customize -v -x -a "$DISK_PATH" \
  --run-command "$REGISTER_CMD" \
  --install "$(echo "$EXTRA_PACKAGES" | tr ' ' ',')" \
  --run-command 'subscription-manager unregister' \
  --run-command 'dnf clean all' \
  --selinux-relabel

# --- Step 3: Compact the image ---
# virt-customize doesn't compress the qcow2; re-convert with -c to keep the
# ContainerDisk layer small.
echo "==> Compacting qcow2..."
qemu-img convert -f qcow2 -O qcow2 -c "$DISK_PATH" "$DISK_PATH.compact"
mv "$DISK_PATH.compact" "$DISK_PATH"
qemu-img info "$DISK_PATH"

# --- Step 4: Build and push ContainerDisk image ---
echo "==> Building and pushing container image..."
cat > "$WORKDIR/Containerfile" <<'EOF'
FROM scratch
COPY *.qcow2 *.img /disk/
EOF

BUILD_ARGS=(build --platform "$PLATFORM" -t "$TARGET_REGISTRY/$TARGET_TAG")
if [[ -n "$SOURCE_DIGEST" ]]; then
  BUILD_ARGS+=(--annotation "org.opencontainers.image.source-digest=$SOURCE_DIGEST")
  BUILD_ARGS+=(--label "org.opencontainers.image.source-digest=$SOURCE_DIGEST")
fi
BUILD_ARGS+=("$WORKDIR")

podman "${BUILD_ARGS[@]}"
podman push "$TARGET_REGISTRY/$TARGET_TAG"
echo "==> Done: $TARGET_REGISTRY/$TARGET_TAG (source-digest=${SOURCE_DIGEST:-<none>})"
