#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

usage() {
    echo "usage: ./deploy.sh <namespace>"
}

if [ $# -lt 1 ]; then
  usage
  exit 1
fi

namespace="$1"

count_running_pods() {
    kubectl -n "${namespace}" get pods --field-selector=status.phase=="Running" -o json | jq -j '.items | length'
}

wait_for_pod_count_to_be() {
    local pod_count="$1"
    local tries=0
    while [[ $(count_running_pods) -ne ${pod_count} ]]; do
        tries=$((tries + 1))
        if [[ ${tries} -gt 40 ]]; then
            kubectl get nodes -o yaml
            echo "Took too long to reach pod count ${pod_count}"
            exit 1
        fi
        echo "Waiting..."
        kubectl -n "${namespace}" get pods
        sleep 10
    done
    kubectl -n "${namespace}" get pods
}

if kubectl get ns "${namespace}"; then
    kubectl delete ns "${namespace}" # handle CI re-runs
fi
kubectl create ns "${namespace}"

export POSTGRES_PASSWORD="${CLAIR_V4_DB_PASSWORD}"
kubectl -n "${namespace}" create secret generic clairv4-config --from-file="${DIR}/config.yaml"
kubectl -n "${namespace}" create -f "${DIR}/clairv4-kubernetes.yaml"

wait_for_pod_count_to_be 2
