#!/usr/bin/env bash
#
# Build a customized RHEL ContainerDisk image for KubeVirt VM scanning tests.
#
# APPROACH: privileged UBI9 container + qemu-img + losetup + chroot.
# Extracts a qcow2 from the source ContainerDisk, spins up a privileged
# container, converts qcow2→raw, attaches a loop device, mounts and chroots
# into the guest rootfs, `dnf install`s extra packages, then converts back
# to qcow2 and repackages as a ContainerDisk.
#
# See vm-image-virt-customize.sh for the alternative libguestfs-based
# approach (simpler, but depends on KVM and libguestfs-tools). The two
# scripts are side-by-side for evaluation; one will be kept.
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
#
# Requires a Linux host with a running `podman` that can launch privileged
# containers directly (e.g. GitHub Actions `ubuntu-latest` runners or a Linux
# developer workstation). macOS podman-machine setups are not supported by
# this version of the script.
#
# Example:
#   RHEL_ACTIVATION_ORG=12345 \
#   RHEL_ACTIVATION_KEY=my-key \
#   TARGET_REGISTRY=quay.io/my-org \
#   TARGET_TAG=vm-images/rhel9-custom:latest \
#     ./vm-image-chroot.sh
#
set -euo pipefail

RHEL_ACTIVATION_ORG="${RHEL_ACTIVATION_ORG:?}"
RHEL_ACTIVATION_KEY="${RHEL_ACTIVATION_KEY:?}"
RHEL_ACTIVATION_ENDPOINT="${RHEL_ACTIVATION_ENDPOINT:-}"
TARGET_REGISTRY="${TARGET_REGISTRY:?}"

# SRC_IMAGE: override via env, or default to RHEL 9 guest image.
# oc get istag -n openshift-virtualization-os-images rhel9-guest:latest -o jsonpath='{.image.dockerImageReference}'
SRC_IMAGE="${SRC_IMAGE:-registry.redhat.io/rhel9/rhel-guest-image@sha256:ab4ec16077fe00e3c7efd0b2f6a77571f3645f5c95befc4d917757dc88b2f423}"
TARGET_TAG="${TARGET_TAG:?}"

# Derive source digest from SRC_IMAGE when not explicitly provided. This lets
# the CI workflow record the upstream digest on the built image and later
# detect when upstream has changed without rebuilding.
if [[ -z "${SOURCE_DIGEST:-}" && "$SRC_IMAGE" == *"@sha256:"* ]]; then
  SOURCE_DIGEST="${SRC_IMAGE##*@}"
fi
SOURCE_DIGEST="${SOURCE_DIGEST:-}"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "vm-image-chroot.sh only supports Linux hosts (detected $(uname -s))." >&2
  echo "Run this script from a Linux workstation or the GitHub Actions workflow." >&2
  exit 1
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

REGISTER_ARGS="--org=$RHEL_ACTIVATION_ORG --activationkey=$RHEL_ACTIVATION_KEY"
[[ -n "$RHEL_ACTIVATION_ENDPOINT" ]] && REGISTER_ARGS+=" --serverurl=$RHEL_ACTIVATION_ENDPOINT"

# --- Step 1: Extract qcow2 from ContainerDisk image ---
PLATFORM="${PLATFORM:-linux/amd64}"
echo "==> Extracting qcow2 from container image (platform=$PLATFORM)..."
# Pass a placeholder command: ContainerDisk images carry no CMD/ENTRYPOINT,
# and `podman create` refuses without one even though we never start the
# container (we only `podman cp` out of it).
CID=$(podman create --platform "$PLATFORM" "$SRC_IMAGE" true)
podman cp "$CID:/disk/." "$WORKDIR/"
podman rm "$CID"
DISK_NAME="$(basename "$(find "$WORKDIR" -maxdepth 1 -type f \( -name '*.qcow2' -o -name '*.img' \) | head -1)")"
echo "==> Disk: $DISK_NAME"

# --- Steps 2-4: Convert, customize, convert back (all in one privileged container) ---
# Everything runs inside a single privileged container so that losetup,
# qemu-img conversions, and chroot all operate on container-local /tmp storage
# rather than across a shared bind mount, which is what previously caused
# virtiofs sync issues and bootloader corruption on macOS podman-machine.
echo "==> Writing customization script..."
cat > "$WORKDIR/customize.sh" <<SCRIPT
#!/bin/bash
set -euxo pipefail

subscription-manager register $REGISTER_ARGS >/dev/null
dnf -y -q install qemu-img util-linux

# qcow2 → raw (container-local /tmp, not shared FS)
qemu-img convert -f qcow2 -O raw '/work/$DISK_NAME' /tmp/disk.raw

# losetup + chroot on local file
LOOP=\$(losetup --find --show --partscan /tmp/disk.raw)
echo "Loop device: \$LOOP"
lsblk "\$LOOP"
# Create partition device nodes inside the container (kernel knows them via
# sysfs but udev may not populate /dev inside a privileged container).
LOOPBASE=\$(basename "\$LOOP")
for syspart in /sys/class/block/\${LOOPBASE}p*; do
  [[ -e "\$syspart" ]] || continue
  PARTNAME=\$(basename "\$syspart")
  IFS=: read -r MAJ MIN < "\$syspart/dev"
  [[ -e "/dev/\$PARTNAME" ]] || mknod "/dev/\$PARTNAME" b "\$MAJ" "\$MIN"
  echo "Created /dev/\$PARTNAME (maj=\$MAJ min=\$MIN)"
done
blkid "\${LOOP}"* || true
ROOT=\$(blkid -o device -t LABEL=root "\${LOOP}"* 2>/dev/null || true)
if [[ -z "\$ROOT" ]]; then
  echo "No LABEL=root found; selecting largest partition as root"
  ROOT=\$(lsblk -lnbpo NAME,SIZE "\$LOOP" | grep -v "^\$LOOP " | sort -k2 -n | tail -1 | awk '{print \$1}')
fi
echo "Root partition: \$ROOT"

MNT=/tmp/guest
mkdir -p "\$MNT"
mount "\$ROOT" "\$MNT"
mount --bind /proc "\$MNT/proc"
mount --bind /sys  "\$MNT/sys"
mount --bind /dev  "\$MNT/dev"
cp /etc/resolv.conf "\$MNT/etc/resolv.conf"

if chroot "\$MNT" /bin/true 2>/dev/null; then
  echo "==> Chroot works, customizing guest directly"
  chroot "\$MNT" bash -c '
    subscription-manager register $REGISTER_ARGS &&
    dnf -y install --setopt=install_weak_deps=False bc &&
    subscription-manager unregister &&
    dnf clean all
  '
else
  echo "==> Chroot failed (cross-arch?), falling back to dnf --installroot"
  GUEST_VER=\$(grep '^VERSION_ID=' "\$MNT/etc/os-release" | cut -d= -f2 | tr -d '"' | cut -d. -f1)
  GUEST_ARCH=\$(uname -m)
  [[ "\$GUEST_ARCH" == "aarch64" ]] || GUEST_ARCH="x86_64"
  echo "==> Guest: RHEL \$GUEST_VER (\$GUEST_ARCH)"

  ENT_CERT=\$(ls /etc/pki/entitlement/*.pem 2>/dev/null | grep -v key | head -1)
  ENT_KEY=\$(ls /etc/pki/entitlement/*-key.pem 2>/dev/null | head -1)
  mkdir -p "\$MNT/etc/pki/entitlement" "\$MNT/etc/rhsm/ca"
  cp "\$ENT_CERT" "\$MNT/etc/pki/entitlement/"
  cp "\$ENT_KEY" "\$MNT/etc/pki/entitlement/"
  cp /etc/rhsm/ca/redhat-uep.pem "\$MNT/etc/rhsm/ca/"

  cat > "\$MNT/etc/yum.repos.d/rhel-baseos-cdn.repo" <<REPOEOF
[rhel-\${GUEST_VER}-for-\${GUEST_ARCH}-baseos-rpms]
name=Red Hat Enterprise Linux \${GUEST_VER} for \${GUEST_ARCH} - BaseOS (RPMs)
baseurl=https://cdn.redhat.com/content/dist/rhel\${GUEST_VER}/\${GUEST_VER}/\${GUEST_ARCH}/baseos/os
enabled=1
gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
sslverify=1
sslcacert=/etc/rhsm/ca/redhat-uep.pem
sslclientcert=/etc/pki/entitlement/\$(basename "\$ENT_CERT")
sslclientkey=/etc/pki/entitlement/\$(basename "\$ENT_KEY")
REPOEOF

  dnf -y --installroot="\$MNT" --releasever="\$GUEST_VER" \
    --setopt=install_weak_deps=False install bc

  rm -f "\$MNT/etc/yum.repos.d/rhel-baseos-cdn.repo"
  rm -rf "\$MNT/etc/pki/entitlement" "\$MNT/etc/rhsm/ca"
  dnf --installroot="\$MNT" clean all 2>/dev/null || true
fi

umount "\$MNT/proc" "\$MNT/sys" "\$MNT/dev"
umount "\$MNT"
sync
losetup -d "\$LOOP"

# raw → qcow2 compressed, write result back to shared volume
rm -f '/work/$DISK_NAME'
qemu-img convert -f raw -O qcow2 -c /tmp/disk.raw '/work/$DISK_NAME'
sync
qemu-img info '/work/$DISK_NAME'
echo "==> Customization complete"
SCRIPT
chmod +x "$WORKDIR/customize.sh"

echo "==> Running customization in privileged container..."
podman run --rm --privileged --platform "$PLATFORM" \
  -v "$WORKDIR:/work" \
  registry.access.redhat.com/ubi9/ubi:latest /work/customize.sh

# --- Step 5: Build and push ContainerDisk image ---
echo "==> Building and pushing container image..."
cat > "$WORKDIR/Containerfile" <<'EOF'
FROM scratch
COPY *.qcow2 *.img /disk/
EOF

BUILD_ARGS=(build --platform "$PLATFORM" -t "$TARGET_REGISTRY/$TARGET_TAG")
if [[ -n "$SOURCE_DIGEST" ]]; then
  # Record the upstream source digest as an OCI annotation so the CI workflow
  # can later `skopeo inspect` the published image and skip rebuilds when the
  # upstream image has not changed.
  BUILD_ARGS+=(--annotation "org.opencontainers.image.source-digest=$SOURCE_DIGEST")
  BUILD_ARGS+=(--label "org.opencontainers.image.source-digest=$SOURCE_DIGEST")
fi
BUILD_ARGS+=("$WORKDIR")

podman "${BUILD_ARGS[@]}"
podman push "$TARGET_REGISTRY/$TARGET_TAG"
echo "==> Done: $TARGET_REGISTRY/$TARGET_TAG (source-digest=${SOURCE_DIGEST:-<none>})"
