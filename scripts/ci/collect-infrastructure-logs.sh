#!/usr/bin/env bash
set -eu

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


SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

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
declare -a waiters

{ timeout 120s kubectl logs -l component=kube-apiserver -n kube-system --insecure-skip-tls-verify-backend=true --ignore-errors=true > "${log_dir}"/infrastructure/kube-apiserver-kubectl.log \
      2>&1; echo $?; ls -latr "${log_dir}"/infrastructure/kube-apiserver-kubectl.log; tail -3 "${log_dir}"/infrastructure/kube-apiserver-kubectl.log; } | sed 's/^/kubectl: /' &
waiters+=($!)

{ timeout 120s curl -s http://localhost:8001/logs/kube-apiserver.log > "${log_dir}"/infrastructure/kube-apiserver.log 2>&1 \
  || { echo "curl exitcode:$?"; echo 'Can we just ignore exitcode 18? downloaded log file tail:'; tail "${log_dir}"/infrastructure/kube-apiserver.log; } } &
waiters+=($!)

{ timeout 120s curl --retry 5 -s http://localhost:8001/logs/kube-apiserver.log -o "${log_dir}"/infrastructure/kube-apiserver-o.log 2>&1 \
  || { echo "curl exitcode:$?"; echo 'Can we just ignore exitcode 18? downloaded log file tail:'; tail "${log_dir}"/infrastructure/kube-apiserver-o.log; }; } | sed 's/^/-o: /' &
waiters+=($!)

{ timeout 120s curl -v \
    --retry 5 \
    -s http://localhost:8001/logs/kube-apiserver.log \
    -o "${log_dir}"/infrastructure/kube-apiserver-retry.log 2>&1; echo $?; } | sed 's/^/none: /' &
waiters+=($!)

{ timeout 120s curl -v \
    --retry 5 \
    --range 0-99999999,-99999999 \
    -s http://localhost:8001/logs/kube-apiserver.log \
    -o "${log_dir}"/infrastructure/kube-apiserver-range.log 2>&1; echo $?; } | sed 's/^/range: /' &
waiters+=($!)

{ timeout 120s curl -v \
    --retry 5 \
    --ignore-content-length \
    -s http://localhost:8001/logs/kube-apiserver.log \
    -o "${log_dir}"/infrastructure/kube-apiserver-range-ignorelength.log 2>&1; echo $?; } | sed 's/^/ignore-len: /' &
waiters+=($!)

{ retry 5 true curl -v \
    --retry 2 \
    --retry-all-errors \
    --continue-at - \
    -s http://localhost:8001/logs/kube-apiserver.log \
    -o "${log_dir}"/infrastructure/kube-apiserver-breakfix.log 2>&1; echo $?; } | sed 's/^/breakfix: /' &
waiters+=($!)

{ retry 5 true curl -v \
    --continue-at - \
    -s http://localhost:8001/logs/kube-apiserver.log \
    -o "${log_dir}"/infrastructure/kube-apiserver-continue.log 2>&1; echo $?; } | sed 's/^/continue: /' &
waiters+=($!)

echo "waiting for bg procs: ${waiters[*]}"
wait ${waiters[*]}
for testlogfile in "${log_dir}"/infrastructure/kube-apiserver-*log; do
  echo ">>> ${testlogfile}"
  tail -3 "$testlogfile"
done

kill $proxy_pid

echo "$(date) Attempting to collect kube API server metrics"
# Ref https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/
kubectl get --raw /metrics > "${log_dir}"/infrastructure/kube-apiserver-metrics.txt
