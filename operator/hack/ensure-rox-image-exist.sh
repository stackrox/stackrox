#!/usr/bin/env bash
set -euo pipefail

# Fetch latest tags etc
git fetch origin

root_dir="$(git rev-parse --show-toplevel)"
image_registry="$(make --quiet -C "$root_dir" default-image-registry)"
main_image_tag="${MAIN_IMAGE_TAG:-"$(make --quiet -C "$root_dir" tag)"}"
main_image="${image_registry}/main:${main_image_tag}"

echo "Ensuring $main_image is available locally"

if [[ -n "$(docker images -q "$main_image")" ]]; then
  echo "Found image $main_image locally"
  exit 0
fi

echo "Trying to pull $main_image"
if ! docker pull "$main_image"; then
  echo "Could not pull $main_image, trying to build it."
  make -C "$root_dir" image
fi

echo "$main_image is now available locally"
