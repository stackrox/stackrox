#!/usr/bin/env bash
set -euo pipefail

# Fetch latest tags etc
git fetch origin

docker_repo="stackrox/main"
base_image="quay.io/rhacs-eng/main"
root_dir="$(git rev-parse --show-toplevel)"
main_image_tag=${MAIN_IMAGE_TAG:-"$(make -C "$root_dir" tag)"}
main_image="$base_image:$main_image_tag"

echo "Ensuring $base_image:$main_image_tag is available locally"

if [[ -n $(docker images -q "${docker_repo}:${main_image_tag}") ]]; then
  echo "Found image $main_image locally"
  exit 0
fi

echo "Trying to pull $main_image"
if ! docker pull "$main_image"; then
  echo "Could not pull $main_image, trying to build it."
  make -C "$root_dir" image
fi

echo "$main_image is now available locally"
