#!/usr/bin/env bash
set -eoux pipefail

artifacts_dir=$1
num_ports=$2
num_per_second=$3
num_concurrent=$4
num_pods=$5

DIR="$(cd "$(dirname "$0")" && pwd)"

start_deployment() {
    echo "Starting deployment"
    kubectl delete deployment open-close-ports-load || true
    kubectl delete configmap open-close-ports-load-configmap || true
    kubectl create configmap open-close-ports-load-configmap --from-literal=num_ports="$num_ports" --from-literal=num_per_second="$num_per_second" --from-literal=num_concurrent="$num_concurrent"
    kubectl create -f "$DIR"/deployment.yml
    kubectl scale deployment/open-close-ports-load --replicas="$num_pods"

    sleep 2
}

wait_for_pod() {
    echo "Waiting for pod"
    while true; do
        nterminating="$({ kubectl get pod | grep Terminating || true; } | wc -l)"
        if [[ "$nterminating" == 0 ]]; then
	    break
        fi
    done

    while true; do
        ready_containers="$(kubectl get pod -o jsonpath='{.items[*].status.containerStatuses[?(@.ready == true)]}')"
        not_ready_containers="$(kubectl get pod -o jsonpath='{.items[*].status.containerStatuses[?(@.ready == false)]}')"
        if [[ -n "$ready_containers" && -z "$not_ready_containers" ]]; then
            echo "All pods are running"
            break
        fi
        sleep 1
    done
}

export KUBECONFIG="$artifacts_dir"/kubeconfig

start_deployment
wait_for_pod
echo "Open close ports load started"
