#!/usr/bin/env bash

# retag.sh <main version> <scanner version> <collector version>
#
# Pulls images from stackrox.io, and retags them locally according to the quay registry naming scheme.

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=./common.sh
source "${DIR}/common.sh"

main_version="${1:-}"
scanner_version="${2:-}"
collector_version="${3:-}"

version_check "$main_version" "$scanner_version" "$collector_version"

images=(
  "main:${main_version}"
  "roxctl:${main_version}"
  "scanner:${scanner_version}"
  "scanner-db:${scanner_version}"
  "collector:${collector_version}-slim"
  "collector:${collector_version}-latest"
)

src_imgs=()
dst_imgs=()

for img in "${images[@]}"; do
  src_img_ref="$(src_img "$img")"
  docker pull "${src_img_ref}"
  src_imgs+=("$src_img_ref")
  dst_imgs+=("$(dst_img "$img")")
done

for i in "${!src_imgs[@]}"; do
  printf "Retagging %-50s => %-50s\n" "${src_imgs[$i]}" "${dst_imgs[$i]}"
  docker tag "${src_imgs[$i]}" "${dst_imgs[$i]}"
done
