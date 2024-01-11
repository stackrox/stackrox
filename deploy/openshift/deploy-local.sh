#!/usr/bin/env bash
set -e

export MONITORING_SUPPORT=false
export LOCAL_DEPLOYMENT="true"

OPENSHIFT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
${OPENSHIFT_DIR}/deploy.sh
