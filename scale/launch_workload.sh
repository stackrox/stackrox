#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name>"
  exit 1
fi

workload_dir="${DIR}/workloads"
file="${workload_dir}/$1.yaml"
if [ ! -f "$file" ]; then
    >&2 echo "$file does not exist."
    >&2 echo "Options are:"
    >&2 echo "$(ls $workload_dir)"
    exit 1
fi

if ! kubectl -n stackrox get pvc/stackrox-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

kubectl -n stackrox delete daemonset collector

kubectl -n stackrox set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0
kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0
kubectl -n stackrox delete configmap scale-workload-config || true
kubectl -n stackrox create configmap scale-workload-config --from-file=workload.yaml="$file"
kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","volumeMounts":[{"name":"scale-workload-config","mountPath":"/var/scale/stackrox"}]}],"volumes":[{"name":"scale-workload-config","configMap":{"name": "scale-workload-config"}}]}}}}'

if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

#if [[ -n "$CI" ]]; then
  kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
#else
#  kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
#  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
#  kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
#fi
