#!/bin/sh
set -eu

# Collect Openshift must-gather information
#
# Extracts and bundles cluster and workload information from the given Openshift cluster and saves it for
# future examination.
#
# Usage:
#   run-must-gather.sh [<output-dir>]
#
# Example:
# $ ./scripts/ci/run-must-gather.sh /tmp/my-bundle
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/ocp-must-gather by default


if [ $# -gt 0 ]; then
    log_dir="$1"
else
    log_dir="/tmp/ocp-must-gather"
fi

echo "$(date) Attempting to gather debugging information from an OpenShift cluster"
oc adm must-gather --dest-dir "${log_dir}"
