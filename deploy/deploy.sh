#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
source "${DIR}/detect.sh"

if is_openshift; then
    "${DIR}/openshift/deploy.sh"
else
    "${DIR}/k8s/deploy.sh"
fi
