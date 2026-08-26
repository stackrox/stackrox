#!/usr/bin/env bash
set -eoux pipefail

get_diagnostic_bundle() {
  diagnostic_bundle_dir=$1

  nc -z 127.0.0.1 8000 || "${DIR}/port-forward-jv.sh" 8000
  until nc -z 127.0.0.1 8000; do
          sleep 1
          echo "Waiting for port forward 8000"
  done
  roxctl central debug dump -e localhost:8000 -p "${ROX_ADMIN_PASSWORD}" --insecure-skip-tls-verify
  ls "$diagnostic_bundle_dir" || mkdir -p "$diagnostic_bundle_dir"
  mv *.zip "$diagnostic_bundle_dir"
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

STACKROX_DIR="$DIR/../.."

num_sensors=$1
run_time=$2
workload_name=$3  # e.g., file-activity-100, file-activity-500
test_with_policy=${4:-false}  # true or false - whether to enable file activity policy

logmein_script="${HOME}/go/src/github.com/stackrox/workflow/bin/logmein"

# Export default value for PAGERDUTY_INTEGRATION_KEY to fix Helm validation error
export PAGERDUTY_INTEGRATION_KEY="${PAGERDUTY_INTEGRATION_KEY:-dummy-key-for-dev-testing}"
export STORAGE="pvc"
export MONITORING_LOAD_BALANCER="none"
export MONITORING_SUPPORT=true
export ROX_SCANNER_V4=false

kubectl delete ns stackrox || true

results_dir="perf/berserker_file_activity_results_${num_sensors}_${run_time}_${workload_name}_policy_${test_with_policy}"
rm -rf "$results_dir" || mkdir -p "$results_dir"

script_start_time=$(date +%s)

error_code=1

# Deploy ACS (central and sensor with collector, but no fake workload)
echo "Deploying ACS for berserker testing..."
start_time=$(date +%s)
while [[ "$error_code" != 0 ]]; do
  # Deploy central
  if ! kubectl -n stackrox get deploy/central; then
    "${DIR}"/launch_central-jv.sh
    kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
  else
    kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
    killpf 8000 || true
    "${DIR}"/port-forward-jv.sh 8000
    sleep 5
  fi

  # Deploy sensor without fake workload
  "${STACKROX_DIR}/deploy/k8s/sensor.sh"

  # Verify central-db PVC exists
  if ! kubectl -n stackrox get pvc/central-db > /dev/null; then
    echo "Error: Running the scale workload requires a PVC"
    exit 1
  fi

  # Apply resource patches for multi-node clusters if needed
  if [[ $(kubectl get nodes -o json | jq '.items | length') != 1 ]]; then
    kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"40Gi","cpu":"4"},"limits":{"memory":"40Gi","cpu":"8"}}}]}}}}'
  fi

  error_code=$?
  echo "error_code= $error_code"
done
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Deploying ACS completed in ${duration} seconds."

# Delete scanner to reduce resource usage
kubectl -n stackrox delete deployment scanner || true
kubectl -n stackrox delete deployment scanner-db || true

# Deploy berserker workload
echo "Deploying berserker workload: ${workload_name}"
start_time=$(date +%s)
"${DIR}/deploy-berserker.sh" "${workload_name}" "stackrox"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Berserker deployment completed in ${duration} seconds."

rox_admin_password="$(cat "$STACKROX_DIR/deploy/k8s/central-deploy/password")"
export ROX_ADMIN_PASSWORD="$rox_admin_password"
export ROX_PASSWORD="$rox_admin_password"

start_time=$(date +%s)
kubectl -n stackrox port-forward deploy/central 8001:8443 > /dev/null 2>&1 &
until nc -z 127.0.0.1 8001; do sleep 1; done
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Waiting for port forward completed in ${duration} seconds."

export HOST=https://127.0.0.1:8001
export ROX_ENDPOINT=https://127.0.0.1:8001
export API_ENDPOINT=https://127.0.0.1:8001

# Get API token for policy management
rox_api_token="$(logmein "$ROX_ENDPOINT")"
export ROX_API_TOKEN="$rox_api_token"

# Disable all policies first
start_time=$(date +%s)
"${STACKROX_DIR}/scratch/policies/disable-policies-db.sh"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Disabled policies completed in ${duration} seconds."

# Enable or disable file activity policy based on parameter
if [[ "$test_with_policy" == "true" ]]; then
  start_time=$(date +%s)
  echo "Enabling file activity test policy..."
  "${DIR}/enable-file-activity-policy-berserker.sh"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Enabling file activity policy completed in ${duration} seconds."
else
  echo "Running test WITHOUT file activity policy enabled"
fi

# Wait for workload to stabilize
sleep 30

start_time=$(date +%s)
echo "Waiting for workload to generate file activity events for ${run_time}..."
sleep "$run_time"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "File activity generation period completed in ${duration} seconds."

start_time=$(date +%s)
get_diagnostic_bundle "${results_dir}/diagnostic_bundle_1"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Get the diagnostic bundle completed in ${duration} seconds."

if [[ "$test_with_policy" == "true" ]]; then
  start_time=$(date +%s)
  "${DIR}/CheckDB-file-activity.sh" > "${results_dir}/file_activity_alerts_1.txt"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Checking the DB for file activity alerts completed in ${duration} seconds."
fi

# Additional monitoring period
sleep 5m

start_time=$(date +%s)
get_diagnostic_bundle "${results_dir}/diagnostic_bundle_2"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Getting the diagnostic bundle completed in ${duration} seconds."

if [[ "$test_with_policy" == "true" ]]; then
  start_time=$(date +%s)
  "${DIR}/CheckDB-file-activity.sh" > "${results_dir}/file_activity_alerts_2.txt"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Checking the DB for file activity alerts completed in ${duration} seconds."
fi

# Run K6 load test
start_time=$(date +%s)
mkdir -p "${DIR}"/performance-results
k6 run "${STACKROX_DIR}"/tests/performance/tests/testK6Integration.js --vus 5 --iterations 10 --out csv=file_activity_test.csv &> "$results_dir/k6_load_test.txt" || true
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "K6 load testing completed in ${duration} seconds."

mv "${DIR}"/performance-results "${results_dir}/performance-results-${num_sensors}-${run_time}" || true

# Final monitoring period
sleep 10m

start_time=$(date +%s)
kubectl -n stackrox port-forward service/monitoring 48443:8443 > /dev/null 2>&1 &
until nc -z 127.0.0.1 48443; do
        sleep 1
        echo "Waiting for port forward 48443"
done
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Port forward completed in ${duration} seconds."

start_time=$(date +%s)
"${DIR}"/prometheus-query-file-activity.sh "$results_dir/metrics"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Prometheus query script completed in ${duration} seconds."

# Cleanup berserker workload
echo "Cleaning up berserker workload..."
start_time=$(date +%s)
"${DIR}/cleanup-berserker.sh" "${workload_name}" "stackrox"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Berserker cleanup completed in ${duration} seconds."

script_end_time=$(date +%s)
duration=$((script_end_time - script_start_time))
echo "Script completed in ${duration} seconds."
