#!/usr/bin/env bash

# publish.sh
# Publishes roxctl artifacts from a local directory to a GCS bucket directory.

die() {
  echo >&2 "$@"
  exit 1
}

source_dir="${1:-}"
version="${2:-}"
gcs_target="${3:-}"

[[ -d "$source_dir" ]] || die "Source directory must exist"
[[ -n "$version" ]] || die "Version must be set"
[[ -n "$gcs_target" ]] || die "GCS target must be specified"

gsutil -m rsync -r "${source_dir}" "${gcs_target}/${version}/"

latest_version="$({
  # list all existing versions
  gsutil ls "${gcs_target}/" | grep '/$' | awk '{print $(NF - 1)}' | grep -E '^\d+(\.\d+)*$'
  # make sure the current version is included in the list
  echo "$version"
} | sort -rV | head -n 1)" # sort all (existing + new) versions in descending order and take the first element

if [[ "${latest_version}" == "${version}" ]]; then
  gsutil -m rsync -r "${source_dir}/" "${gcs_target}/latest/"
fi
