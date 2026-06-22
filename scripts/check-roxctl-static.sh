#!/usr/bin/env bash

# Verify that roxctl CLI binaries in the main image are statically linked.
# Dynamically linked roxctl binaries break downstream consumers that run
# them on Alpine or older glibc distros.
# This script must be called from the Makefile which sets TAG.

set -euo pipefail

image="stackrox/main:${TAG}"
container=$(docker create "$image")
trap 'docker rm "$container" &>/dev/null' EXIT

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"; docker rm "$container" &>/dev/null' EXIT

docker cp "$container":/assets/downloads/cli "$tmpdir/cli"

failed=0
for bin in "$tmpdir"/cli/roxctl-linux-*; do
  name=$(basename "$bin")
  [[ "$name" == "roxctl-linux" ]] && continue
  out=$(file -b "$bin")
  if echo "$out" | grep -q "dynamically linked"; then
    echo >&2 "FAIL: /assets/downloads/cli/$name is dynamically linked"
    echo >&2 "  $out"
    failed=1
  elif echo "$out" | grep -q "statically linked"; then
    echo "OK: /assets/downloads/cli/$name is statically linked"
  elif echo "$out" | grep -q "ELF"; then
    echo >&2 "WARN: /assets/downloads/cli/$name has unknown linking: $out"
  fi
done

if [[ "$failed" -eq 1 ]]; then
  echo >&2 "roxctl CLI binaries must be statically linked for portability."
  exit 1
fi
