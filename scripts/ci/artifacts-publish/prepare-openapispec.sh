#!/usr/bin/env bash

# prepare.sh
# Prepares OpenAPI spec in a layout in which they should be stored in a remote GCS bucket.

set -euo pipefail

die() {
  echo >&2 "$@"
  exit 1
}

copy_swagger_json_from_image() {
  container_id="$(docker create "quay.io/stackrox-io/main:${version}")"
  docker cp "${container_id}:/stackrox/static-data/docs/api/v1/swagger.json" "${target_dir}/v1.swagger.json"
  docker cp "${container_id}:/stackrox/static-data/docs/api/v2/swagger.json" "${target_dir}/v2.swagger.json"
  docker rm "${container_id}" >/dev/null
}

create_checksum_files() {
  find "${target_dir}" -name "sha256sum.txt" -exec rm {} \;
  while IFS='' read -r dir || [[ -n "$dir" ]]; do
    ( cd "$dir" ; sha256sum ./* >sha256sum.txt )
  done < <(find "${target_dir}" -type f -print0 | xargs -0 -n 1 dirname | sort -u)
}

target_dir="${1:-}"
version="${2:-}"

[[ -d "$target_dir" ]] || die "Target directory ${target_dir} does not exist"
[[ -n "$version" ]] || die "Version not provided"

copy_swagger_json_from_image
create_checksum_files
