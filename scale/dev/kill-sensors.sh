#!/usr/bin/env bash
set -eoux pipefail


echo "Killing sensors"
mapfile -t sensor_namespaces < <(kubectl get namespaces -o custom-columns=:metadata.name | grep -E 'stackrox[0-9]+')
nnamespace="${#sensor_namespaces[@]}"
for ((i = 0; i < nnamespace; i = i + 1)); do
  kubectl annotate -n "${sensor_namespaces[i]}" securedclusters.platform.stackrox.io \
    stackrox-secured-cluster-services stackrox.io/pause-reconcile- || true # ignore if CR is missing
  roxie teardown secured-cluster --set securedCluster.namespace="${sensor_namespaces[i]}"
  kubectl delete namespace "${sensor_namespaces[i]}"
done
