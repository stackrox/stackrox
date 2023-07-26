#!/usr/bin/env bash
# Creates a scripts directory and a tgz bundle of binaries and data files
# needed for main-rhel

set -euo pipefail

die() {
    echo >&2 "$@"
    exit 1
}

INPUT_ROOT="${1:-}"
OUTPUT_DIR="${2:-}"

[[ -n "$INPUT_ROOT" && -n "$OUTPUT_DIR" ]] \
    || die "Usage: $0 <input-root-directory> <builder-image> <output-directory>"
[[ -d "$INPUT_ROOT" ]] \
    || die "Input root directory doesn't exist or is not a directory."
[[ -d "$OUTPUT_DIR" ]] \
    || die "Output directory doesn't exist or is not a directory."

OUTPUT_BUNDLE="${OUTPUT_DIR}/bundle.tar.gz"

# Create tmp directory with stackrox directory structure
bundle_root="$(mktemp -d)"
mkdir -p "${bundle_root}"/{assets/downloads/cli,stackrox/bin,ui,usr/local/bin}
chmod -R 755 "${bundle_root}"

# =============================================================================
# Copy binaries and data files into bundle

# Add binaries and data files to be included in the Dockerfile here. This
# includes artifacts that would be otherwise downloaded or included via a COPY
# command in the Dockerfile.

cp -pr "${INPUT_ROOT}/THIRD_PARTY_NOTICES"  "${bundle_root}/"
cp -pr "${INPUT_ROOT}/ui/build/"*           "${bundle_root}/ui/"

arch="x86_64"
goarch="amd64"
if [[ $(uname -m) == "arm64" ]]; then
  arch="aarch64"
  goarch="arm64"
fi

mkdir -p "${bundle_root}/go/bin"
if [[ "$DEBUG_BUILD" == "yes" ]]; then
  if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    GOBIN= GOOS=linux GOARCH="${goarch}" GOPATH="${bundle_root}/go" go install github.com/go-delve/delve/cmd/dlv@latest
    mv "${bundle_root}/go/bin/linux_${goarch}/dlv" "${bundle_root}/go/bin/dlv"
    rm -r "${bundle_root}/go/bin/linux_${goarch}"
  else
    GOBIN="${bundle_root}/go/bin" go install github.com/go-delve/delve/cmd/dlv@latest
  fi
fi

# Install all the required compression packages for RocksDB to compile
rpm_base_url="http://mirror.centos.org/centos/8-stream/BaseOS/${arch}/os/Packages"
rpm_suffix="el8.${arch}.rpm"

curl --retry 3 -s -f -o "${bundle_root}/snappy.rpm" "${rpm_base_url}/snappy-1.1.8-3.${rpm_suffix}"

# Install Postgres Client so central can initiate backups/restores
# Get postgres RPMs directly
postgres_major="13"
pg_rhel_major="8"
pg_rhel_minor="6"
pg_rhel_version="${pg_rhel_major}.${pg_rhel_minor}"
postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_major}-${arch}"
postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-${arch}/pgdg-redhat-repo-latest.noarch.rpm"

build_dir="$(mktemp -d)"
docker build -q -t postgres-minor-image "${build_dir}" -f - <<EOF
FROM registry.access.redhat.com/ubi8/ubi:${pg_rhel_version}
RUN dnf install --disablerepo='*' -y "${postgres_repo_url}"
ENTRYPOINT dnf list --disablerepo='*' --enablerepo=pgdg${postgres_major} -y postgresql${postgres_major}-server.$arch | tail -n 1 | awk '{print \$2}'
EOF

postgres_minor="$(docker run --rm postgres-minor-image).${arch}"
rm -rf "${build_dir}"

curl --retry 3 -sS --fail -o "${bundle_root}/postgres.rpm" \
    "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
curl --retry 3 -sS --fail -o "${bundle_root}/postgres-libs.rpm" \
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
