#!/usr/bin/env bash
set -e

# Enable new event pipeline without re-sync for k8s deployments
export ROX_RESYNC_DISABLED="true"

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
# shellcheck source=/dev/null
source "${K8S_DIR}/central.sh"
# shellcheck source=/dev/null
source "${K8S_DIR}/sensor.sh"
