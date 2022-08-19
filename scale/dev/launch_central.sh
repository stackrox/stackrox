#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export STORAGE=pvc
export STORAGE_SIZE=100
# This is GKE specific so we may need to be careful in the future
if kubectl version -o json | jq .serverVersion.gitVersion | grep "gke" > /dev/null; then
  echo "Setting storage class to faster"
  export STORAGE_CLASS=faster
fi

# Launch Central
$DIR/../../deploy/k8s/central.sh

kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0  ROX_SCALE_TEST=true
if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
  kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"32Gi","cpu":"16"},"limits":{"memory":"32Gi","cpu":"16"}}}]}}}}'
fi

$DIR/port-forward.sh 8000
