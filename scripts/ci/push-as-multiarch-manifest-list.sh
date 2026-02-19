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

# Try pushing manifest a few times for the case when quay.io has issues.
# Each attempt re-pulls and re-fetches arch manifests from the registry to
# avoid retrying with a stale or inconsistent manifest from a previous attempt.
pushed=0
for i in {1..5}; do
  echo "Pushing manifest for ${image}. Attempt ${i}..."

  image_list=()
  for arch in "${architectures[@]}"; do
    arch_image="${image}-${arch}"
    docker pull "${arch_image}"
    echo "=== Arch manifest inspect: ${arch_image} ==="
    docker manifest inspect "${arch_image}" || true
    echo "============================================="
    image_list+=("$arch_image")
  done

  # Clear any previously cached manifest to force a fresh fetch from the
  # registry on every attempt. docker manifest create caches each arch manifest
  # locally after the first fetch and --amend reuses that cache without
  # re-fetching. If the registry served a stale/inconsistent manifest on a
  # prior attempt, all subsequent retries would push the same invalid manifest
  # list without this rm.
  docker manifest rm "$image" 2>/dev/null || true
  docker manifest create --amend "$image" "${image_list[@]}"

  echo "=== Manifest list inspect: ${image} ==="
  docker manifest inspect "$image" || true
  echo "========================================"

  echo docker manifest push "$image"
  if docker manifest push "$image"; then
    pushed=1
    break
  fi
  sleep 10
done
(( pushed ))
