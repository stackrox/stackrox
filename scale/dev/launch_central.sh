#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

$DIR/../../deploy/k8s/central.sh

# This is purposefully kept as stackrox because this is where central should be run
if ! kubectl -n stackrox get pvc/stackrox-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0  ROX_SCALE_TEST=true
if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
  kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"32Gi","cpu":"16"},"limits":{"memory":"32Gi","cpu":"16"}}}]}}}}'
fi

./port-forward.sh 8000
