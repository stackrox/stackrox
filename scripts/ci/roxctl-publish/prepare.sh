#!/usr/bin/env bash

# prepare.sh
# Prepares roxctl artifacts in a layout in which they should be stored in a remote GCS bucket.

set -euo pipefail

die() {
  echo >&2 "$@"
  exit 1
}

source_dir="${1:-}"
target_dir="${2:-}"

[[ -d "$source_dir" ]] || die "Source directory ${source_dir} does not exist"
[[ -d "$target_dir" ]] || die "Target directory ${target_dir} does not exist"

# Set up directory structure

mkdir "${target_dir}/bin"

for platform in Linux Darwin Windows; do
  platform_lower="$(echo "$platform" | tr A-Z a-z)"

  mkdir "${target_dir}/bin/${platform}"
  mkdir "${target_dir}/bin/${platform_lower}"

  roxctl_bin="roxctl"
  if [[ "${platform}" == "Windows" ]]; then
    roxctl_bin="roxctl.exe"
  fi
  cp "${source_dir}/bin/${platform_lower}/${roxctl_bin}" "${target_dir}/bin/${platform}/${roxctl_bin}"
  cp "${source_dir}/bin/${platform_lower}/${roxctl_bin}" "${target_dir}/bin/${platform_lower}/${roxctl_bin}"
done

# Create sha256sum.txt checksum files

find "${target_dir}" -name "sha256sum.txt" -exec rm {} \;
while IFS='' read -r dir || [[ -n "$dir" ]]; do
  ( cd "$dir" ; sha256sum * >sha256sum.txt )
done < <(find "${target_dir}" -type f -print0 | xargs -0 -n 1 dirname | sort -u)
