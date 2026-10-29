#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

# Set common local deployment defaults
export COLLECTION_METHOD="${COLLECTION_METHOD:-core_bpf}"
export MONITORING_SUPPORT="${MONITORING_SUPPORT:-false}"
export POD_SECURITY_POLICIES="${POD_SECURITY_POLICIES:-false}"
export LOCAL_DEPLOYMENT="true"

# Forward to roxie-based deploy.sh
"${DIR}/deploy.sh" "$@"
