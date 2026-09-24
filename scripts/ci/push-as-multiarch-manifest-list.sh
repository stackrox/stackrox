#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/lib.sh"

set -euo pipefail

[[ "$#" == 2 ]] || die "Usage: $0 <image> <csv of architectures>"

image="$1"
IFS=',' read -ra architectures <<< "$2"

[[ -n "$image" ]] || die "No image specified"
[[ "$image" == *:* ]] || die "Must specify a tagged image reference when using this script"

image_list=()
for arch in "${architectures[@]}"
do
    arch_image="${image}-${arch}"
    # No pull needed - docker manifest create works with remote references
    image_list+=("$arch_image")
done

# --amend: this script is retried on failure (see push_*_manifest_lists in
# scripts/ci/lib.sh), and "docker manifest create" errors out if a local
# manifest list of the same name already exists, which it would on a retry.
docker manifest create --amend "$image" "${image_list[@]}"

# Try pushing manifest a few times for the case when quay.io has issues
pushed=0
for i in {1..5}; do
  echo "Pushing manifest for ${image}. Attempt ${i}..."
  echo docker manifest push "$image"
  if docker manifest push "$image"; then
    pushed=1
    break
  fi
  sleep 10
done
(( pushed ))
