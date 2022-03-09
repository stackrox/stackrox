#!/usr/bin/env bash

set -euo pipefail

die() {
  echo >&2 "$@"
  exit 1
}

[[ "$#" == 1 ]] || die "Usage: $0 <image>"

image="$1"

[[ -n "$image" ]] || die "No image specified"
[[ "$image" == *:* ]] || die "Must specify a tagged image reference when using this script"

arch_image="${image}-amd64"
docker tag "$image" "$arch_image"

# Try pushing image a few times for the case when quay.io has issues such as "unknown blob"
pushed=0
for i in {1..5}; do
  echo "Pushing ${arch_image}. Attempt ${i}..."
  if docker push "$arch_image"; then
    pushed=1
    break
  fi
  sleep 10
done
(( pushed ))

docker manifest create "$image" "$arch_image"

# Try pushing manifest a few times for the case when quay.io has issues
pushed=0
for i in {1..5}; do
  echo "Pushing manifest for ${image}. Attempt ${i}..."
  if docker manifest push "$image"; then
    pushed=1
    break
  fi
  sleep 10
done
(( pushed ))
