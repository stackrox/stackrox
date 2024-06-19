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

# This will attempt to collect kube API server audit logs on OpenShift.
# It would be great to do the same on other cluster types but that would be much harder do in a portable way.
echo "$(date) Attempting to collect kube API server audit logs"
(cd "${log_dir}" && oc version && oc adm must-gather --timeout=7m -- /usr/bin/gather_audit_logs && du -sh must-gather*) || true
