#!/usr/bin/env bash
set -e

export MONITORING_SUPPORT=false

OPENSHIFT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
"${OPENSHIFT_DIR}"/deploy.sh
