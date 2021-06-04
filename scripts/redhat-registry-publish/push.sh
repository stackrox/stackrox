#!/usr/bin/env bash

# push.sh <main version> <scanner version> <collector version>
#
# Pushes (locally) retagged StackRox images to the quay.io registry, to be served from registry.redhat.io.
#
# Important: this script runs in dry-run mode by default. You MUST set DRY_RUN=false in the environment
# if you want it to actually push images.
#

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=./common.sh
source "${DIR}/common.sh"

main_version="${1:-}"
scanner_version="${2:-}"
collector_version="${3:-}"

# Check that the versions are consistent with (a) the repository contents and
# (b) the scanner/collector versions referenced in the main image.
version_check "${main_version}" "${scanner_version}" "${collector_version}"

# Compute all the destination images on quay.io

images=(
  "main:${main_version}"
  "roxctl:${main_version}"
  "scanner:${scanner_version}"
  "scanner-db:${scanner_version}"
  "collector:${collector_version}-slim"
  "collector:${collector_version}-latest"
)

dst_imgs=()

for img in "${images[@]}"; do
  dst="$(dst_img "$img")"
  docker image inspect "$dst" >/dev/null
  dst_imgs+=("$dst")
done

echo "Pushing the following images:"
printf " - %s\n" "${dst_imgs[@]}"

push_cmd=("${DIR}/../ci/push-as-manifest-list.sh")
if [[ "${DRY_RUN:-}" != "false" ]]; then
  echo
  echo "==================================================================================="
  echo " DRY RUN - NOT PUSHING ANYTHING                                                    "
  echo " You usually should not have to run this command locally. It is meant for CI only. "
  echo " If you DO need to push locally, invoke this script with DRY_RUN=false set.        "
  echo "==================================================================================="
  echo
  push_cmd=(echo "${push_cmd[@]}")
fi

# Try to pull an existing image. This just verifies that we have pull access to the registry,
# otherwise the image existence check is not reliable.
docker pull "quay.io/rhacs/rh-acs----roxctl:3.0.58.1" &>/dev/null ||
  die "Pulling from quay.io/rhacs does not work, are you logged in?"

for img in "${dst_imgs[@]}"; do
  # Check if the image exists already. We must not override existing tags, otherwise customers
  # might see signature validation fail.
  if docker pull "$img" &>/dev/null; then
    echo "Image $img exists on the target registry -- not pushing!"
  else
    "${push_cmd[@]}" "$img"
  fi
done
