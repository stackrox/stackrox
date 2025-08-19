#!/bin/bash
set -euo pipefail

# Simple script to copy CRD and call patch-csv.py with all arguments passed through
# This eliminates duplication between Makefile and Dockerfile

# Ensure output directory exists and copy the securitypolicies CRD
mkdir -p build/bundle/manifests
cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/

# Call patch-csv.py with input and output files, passing through all arguments
"$(dirname "$0")/patch-csv.py" "$@" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml