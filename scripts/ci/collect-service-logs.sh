#!/bin/sh
set -eu

# Collect Service Logs script
#
# Extracts service logs from the given Kubernetes cluster and saves them off for
# future examination.
#
# Usage:
#   collect-service-logs.sh NAMESPACE [DIR]
#
# Example:
# $ ./scripts/ci/collect-service-logs.sh stackrox
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/ or DIR if passed

usage() {
    echo "./scripts/ci/collect-service-logs.sh <namespace> [<output-dir>]"
    echo "e.g. ./scripts/ci/collect-service-logs.sh stackrox"
}

dump_logs() {
    for ctr in $(kubectl -n "${namespace}" get "${object}" "${item}" -o jsonpath="{${jpath}[*].name}"); do
        kubectl -n "${namespace}" logs "${object}/${item}" -c "${ctr}" > "${log_dir}/${object}/${item}-${ctr}.log"
        exit_code="$(kubectl -n "${namespace}" get "${object}/${item}" -o jsonpath="{${jpath}}" | jq --arg ctr "$ctr" '.[] | select(.name == $ctr) | .lastState.terminated.exitCode')"
        if [ "${exit_code}" != "null" ]; then
            prev_log_file="${log_dir}/${object}/${item}-${ctr}-previous.log"
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
        echo "Skipping missing namespace"
        exit 0
    fi

    if [ $# -gt 1 ]; then
        log_dir="$2"
    else
        log_dir="/tmp/k8s-service-logs"
    fi
    log_dir="${log_dir}/${namespace}"
    mkdir -p "${log_dir}"

    echo
    echo ">>> Collecting from namespace ${namespace} <<<"
    echo
    set +e

    for object in daemonsets deployments services pods secrets serviceaccounts validatingwebhookconfigurations catalogsources subscriptions clusterserviceversions nodes; do
        # A feel good command before pulling logs
        echo ">>> ${object} <<<"
        out="$(mktemp)"
        if ! kubectl -n "${namespace}" get "${object}" -o wide > "$out" 2>&1; then
            echo "Cannot get $object in $namespace: $(cat "$out")"
            continue
        fi
        cat "$out"

        mkdir -p "${log_dir}/${object}"

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

            jpath='.status.containerStatuses'
            dump_logs

            jpath='.status.initContainerStatuses'
            dump_logs
        done
    done

    kubectl -n "${namespace}" get events -o wide >"${log_dir}/events.txt"
    kubectl -n "${namespace}" get events -o yaml >"${log_dir}/events.yaml"

    find "${log_dir}" -type f -size 0 -delete
}

main "$@"
