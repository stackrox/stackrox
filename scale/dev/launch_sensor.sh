#!/bin/bash
set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <namespace optional>"
  exit 1
fi

namespace=${2:-stackrox}

workload_dir="${DIR}/../workloads"
file="${workload_dir}/$1.yaml"
if [ ! -f "$file" ]; then
    >&2 echo "$file does not exist."
    >&2 echo "Options are:"
    >&2 echo "$(ls $workload_dir)"
    exit 1
fi

if ! kubectl -n stackrox get pvc/central-db > /dev/null; then
  >&2 echo "Running the scale workload requires a PVC"
  exit 1
fi

ROXIE_CONFIG=$(mktemp)
trap "rm -f $ROXIE_CONFIG" EXIT

cat > "$ROXIE_CONFIG" <<EOF
securedCluster:
  namespace: "$namespace"
  pauseReconciliation: true
  metadata:
    annotations:
      platform.stackrox.io/namespace-prefix-global-resources: "true"
  spec:
    clusterName: "$namespace"
    centralEndpoint: "central.stackrox.svc:443"
    scannerV4:
      scannerComponent: Disabled
EOF

roxie deploy secured-cluster --config "$ROXIE_CONFIG" --verbose --early-readiness

kubectl -n "${namespace}" delete deploy/admission-control || true
kubectl -n "${namespace}" delete daemonset collector || true

kubectl -n "${namespace}" set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0 ROX_FAKE_WORKLOAD_STORAGE=/var/cache/stackrox/pebble.db
kubectl -n "${namespace}" delete configmap scale-workload-config || true
kubectl -n "${namespace}" create configmap scale-workload-config --from-file=workload.yaml="$file"
kubectl -n "${namespace}" patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","volumeMounts":[{"name":"scale-workload-config","mountPath":"/var/scale/stackrox"}]}],"volumes":[{"name":"scale-workload-config","configMap":{"name": "scale-workload-config"}}]}}}}'

if [[ $(kubectl get nodes -o json | jq '.items | length') == 1 ]]; then
  exit 0
fi

kubectl -n "${namespace}" patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"20Gi","cpu":"4"},"limits":{"memory":"20Gi","cpu":"8"}}}]}}}}'
