#!/usr/bin/env bash
set -eoux pipefail


echo "Killing sensors"
mapfile -t sensor_namespaces < <(kubectl get namespaces -o custom-columns=:metadata.name | grep -E 'stackrox[0-9]+')
nnamespace="${#sensor_namespaces[@]}"
for ((i = 0; i < nnamespace; i = i + 1)); do
  kubectl delete namespace "${sensor_namespaces[i]}"
done
