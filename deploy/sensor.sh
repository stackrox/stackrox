#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
source "${DIR}/detect.sh"

if is_openshift; then
    "${DIR}/openshift/sensor.sh"
else
    "${DIR}/k8s/sensor.sh"
fi
