#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <namespace optional>"
  exit 1
fi

namespace=${2:-stackrox}

workload_dir="${DIR}/workloads"
file="${workload_dir}/$1.yaml"
if [ ! -f "$file" ]; then
    >&2 echo "$file does not exist."
    >&2 echo "Options are:"
    >&2 echo "$(ls $workload_dir)"
    exit 1
fi

# This is purposefully kept as stackrox because this is where central should be run
if ! kubectl -n stackrox get pvc/central-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

# Create signature integrations to verify image signatures.
"${DIR}"/signatures/create-signature-integrations.sh

kubectl -n "${namespace}" delete deploy/admission-control || true
kubectl -n "${namespace}" delete daemonset collector || true

kubectl -n "${namespace}" set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0 ROX_FAKE_WORKLOAD_STORAGE=/var/cache/stackrox/pebble.db
kubectl -n "${namespace}" set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0  ROX_SCALE_TEST=true
kubectl -n "${namespace}" delete configmap scale-workload-config || true
kubectl -n "${namespace}" create configmap scale-workload-config --from-file=workload.yaml="$file"
kubectl -n "${namespace}" patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","volumeMounts":[{"name":"scale-workload-config","mountPath":"/var/scale/stackrox"}]}],"volumes":[{"name":"scale-workload-config","configMap":{"name": "scale-workload-config"}}]}}}}'

if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

if [[ -n "$CI" ]]; then
  kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
    kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"5"},"limits":{"memory":"16Gi","cpu":"8"}}}]}}}}'
  fi
else
  kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"55Gi","cpu":"2"},"limits":{"memory":"60Gi","cpu":"3"}}}]}}}}'
  kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
  if [[ "$ROX_POSTGRES_DATASTORE" == "true" ]]; then
    kubectl -n stackrox patch deploy/central-db -p '{"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'
  fi
fi
