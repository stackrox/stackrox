#!/usr/bin/env bash
# Creates a tgz bundle of all binary artifacts needed for scanner-db-rhel

set -euo pipefail
set -x

die() {
    echo >&2 "$@"
    exit 1
}

INPUT_ROOT="$1"
OUTPUT_DIR="$2"

[[ -n "$INPUT_ROOT" && -n "$OUTPUT_DIR" ]] \
    || die "Usage: $0 <input-root-dir> <output-dir>"
[[ -d "$INPUT_ROOT" ]] \
    || die "Input root directory doesn't exist or is not a directory."
[[ -d "$OUTPUT_DIR" ]] \
    || die "Output directory doesn't exist or is not a directory."

OUTPUT_BUNDLE="${OUTPUT_DIR}/bundle.tar.gz"

# Create tmp directory with stackrox directory structure
bundle_root="$(mktemp -d)"
mkdir -p "${bundle_root}/"{"usr/local/bin","etc","docker-entrypoint-initdb.d"}
chmod -R 755 "${bundle_root}"

# =============================================================================
# Get latest postgres minor version

postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-x86_64/pgdg-redhat-repo-latest.noarch.rpm"
postgres_major="14"
pg_rhel_version="8.5"

build_dir="$(mktemp -d)"
docker build -q -t postgres-minor-image "${build_dir}" -f - <<EOF
FROM registry.access.redhat.com/ubi8/ubi:${pg_rhel_version}
RUN dnf install -y "${postgres_repo_url}"
ENTRYPOINT dnf list -y postgresql${postgres_major}-server.x86_64 | tail -n 1 | awk '{print \$2}'
EOF
postgres_minor="$(docker run --rm postgres-minor-image).x86_64"
rm -rf "${build_dir}"

# =============================================================================

# Add files to be included in the Dockerfile here. This includes artifacts that
# would be otherwise downloaded or included via a COPY command in the
# Dockerfile.

cp -p "${INPUT_ROOT}"/*.conf "${bundle_root}/etc/"

# Get postgres RPMs directly
postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_version}-x86_64"
curl -s -o "${bundle_root}/postgres.rpm" \
    "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
curl -s -o "${bundle_root}/postgres-server.rpm" \
    "${postgres_url}/postgresql${postgres_major}-server-${postgres_minor}.rpm"
curl -s -o "${bundle_root}/postgres-libs.rpm" \
    "${postgres_url}/postgresql${postgres_major}-libs-${postgres_minor}.rpm"
curl -s -o "${bundle_root}/postgres-contrib.rpm" \
    "${postgres_url}/postgresql${postgres_major}-contrib-${postgres_minor}.rpm"

# =============================================================================

# Files should have owner/group equal to root:root
if tar --version | grep -q "gnu" ; then
  tar_chown_args=("--owner=root:0" "--group=root:0")
else
  tar_chown_args=("--uid=root:0" "--gid=root:0")
fi

# Create output bundle of all files in $bundle_root
tar cz "${tar_chown_args[@]}" --file "$OUTPUT_BUNDLE" --directory "${bundle_root}" .

# Create checksum
sha512sum "${OUTPUT_BUNDLE}" > "${OUTPUT_BUNDLE}.sha512"
sha512sum --check "${OUTPUT_BUNDLE}.sha512"

# Clean up after success
rm -r "${bundle_root}"

