#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
# shellcheck source=./detect.sh
source "${DIR}/detect.sh"

if is_openshift; then
    "${DIR}/openshift/deploy-local.sh"
else
    "${DIR}/k8s/deploy-local.sh"
fi
