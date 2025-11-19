#!/bin/bash
set -euo pipefail

# Simple script to prepare the bundle build directory.
# This eliminates duplication between operator Makefile (GHA buld) and Dockerfile (Konflux build).

# Always start with a clean bundle build directory.
mkdir -p build/
rm -rf build/bundle
cp -a bundle build/

cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/

"$(dirname "$0")/patch-csv.py" "$@" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
