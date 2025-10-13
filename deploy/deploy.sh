#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
# shellcheck source=./detect.sh
source "${DIR}/detect.sh"

if is_openshift; then
    source "${DIR}/openshift/deploy.sh"
else
    source "${DIR}/k8s/deploy.sh"
fi
