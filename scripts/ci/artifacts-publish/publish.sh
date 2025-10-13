#!/usr/bin/env bash

# publish.sh
# Publishes artifacts from a local directory to a GCS bucket directory.

set -euxo pipefail

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
  # List all existing versions.
  # sed expression should extract and print version consisting of numbers and dots that goes as the last component of url.
  # E.g. "gs://rhacs-openshift-mirror-src/assets/3.0.59.1/" -> "3.0.59.1"
  gsutil ls "${gcs_target}/" | sed -En 's/^.*\/([[:digit:]]+(\.[[:digit:]]+)*)\/$/\1/p'
  # make sure the current version is included in the list
  echo "$version"
} | sort -rV | head -n 1)" # sort all (existing + new) versions in descending order and take the first element

if [[ "${latest_version}" == "${version}" ]]; then
  gsutil -m rsync -r "${source_dir}/" "${gcs_target}/latest/"
fi
