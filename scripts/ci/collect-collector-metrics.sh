#!/usr/bin/env bash
# shellcheck disable=SC1091

set -eu

# Gather collector metrics script

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/tests/e2e/separate-clusters.sh"

usage() {
    echo "$0 <namespace> <output-dir> <pod-port> <path>"
    echo "e.g. $0 stackrox /logs 9090 metrics"
}

main() {
    service="collector"

    if [ $# -gt 0 ]; then
        namespace="$1"
    else
        namespace="stackrox"
    fi

    if [ $# -gt 1 ]; then
        metrics_dir="$2"
    else
        metrics_dir="/tmp/k8s-service-logs/$namespace/metrics"
    fi
    mkdir -p "${metrics_dir}"

    if [ $# -gt 2 ]; then
        pod_port="$3"
    else
        pod_port="9090"
    fi

    if [ $# -gt 3 ]; then
        metrics_path="$4"
    else
        metrics_path="metrics"
    fi

    if separate_clusters_test; then
        target_cluster "sensor"
    fi

    set +e

    local_port=9090
    local="localhost:${local_port}"

    pods="$(kubectl -n $namespace get pods -l app=$service --no-headers -o custom-columns=:metadata.name)"
    if [[ -z "${pods}" ]]; then
        echo "No pods found for $service service"
        exit 0
    fi

    for pod in ${pods}; do
        remote="${pod}:${pod_port}"
        metrics_file="${pod}.txt"
        nohup kubectl -n "$namespace" port-forward "$pod" "${local_port}:${pod_port}" >/dev/null &
        PID=$!
        trap 'kill -TERM ${PID}; wait ${PID}' TERM INT
        max_retries=5
        retries=1
        until curl --output /dev/null --silent --fail -k "${local}/${metrics_path}"; do
            echo -n '.'
            if ((retries==max_retries)); then
                kill ${PID}
                die "failed to collect metrics from $pod after $retries retries"
            else
                ((retries++))
            fi
            sleep 5
        done
        echo
        echo "set up port-forwarding from $remote to $local"
        curl --silent --fail -k "${local}/${metrics_path}" > "${metrics_dir}/${metrics_file}"
        echo "finished download ${metrics_file}"
        kill ${PID}
        echo "finished tear down of port-forwarding from $remote to $local"
    done
}

main "$@"
