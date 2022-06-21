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

# Verify images exist
if [[ "${DATA_IMAGE}" != "local" ]]; then
  image_exists "${DATA_IMAGE}"
fi

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
  if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    GOBIN= GOOS=linux GOARCH=amd64 GOPATH="${bundle_root}/go" go install github.com/go-delve/delve/cmd/dlv@latest
    mv "$bundle_root"/go/bin/linux_amd64/dlv "$bundle_root"/go/bin/dlv
    rm -r "$bundle_root"/go/bin/linux_amd64
  else
    GOBIN="${bundle_root}/go/bin" go install github.com/go-delve/delve/cmd/dlv@latest
  fi
fi

if [[ "${DATA_IMAGE}" != "local" ]]; then
  # Extract data from data container image
  extract_from_image "${DATA_IMAGE}" "/stackrox-data" "${bundle_root}/stackrox/static-data/"
  extract_from_image "${BUILDER_IMAGE}" "/usr/local/bin/ldb" "${bundle_root}/usr/local/bin/ldb"
else
  cp -a "/stackrox-data" "${bundle_root}/stackrox/static-data/"
  cp "/usr/local/bin/ldb" "${bundle_root}/usr/local/bin/ldb"
fi

# Install all the required compression packages for RocksDB to compile
rpm_base_url="http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages"
rpm_suffix="el9.x86_64.rpm"

curl -s -f -o "${bundle_root}/snappy.rpm" "${rpm_base_url}/snappy-1.1.8-8.${rpm_suffix}"

# Install Postgres Client so central can initiate backups/restores
# Get postgres RPMs directly
postgres_major="14"
pg_rhel_version="9"
postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_version}-x86_64"
postgres_minor="14.2-1PGDG.rhel9.x86_64"

curl -sS --fail -o "${bundle_root}/postgres.rpm" \
    "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
curl -sS --fail -o "${bundle_root}/postgres-libs.rpm" \
    "${postgres_url}/postgresql${postgres_major}-libs-${postgres_minor}.rpm"

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
chmod -R u+w "${bundle_root}"
rm -r "${bundle_root}"
