#!/usr/bin/env bash

# prepare-cli.sh
# Prepares cli artifacts in a layout in which they should be stored in a remote GCS bucket.

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

for platform_upper in Linux Darwin Windows; do
  platform_lower="$(echo "$platform_upper" | tr '[:upper:]' '[:lower:]')"

  for platform in platform_upper platform_lower; do
    mkdir "${target_dir}/bin/${platform}"

    for app in roxagent roxctl; do
      app_bin="${app}"
      if [[ "${platform_upper}" == "Windows" ]]; then
        app_bin="${app}.exe"
      fi

      # x86_64 binaries don't mention architecture for compatibility with existing users (and their scripts).
      cp "${source_dir}/bin/${platform_lower}_amd64/${app_bin}" "${target_dir}/bin/${platform}/${app_bin}"

      # Binaries for other architectures should mention arch. The suggestion is to do it in the filename:
      #   https://mirror.openshift.com/pub/rhacs/assets/<version>/<platform>/roxctl-<arch>[.filetype]
      # See https://issues.redhat.com/browse/ROX-14701.
      # We may later want to add binaries with explicit x86_64 architecture which would be roxctl-amd64[.exe].
      if [[ "${platform_upper}" == "Linux" ]]; then
        for arch in "arm64" "ppc64le" "s390x"; do
          cp "${source_dir}/bin/${platform_lower}_${arch}/${app_bin}" "${target_dir}/bin/${platform}/${app_bin}-${arch}"
        done
      fi

      if [[ "${platform_upper}" == "Darwin" ]]; then
        cp "${source_dir}/bin/${platform_lower}_arm64/${app_bin}" "${target_dir}/bin/${platform}/${app_bin}-arm64"
      fi
    done
  done
done

# Create sha256sum.txt checksum files

find "${target_dir}" -name "sha256sum.txt" -exec rm {} \;
while IFS='' read -r dir || [[ -n "$dir" ]]; do
  ( cd "$dir" ; sha256sum ./* >sha256sum.txt )
done < <(find "${target_dir}" -type f -print0 | xargs -0 -n 1 dirname | sort -u)
