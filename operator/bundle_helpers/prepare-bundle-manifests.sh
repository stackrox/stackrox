#!/bin/bash
set -euo pipefail

# Simple script to prepare the bundle build directory.
# This eliminates duplication between operator Makefile (GHA buld) and Dockerfile (Konflux build).

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

# Always start with a clean bundle build directory.
mkdir -p build/
rm -rf build/bundle
cp -a bundle build/

go run "${script_dir}/main.go" patch-csv "$@" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
