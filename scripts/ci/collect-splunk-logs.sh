#!/usr/bin/env bash

set -euo pipefail

# Collect logs from as splunk pod daemon
#
# Usage:
#   collect-splunk-logs.sh NAMESPACE DIR

usage() {
    echo "./scripts/ci/collect-splunk-logs.sh <namespace> <output-dir>"
}

main() {
    if [[ "$#" -lt 2 ]]; then
        usage
        exit 1
    fi

    namespace="$1"
    log_dir="$2"

    if ! kubectl get ns "${namespace}"; then
        echo "Collection is skipped when the splunk namespace is missing: ${namespace}"
        exit 0
    fi

    mkdir -p "${log_dir}"

    splunk_pod="$(kubectl -n "${namespace}" get pods -o json \
      | jq -r '.items[] | select(.metadata.labels["app"]? | match("^splunk")) | .metadata.name')"

    kubectl -n "${namespace}" exec "${splunk_pod}" -- \
      sudo tar cf - --warning=no-file-changed /opt/splunk/var/log/splunk | \
      tar xf - --strip 5 -C "${log_dir}"

    echo "Splunk logs saved to ${log_dir}"
}

main "$@"
