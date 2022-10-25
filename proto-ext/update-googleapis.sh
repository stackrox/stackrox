#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
rm -rf "${SCRIPT_DIR}/google"

tmpdir="$(mktemp -d)"

git clone --depth 1 --single-branch https://github.com/googleapis/googleapis "$tmpdir"

mkdir -p "${SCRIPT_DIR}/google/api"
cp "${tmpdir}/google/api"/*.proto "${SCRIPT_DIR}/google/api/"

rm -rf "$tmpdir"
