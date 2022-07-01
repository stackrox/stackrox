#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

SENSOR_HELM_DEPLOY=false STORAGE_CLASS=faster ./deploy/k8s/central.sh

# This is purposefully kept as stackrox because this is where central should be run
if ! kubectl -n stackrox get pvc/stackrox-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

# Create signature integrations to verify image signatures.
"${DIR}"/signatures/create-signature-integrations.sh

kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0 ROX_SCALE_TEST=true

if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

if [[ -n "$CI" ]]; then
  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
    kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  fi
else
  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"24Gi","cpu":"8"}}}]}}}}'
  if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
    kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"16Gi","cpu":"8"},"limits":{"memory":"24Gi","cpu":"8"}}}]}}}}'
  fi
fi

kubectl -n stackrox delete po -l app=central --grace-period=0
kubectl -n stackrox delete po -l app=central-db --grace-period=0

echo "Sleeping for 30 for Central to come up"
sleep 30
pf & > /dev/null
