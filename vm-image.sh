#!/usr/bin/env bash
#
# Build a customized RHEL ContainerDisk image for KubeVirt VM scanning tests.
#
# Extracts a qcow2 guest disk from a source ContainerDisk image, injects
# additional packages (e.g. bc) via chroot inside a privileged container,
# then rebuilds and pushes a new ContainerDisk to the target registry.
#
# Required env vars:
#   RHEL_ACTIVATION_ORG   - Red Hat subscription org ID
#   RHEL_ACTIVATION_KEY   - Red Hat activation key
#   TARGET_REGISTRY       - Container registry to push the image to
#   TARGET_TAG            - Image tag (e.g. vm-images/rhel9-custom:latest)
#
# Optional env vars:
#   SRC_IMAGE               - Source ContainerDisk image (default: RHEL 9 guest)
#   RHEL_ACTIVATION_ENDPOINT - Custom subscription server URL
#   PLATFORM                - Target platform (default: linux/amd64)
#
# Example:
#   RHEL_ACTIVATION_ORG=12345 \
#   RHEL_ACTIVATION_KEY=my-key \
#   TARGET_REGISTRY=quay.io/my-org \
#   TARGET_TAG=vm-images/rhel9-custom:latest \
#     ./vm-image.sh
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

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

REGISTER_ARGS="--org=$RHEL_ACTIVATION_ORG --activationkey=$RHEL_ACTIVATION_KEY"
[[ -n "$RHEL_ACTIVATION_ENDPOINT" ]] && REGISTER_ARGS+=" --serverurl=$RHEL_ACTIVATION_ENDPOINT"

# --- Step 1: Extract qcow2 from ContainerDisk image ---
PLATFORM="${PLATFORM:-linux/amd64}"
echo "==> Extracting qcow2 from container image (platform=$PLATFORM)..."
CID=$(podman create --platform "$PLATFORM" "$SRC_IMAGE")
podman cp "$CID:/disk/." "$WORKDIR/"
podman rm "$CID"
DISK_NAME="$(basename "$(find "$WORKDIR" -maxdepth 1 -type f \( -name '*.qcow2' -o -name '*.img' \) | head -1)")"
echo "==> Disk: $DISK_NAME"

# --- Steps 2-4: Convert, customize, convert back (all in one privileged container) ---
# Everything runs inside a single privileged container on the podman machine VM
# so that losetup, qemu-img conversions, and chroot all operate on local
# storage (/tmp inside the container) — avoiding virtiofs sync issues that
# corrupted the bootloader when these steps ran in separate contexts.
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

echo "==> Running customization in privileged container on podman machine..."
podman machine ssh sudo podman run --rm --privileged --platform "$PLATFORM" \
  -v "$WORKDIR:/work" \
  registry.access.redhat.com/ubi9/ubi:latest /work/customize.sh

# --- Step 5: Build and push ContainerDisk image ---
echo "==> Building and pushing container image..."
cat > "$WORKDIR/Containerfile" <<'EOF'
FROM scratch
COPY *.qcow2 *.img /disk/
EOF

podman build --platform "$PLATFORM" -t "$TARGET_REGISTRY/$TARGET_TAG" "$WORKDIR"
podman push "$TARGET_REGISTRY/$TARGET_TAG"
echo "==> Done: $TARGET_REGISTRY/$TARGET_TAG"
