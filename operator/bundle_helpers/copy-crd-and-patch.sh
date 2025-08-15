#!/bin/bash
set -euo pipefail

# Simple script to copy CRD and call patch-csv.py with all arguments passed through
# This eliminates duplication between Makefile and Dockerfile

# Copy the securitypolicies CRD to build/bundle/manifests/
mkdir -p build/bundle/manifests
cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/

# Execute patch-csv.py with all provided arguments
exec "$(dirname "$0")/patch-csv.py" "$@"