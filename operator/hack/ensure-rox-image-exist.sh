#!/usr/bin/env bash
set -o pipefail

# Fetch latest tags etc
git fetch origin

docker_repo="stackrox/main"
base_image="docker.io/stackrox/main"
root_dir="$(git rev-parse --show-toplevel)"
main_image_tag="$(make -C "$root_dir" tag)"
main_image="$base_image:$main_image_tag"

echo "Ensuring $base_image:$main_image_tag is available locally"

docker images | grep "$docker_repo" | grep "$main_image_tag"
if [ $? -eq 0 ]; then
  echo "Found image $main_image locally"
  exit 0
fi

echo "Trying to pull $main_image"
docker pull "$main_image"
if [ $? -ne 0 ]; then
  echo "Could not pull $main_image, trying to build it."
  make -C "$root_dir" image
fi

echo "$main_image is now available locally"
