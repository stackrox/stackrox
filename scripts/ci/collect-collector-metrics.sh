#!/usr/bin/env bash
set -eu

# Gather collector metrics script

usage() {
    echo "$0 <namespace> <output-dir> [<pod-port> [<path>]]"
    echo "e.g. $0 stackrox /logs 9090 metrics"
}

die() {
    echo >&2 "$@"
    exit 1
}

main() {
    service="collector"

    if [ $# -eq 0 ]; then
        usage
        die "Please specify the namespace argument."
    else
        namespace="$1"
    fi

    if [ $# -lt 2 ]; then
        usage
        die "Please specify the metrics dir argument."
    else
        metrics_dir="$2"
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
    set +e

    local_port=9090
    local="localhost:${local_port}"

    pods="$(kubectl -n "$namespace" get pods -l "app=$service" --no-headers -o custom-columns=:metadata.name)"
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
