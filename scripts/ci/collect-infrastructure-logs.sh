#!/bin/sh
set -eu

# Collect k8s infrastructure Logs script
#
# Extracts infrastructure logs from the given Kubernetes cluster and saves them for
# future examination.
#
# Usage:
#   collect-infrastructure-logs.sh
#
# Example:
# $ ./scripts/ci/collect-infrastructure-logs.sh
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/

kubectl proxy &
proxy_pid=$!

sleep 5 # Let kubectl proxy stabilize
mkdir -p /tmp/k8s-service-logs/infrastructure
curl -s http://localhost:8001/logs/kube-apiserver.log > /tmp/k8s-service-logs/infrastructure/kube-apiserver.log

kill $proxy_pid
