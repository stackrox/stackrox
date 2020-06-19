#!/bin/bash

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name>"
  exit 1
fi

if ! kubectl -n stackrox get pvc/stackrox-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

kubectl -n stackrox set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0
kubectl -n stackrox set env deploy/sensor ROX_FAKE_KUBERNETES_WORKLOAD="$1"
kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0 ROX_ROCKSDB=true
kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
