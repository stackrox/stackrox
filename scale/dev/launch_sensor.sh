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

roxie_config=$(mktemp)
trap 'rm -f "${roxie_config}"' EXIT

CENTRAL_ENDPOINT=${CENTRAL_ENDPOINT:-central.stackrox.svc:443}

cat > "${roxie_config}" <<EOF
securedCluster:
  namespace: "${namespace}"
  pauseReconciliation: true  # to be able to delete deployments below
  metadata:
    annotations:
      platform.stackrox.io/namespace-prefix-global-resources: "true"
  spec:
    clusterName: "${namespace}"
    centralEndpoint: "${CENTRAL_ENDPOINT}"
    scannerV4:
      scannerComponent: Disabled
    customize:
      envVars:
      - name: MUTEX_WATCHDOG_TIMEOUT_SECS
        value: "0"
      - name: ROX_FAKE_WORKLOAD_STORAGE
        value: "/var/cache/stackrox/pebble.db"
    overlays:
    - apiVersion: apps/v1
      kind: Deployment
      name: sensor
      patches:
      - path: spec.template.spec.containers[name:sensor].volumeMounts[-1]
        value: |
          name: scale-workload-config
          mountPath: /var/scale/stackrox
      - path: spec.template.spec.volumes[-1]
        value: |
          name: scale-workload-config
          configMap:
            name: scale-workload-config
EOF

if [[ $(kubectl get nodes -o json | jq '.items | length') != 1 ]]; then
  "${DIR}/../../tests/e2e/lib-yaml.sh" merge_yaml "${roxie_config}" <<EOF
securedCluster:
  spec:
    sensor:
      resources:
          requests:
            memory: 20Gi
            cpu: "4"
          limits:
            memory: 20Gi
            cpu: "8"
EOF
fi
kubectl get namespace "${namespace}" || kubectl create namespace "${namespace}"
kubectl -n "${namespace}" delete configmap scale-workload-config || true
kubectl -n "${namespace}" create configmap scale-workload-config --from-file=workload.yaml="$file"

API_ENDPOINT="localhost:${LOCAL_PORT:-8000}" roxie deploy secured-cluster --config "${roxie_config}" --verbose --early-readiness

kubectl -n "${namespace}" delete deploy/admission-control || true
kubectl -n "${namespace}" delete daemonset collector || true
