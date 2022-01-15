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
pushed=false
for i in {1..5}; do
  if docker push "$arch_image"; then
    pushed=true
    break
  else
    sleep 10
  fi
done
[[ "$pushed" == "true" ]]

docker manifest create "$image" "$arch_image"

docker manifest push "$image"
