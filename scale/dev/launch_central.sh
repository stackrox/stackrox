#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

local_port=${1:-8000}

# This is GKE specific so we may need to be careful in the future
if kubectl version -o json | jq .serverVersion.gitVersion | grep "gke" > /dev/null; then
  echo "Setting storage class to SSD-based"
  extra_args=(--set central.spec.central.db.persistence.persistentVolumeClaim.storageClassName=premium-rwo)
fi

# Launch Central

roxie_config=$(mktemp)
trap 'rm -f "${roxie_config}"' EXIT
cat > "${roxie_config}" <<EOF
central:
  exposure: none
  portForwarding: true # Needed for roxie to health-check central, but otherwise ignored.
  spec:
    customize:
      envVars:
      - name: MUTEX_WATCHDOG_TIMEOUT_SECS
        value: "0"
      - name: ROX_SCALE_TEST
        value: "true"
    scannerV4:
      # ROX-29641: We are facing issues with workload generation on GKE when Scanner v4 is enabled.
      scannerComponent: Disabled
EOF

if [[ $(kubectl get nodes -o json | jq '.items | length') != 1 ]]; then
  "${DIR}/../../tests/e2e/lib-yaml.sh" merge_yaml "${roxie_config}" <<EOF
central:
  spec:
    central:
      resources:
        requests:
          memory: 16Gi
          cpu: "8"
        limits:
          memory: 16Gi
          cpu: "8"
      db:
        resources:
          requests:
            memory: 32Gi
            cpu: "16"
          limits:
            memory: 32Gi
            cpu: "16"
EOF
fi

roxie deploy central --single-namespace --verbose --envrc .roxie.env --config "${roxie_config}" "${extra_args[@]}"
rm -f "${roxie_config}"

$DIR/port-forward.sh "${local_port}"
