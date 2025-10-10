#!/usr/bin/env bash

set -euo pipefail

# Collect Service Logs script
#
# Extracts service logs from the given Kubernetes cluster and saves them off for
# future examination.
#
# Usage:
#   collect-service-logs.sh NAMESPACE DIR
#
# Example:
# $ ./scripts/ci/collect-service-logs.sh stackrox /tmp/some-directory
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Requires a two arguments: namespace name, and path to save logs to.
# - Creates the path if missing.

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

usage() {
    echo "./scripts/ci/collect-service-logs.sh <namespace> <output-dir>"
    echo "e.g. ./scripts/ci/collect-service-logs.sh stackrox /tmp/some-directory"
}

dump_logs() {
    local json_path="$1"
    for ctr in $(kubectl -n "${namespace}" get "${object}" "${item}" -o jsonpath="{${json_path}[*].name}"); do
        info "Dumping log of ${object}/${item} current container ${ctr} in ${namespace}..."
        kubectl -n "${namespace}" logs "${object}/${item}" -c "${ctr}" > "${log_dir}/${object}/${item}-${ctr}.log"
        exit_code="$(kubectl -n "${namespace}" get "${object}/${item}" -o jsonpath="{${json_path}}" | jq --arg ctr "$ctr" '.[] | select(.name == $ctr) | .lastState.terminated.exitCode')"
        if [ "${exit_code}" != "null" ]; then
            prev_log_file="${log_dir}/${object}/${item}-${ctr}-previous.log"
            info "Dumping log of ${object}/${item} previous container ${ctr} in ${namespace}..."
            if kubectl -n "${namespace}" logs "${object}/${item}" -p -c "${ctr}" > "${prev_log_file}"; then
                if [ "$exit_code" -eq "0" ]; then
                    mv "${prev_log_file}" "${log_dir}/${object}/${item}-${ctr}-prev-success.log"
                fi
            fi
        fi
    done
}

main() {
    namespace="$1"
    if [ -z "${namespace}" ]; then
        usage
        exit 1
    fi

    if ! kubectl get ns "${namespace}"; then
        info "Skipping missing namespace ${namespace}"
        exit 0
    fi

    if [ $# -lt 2 ]; then
        die "Usage: $0 <namespace> <path-to-collect-logs-to>"
    else
        log_dir="$2"
    fi
    log_dir="${log_dir}/${namespace}"
    mkdir -p "${log_dir}"

    info ">>> Collecting from namespace ${namespace} <<<"
    set +e

    for object in daemonsets deployments services pods secrets serviceaccounts validatingwebhookconfigurations \
      catalogsources subscriptions clusterserviceversions central securedclusters nodes; do
        # A feel good command before pulling logs
        info ">>> Collecting ${object} from namespace ${namespace} <<<"
        out="$(mktemp)"
        if ! kubectl -n "${namespace}" get "${object}" -o wide > "$out" 2>&1; then
            info "Cannot get $object in $namespace: $(cat "$out")"
            continue
        fi
        cat "$out"

        mkdir -p "${log_dir}/${object}"
        local item_count=0

        for item in $(kubectl -n "${namespace}" get "${object}" -o jsonpath='{.items}' | jq -r '.[] | select(.metadata.deletionTimestamp | not) | .metadata.name'); do
            kubectl get "${object}" "${item}" -n "${namespace}" -o json > "${log_dir}/${object}/${item}_object.json" 2>&1
            {
              kubectl describe "${object}" "${item}" -n "${namespace}" 2>&1
              echo
              echo
              echo '----------------------'
              echo '# Full YAML definition'
              kubectl get "${object}" "${item}" -n "${namespace}" -o yaml 2>&1
            } > "${log_dir}/${object}/${item}_describe.log"

            dump_logs '.status.containerStatuses'
            dump_logs '.status.initContainerStatuses'
            (( item_count++ ))
        done

        # Save the count of objects found in order to couple this script with
        # other functionality that expects filenames in these directories to
        # follow a certain pattern.
        echo "${item_count}" > "${log_dir}/${object}/ITEM_COUNT.txt"
    done

    kubectl -n "${namespace}" get events -o wide >"${log_dir}/events.txt"
    kubectl -n "${namespace}" get events -o yaml >"${log_dir}/events.yaml"

    find "${log_dir}" -type f -size 0 -delete
}

main "$@"
