#!/bin/bash
set -euo pipefail

# Simple script to copy CRD and call patch-csv.py with all arguments passed through
# This eliminates duplication between Makefile and Dockerfile

cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/

"$(dirname "$0")/patch-csv.py" "$@" \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
