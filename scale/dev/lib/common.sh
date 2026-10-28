#!/usr/bin/env bash
# Shared helpers for the file-activity perf-test harness.
#
# Source this file; it does not run anything on its own. Functions are prefixed
# `pt::`. The two load drivers (lib/load-berserker.sh, lib/load-fake.sh) and the
# unified driver (perf-test.sh) all build on these.
#
# PT_DEV_DIR      - scale/dev (where the harness scripts live)
# PT_STACKROX_DIR - repo root
PT_DEV_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PT_STACKROX_DIR="$(cd "${PT_DEV_DIR}/../.." && pwd)"
export PT_DEV_DIR PT_STACKROX_DIR

# Environment common to every run. Previously duplicated verbatim at the top of
# both perf-test-file-activity*.sh scripts.
pt::setup_env() {
  export PAGERDUTY_INTEGRATION_KEY="${PAGERDUTY_INTEGRATION_KEY:-dummy-key-for-dev-testing}"
  export STORAGE="pvc"
  export MONITORING_LOAD_BALANCER="none"
  export MONITORING_SUPPORT=true
  export ROX_SCANNER_V4=false
}

# Download a platform-wide diagnostic bundle (Central + secured clusters) into the
# given directory. Needs the Central admin port-forward on 8000.
pt::get_diagnostic_bundle() {
  local diagnostic_bundle_dir=$1

  nc -z 127.0.0.1 8000 || "${PT_DEV_DIR}/port-forward-jv.sh" 8000
  until nc -z 127.0.0.1 8000; do
    sleep 1
    echo "Waiting for port forward 8000"
  done
  # download-diagnostics (unlike debug dump, which is Central-only) collects a
  # platform-wide bundle: pod info and logs from Central and the Secured
  # Clusters (Sensor, Collector, Admission Controller).
  roxctl central debug download-diagnostics -e localhost:8000 -p "${ROX_ADMIN_PASSWORD}" --insecure-skip-tls-verify
  ls "$diagnostic_bundle_dir" || mkdir -p "$diagnostic_bundle_dir"
  mv -- *.zip "$diagnostic_bundle_dir"
}

# Drop scanner to reduce resource usage during the test.
pt::delete_scanner() {
  kubectl -n stackrox delete deployment scanner || true
  kubectl -n stackrox delete deployment scanner-db || true
}

# Wait for the sensor pod to become Ready, failing fast with diagnostics if it does
# not schedule in time. Without this the harness would run a cell's full measurement
# cycle against a Pending/absent sensor (e.g. an unschedulable resource request),
# wasting ~30+ minutes per cell and producing empty results.
#   $1 namespace (default stackrox)  $2 timeout (default 5m)
pt::wait_for_sensor() {
  local ns=${1:-stackrox} timeout=${2:-5m}
  echo "Waiting for sensor to become ready in namespace ${ns} (timeout ${timeout})..."
  if kubectl -n "$ns" wait --for=condition=ready pod -l app=sensor --timeout="$timeout"; then
    echo "Sensor is ready."
    return 0
  fi
  echo "ERROR: sensor pod did not become ready in namespace ${ns} within ${timeout}." >&2
  echo "Sensor pod status:" >&2
  kubectl -n "$ns" get pods -l app=sensor -o wide >&2 || true
  echo "Scheduling events (look for FailedScheduling / Insufficient memory|cpu):" >&2
  kubectl -n "$ns" describe pod -l app=sensor 2>&1 | grep -iA12 'Events:' >&2 || true
  exit 1
}

# Port-forward the Central API on 8001 and export the endpoint vars the policy and
# token helpers use.
pt::port_forward_api() {
  local start_time end_time duration
  start_time=$(date +%s)
  kubectl -n stackrox port-forward deploy/central 8001:8443 > /dev/null 2>&1 &
  until nc -z 127.0.0.1 8001; do sleep 1; done
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Waiting for port forward completed in ${duration} seconds."

  export HOST=https://127.0.0.1:8001
  export ROX_ENDPOINT=https://127.0.0.1:8001
  export API_ENDPOINT=https://127.0.0.1:8001
}

# Load the admin password from the deploy output and export password + API token.
pt::acquire_token() {
  local rox_admin_password rox_api_token
  rox_admin_password="$(cat "${PT_STACKROX_DIR}/deploy/k8s/central-deploy/password")"
  export ROX_ADMIN_PASSWORD="$rox_admin_password"
  export ROX_PASSWORD="$rox_admin_password"

  rox_api_token="$(logmein "$ROX_ENDPOINT")"
  export ROX_API_TOKEN="$rox_api_token"
}

# Disable all default policies so only our test policy can fire.
pt::disable_all_policies() {
  local start_time end_time duration
  start_time=$(date +%s)
  "${PT_STACKROX_DIR}/scratch/policies/disable-policies-db.sh"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Disabled policies completed in ${duration} seconds."
}

# Count alerts for the given policy name into the given output file.
pt::checkdb() {
  local policy_name=$1 outfile=$2
  "${PT_DEV_DIR}/CheckDB-file-activity.sh" "${policy_name}" > "${outfile}"
}

# Run the K6 integration load test, writing results under results_dir.
pt::run_k6() {
  local results_dir=$1
  local num_sensors=$2
  local run_time=$3
  mkdir -p "${PT_DEV_DIR}/performance-results"
  k6 run "${PT_STACKROX_DIR}/tests/performance/tests/testK6Integration.js" \
    --vus 5 --iterations 10 --out csv=file_activity_test.csv \
    &> "${results_dir}/k6_load_test.txt" || true
  mv "${PT_DEV_DIR}/performance-results" \
    "${results_dir}/performance-results-${num_sensors}-${run_time}" || true
}

# Port-forward the monitoring service and scrape Prometheus metrics into metrics_dir.
# sensor_ns is the namespace of the secured-cluster components (sensor/collector/
# fact); defaults to stackrox. Central/central-db/table metrics are always queried in
# the central namespace (stackrox) regardless.
pt::query_prometheus() {
  local metrics_dir=$1
  local sensor_ns=${2:-stackrox}
  kubectl -n stackrox port-forward service/monitoring 48443:8443 > /dev/null 2>&1 &
  until nc -z 127.0.0.1 48443; do
    sleep 1
    echo "Waiting for port forward 48443"
  done
  "${PT_DEV_DIR}/prometheus-query-file-activity.sh" "${metrics_dir}" "${sensor_ns}"
}
