#!/usr/bin/env bash
set -eoux pipefail


echo "Killing sensors"
if [[ -n ${1:-} ]]; then
  mapfile -t sensor_namespaces < <(for i in $(seq 1 $1); do echo "stackrox${i}"; done)
else
  mapfile -t sensor_namespaces < <(kubectl get namespaces -o custom-columns=:metadata.name | grep -E 'stackrox[0-9]+')
fi
nnamespace="${#sensor_namespaces[@]}"
for ((i = 0; i < nnamespace; i = i + 1)); do
  roxie teardown secured-cluster --set securedCluster.namespace="${sensor_namespaces[i]}"
done
# The pods need a while to shut down which blocks namespace deletion,
# so deleting the namespaces separately speeds things up esp. when there are many.
for ((i = 0; i < nnamespace; i = i + 1)); do
  kubectl delete namespace "${sensor_namespaces[i]}"
done
