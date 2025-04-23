#!/bin/sh
set -eu
set -x

# Collect k8s infrastructure Logs script
#
# Extracts infrastructure logs from the given Kubernetes cluster and saves them for
# future examination.
#
# Usage:
#   collect-infrastructure-logs.sh [<output-dir>]
#
# Example:
# $ ./scripts/ci/collect-infrastructure-logs.sh
#
# Assumptions:
# - Must be called from the root of the Apollo git repository.
# - Logs are saved under /tmp/k8s-service-logs/ by default


if [ $# -gt 0 ]; then
    log_dir="$1"
else
    log_dir="/tmp/k8s-service-logs"
fi

# This will attempt to collect kube API server audit logs on OpenShift.
# It would be great to do the same on other cluster types but that would be much harder do in a portable way.
echo "$(date) Attempting to collect kube API server audit logs"
(cd "${log_dir}" && oc version && oc adm must-gather --timeout=7m -- /usr/bin/gather_audit_logs && du -sh must-gather*) || true

echo "$(date) Attempting to collect kube API server log"
kubectl proxy &
proxy_pid=$!

sleep 5 # Let kubectl proxy stabilize
mkdir -p "${log_dir}"/infrastructure
# exit code 18 = download incomplete
# last retry wait 2^8=256s
# -C -  : continue if retrying an interrupted download
curl -s http://localhost:8001/logs/kube-apiserver.log > "${log_dir}"/infrastructure/kube-apiserver.log \
  || curl -v --retry 8 --retry-max-time 30 -C - -s http://localhost:8001/logs/kube-apiserver.log -o "${log_dir}"/infrastructure/kube-apiserver.log \
  || curl -v --retry 3 -C - -s http://localhost:8001/logs/kube-apiserver.log -o "${log_dir}"/infrastructure/kube-apiserver.log \
  || curl -v --retry 3 -s http://localhost:8001/logs/kube-apiserver.log -o "${log_dir}"/infrastructure/kube-apiserver.log
  || curl -v --ignore-content-length --retry 3 -s http://localhost:8001/logs/kube-apiserver.log -o "${log_dir}"/infrastructure/kube-apiserver.log

kill $proxy_pid || true

echo "$(date) Attempting to collect kube API server metrics"
# Ref https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/
kubectl get --raw /metrics > "${log_dir}"/infrastructure/kube-apiserver-metrics.txt
