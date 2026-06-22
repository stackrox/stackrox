#!/usr/bin/env bash

# Verify that roxctl CLI binaries in the main image are statically linked.
# Dynamically linked roxctl binaries break downstream consumers that run
# them on Alpine or older glibc distros.
# This script must be called from the Makefile which sets TAG.

set -euo pipefail

image="stackrox/main:${TAG}"
container=$(docker create "$image")

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"; docker rm "$container" &>/dev/null' EXIT

docker cp "$container":/assets/downloads/cli "$tmpdir/cli"

failed=0
checked=0
for bin in "$tmpdir"/cli/roxctl-linux-*; do
  name=$(basename "$bin")
  [[ "$name" == "roxctl-linux" ]] && continue
  out=$(file -b "$bin")
  if echo "$out" | grep -q "ELF"; then
    checked=$((checked + 1))
    if echo "$out" | grep -q "dynamically linked"; then
      echo >&2 "FAIL: /assets/downloads/cli/$name is dynamically linked"
      echo >&2 "  $out"
      failed=1
    else
      echo "OK: /assets/downloads/cli/$name is statically linked"
    fi
  else
    echo "SKIP: /assets/downloads/cli/$name is not an ELF binary (stub)"
  fi
done

if [[ "$failed" -eq 1 ]]; then
  echo >&2 "roxctl CLI binaries must be statically linked for portability."
  exit 1
fi

if [[ "$checked" -eq 0 ]]; then
  echo "No ELF roxctl binaries found (CLI build was likely skipped). Skipping check."
fi
