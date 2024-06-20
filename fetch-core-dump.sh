#!/usr/bin/env bash

# get sensor pod name
sensor_pod_name="$(kubectl -n stackrox get pods -l app=sensor -o=jsonpath='{.items[0].metadata.name}')"

# Create core dump
kubectl exec ${sensor_pod_name} -- /bin/bash -c "cd /var/cache/stackrox && /bin/bash -c 'gcore 1'"

# Copy core-dump
kubectl cp "${sensor_pod_name}:/var/cache/stackrox/core.1" "core-dump-${sensor_pod_name}"
