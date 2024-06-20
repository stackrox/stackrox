#!/usr/bin/env bash

# get sensor pod name
sensor_pod_name=$(kubectl -n stackrox get pods -l app=sensor -o custom-columns=NAME:.metadata.name | tail -n 1)

# Create core dump
kubectl exec ${sensor_pod_name} -- /bin/bash -c "cd /var/cache/stackrox && /bin/bash -c 'gcore 1'"

# Copy core-dump
kubectl cp ${sensor_pod_name}:/var/cache/stackrox/core.1 ./core.1
