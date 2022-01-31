#!/usr/bin/env bash
# Creates a scripts directory and a tgz bundle of binaries and data files
# needed for main-rhel

set -euo pipefail

die() {
    echo >&2 "$@"
    exit 1
}

image_exists() {
  if ! docker image inspect "$1" > /dev/null ; then
     die "Image file $1 not found."
  fi
}

extract_from_image() {
  local image=$1
  local src=$2
  local dst=$3

  [[ -n "$image" && -n "$src" && -n "$dst" ]] \
      || die "extract_from_image: <image> <src> <dst>"

  docker create --name copier "${image}"
  docker cp "copier:${src}" "${dst}"
  docker rm copier

  [[ -s $dst ]] || die "file extracted from image is empty: $dst"
}

INPUT_ROOT="$1"
DATA_IMAGE="$2"
BUILDER_IMAGE="$3"
OUTPUT_DIR="$4"

[[ -n "$INPUT_ROOT" && -n "$DATA_IMAGE" && -n "$BUILDER_IMAGE" && -n "$OUTPUT_DIR" ]] \
    || die "Usage: $0 <input-root-directory> <enc-data-image> <builder-image> <output-directory>"
[[ -d "$INPUT_ROOT" ]] \
    || die "Input root directory doesn't exist or is not a directory."
[[ -d "$OUTPUT_DIR" ]] \
    || die "Output directory doesn't exist or is not a directory."

OUTPUT_BUNDLE="${OUTPUT_DIR}/bundle.tar.gz"

# Verify image exists
image_exists "${DATA_IMAGE}"

# Create tmp directory with stackrox directory structure
bundle_root="$(mktemp -d)"
mkdir -p "${bundle_root}"/{assets/downloads/cli,stackrox/bin,ui,usr/local/bin}
chmod -R 755 "${bundle_root}"

# =============================================================================
# Copy scripts to image build context directory

# Add scripts a be included in the Dockerfile here. These scripts are copied to
# the /stackrox directory in the container image.

mkdir -p "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/central-entrypoint.sh"               "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/entrypoint-wrapper.sh"    "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/import-additional-cas"    "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/db-functions"             "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/move-to-current"          "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/restore-all-dir-contents" "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/save-dir-contents"        "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/start-central.sh"         "${OUTPUT_DIR}/scripts"
cp "${INPUT_ROOT}/static-bin/debug"                    "${OUTPUT_DIR}/scripts"

# =============================================================================
# Copy binaries and data files into bundle

# Add binaries and data files to be included in the Dockerfile here. This
# includes artifacts that would be otherwise downloaded or included via a COPY
# command in the Dockerfile.

cp -p "${INPUT_ROOT}/bin/migrator"          "${bundle_root}/stackrox/bin/"
cp -p "${INPUT_ROOT}/bin/central"           "${bundle_root}/stackrox/"
cp -p "${INPUT_ROOT}/bin/compliance"        "${bundle_root}/stackrox/bin/"
cp -p "${INPUT_ROOT}/bin/roxctl"*           "${bundle_root}/assets/downloads/cli/"
cp -p "${INPUT_ROOT}/bin/kubernetes-sensor" "${bundle_root}/stackrox/bin/"
cp -p "${INPUT_ROOT}/bin/sensor-upgrader"   "${bundle_root}/stackrox/bin/"
cp -p "${INPUT_ROOT}/bin/admission-control" "${bundle_root}/stackrox/bin/"
cp -pr "${INPUT_ROOT}/THIRD_PARTY_NOTICES"  "${bundle_root}/"
cp -pr "${INPUT_ROOT}/ui/build/"*           "${bundle_root}/ui/"

mkdir -p "${bundle_root}/go/bin"
if [[ "$DEBUG_BUILD" == "yes" ]]; then
  GOBIN="${bundle_root}/go/bin" go install github.com/go-delve/delve/cmd/dlv@latest
fi

# Extract data from data container image
extract_from_image "${DATA_IMAGE}" "/stackrox-data" "${bundle_root}/stackrox/static-data/"
extract_from_image "${BUILDER_IMAGE}" "/usr/local/bin/ldb" "${bundle_root}/usr/local/bin/ldb"

# Install all the required compression packages for RocksDB to compile
rpm_base_url="http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages"
rpm_suffix="el8.x86_64.rpm"

curl -s -f -o "${bundle_root}/snappy.rpm" "${rpm_base_url}/snappy-1.1.8-3.${rpm_suffix}"

# =============================================================================

# Files should have owner/group equal to root:root
if tar --version | grep -q "gnu" ; then
  tar_chown_args=("--owner=root:0" "--group=root:0")
else
  tar_chown_args=("--disable-copyfile")
fi

# Create output bundle of all files in $bundle_root
tar cz "${tar_chown_args[@]}" --file "$OUTPUT_BUNDLE" --directory "${bundle_root}" .

# Clean up after success
rm -r "${bundle_root}"
