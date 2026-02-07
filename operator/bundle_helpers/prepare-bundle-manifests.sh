#!/bin/bash
set -euo pipefail

# Simple script to prepare the bundle build directory.
# This eliminates duplication between operator Makefile (GHA build) and Dockerfile (Konflux build).

# Always start with a clean bundle build directory.
mkdir -p build/
rm -rf build/bundle
cp -a bundle build/

# Call csv-patcher (Go replacement for patch-csv.py) with stdin/stdout
csv-patcher "$@" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
