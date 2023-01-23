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

if [[ -z "${CLAIR_DB_PASSWORD}" ]]; then
    echo "CLAIR_DB_PASSWORD is required."
    exit 1
fi

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
if [[ "$POD_SECURITY_POLICIES" != "false" ]]; then
    kubectl -n "${namespace}" apply -f "${DIR}/psp.yaml"
fi

export POSTGRES_PASSWORD="${CLAIR_DB_PASSWORD}"
kubectl -n "${namespace}" create secret generic clairsecret --from-file="${DIR}/config.yaml"
kubectl -n "${namespace}" create -f "${DIR}/clair-kubernetes.yaml"

wait_for_pod_count_to_be 2

kubectl -n "${namespace}" scale --replicas 0 deployment clair

wait_for_pod_count_to_be 1

# Seed the DB with nginx:1.12.1 data

kubectl -n "${namespace}" port-forward "svc/postgres" 5432:5432 > /dev/null &
PID=$!

sleep 5

gsutil cat gs://stackrox-ci/clair-1-2-4-nginx-1-12-1.sql | psql -h localhost -U postgres

kill ${PID}
wait

kubectl -n "${namespace}" scale --replicas 1 deployment clair

wait_for_pod_count_to_be 2
