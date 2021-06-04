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

docker push "$arch_image"
docker manifest create "$image" "$arch_image"

docker manifest push "$image"
