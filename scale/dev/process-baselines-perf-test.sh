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
PROCESS_BASELINE_SCRIPT="${STACKROX_DIR}/scratch/process-baselines/lock-all-process-baselines.sh"

num_sensors=$1
run_time=$2
lock_baselines=$3
num_deployments=$4

logmein_script="${HOME}/go/src/github.com/stackrox/workflow/bin/logmein"

export MONITORING_SUPPORT=true
export ROX_BASELINE_GENERATION_DURATION=5m
export ROX_SCANNER_V4=false

kubectl delete ns stackrox1 || true

results_dir="process_baseline_results_${num_sensors}_${run_time}_${lock_baselines}_${num_deployments}"
rm -r "$results_dir" || mkdir "$results_dir"

script_start_time=$(date +%s)

error_code=1

start_time=$(date +%s)
while [[ "$error_code" != 0 ]]; do
  #"${DIR}"/run-many-jv.sh default "$num_sensors"
  #"${DIR}"/run-many-jv.sh process-baselines-"${num_deployments}"-deployments "$num_sensors" || true
  "${DIR}"/run-many-jv.sh process-baselines-"${num_deployments}"-deployments "$num_sensors"
  error_code=$?
  echo "error_code= $error_code"
done
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Deploying ACS completed in ${duration} seconds."


kubectl -n stackrox delete deployment scanner
kubectl -n stackrox delete deployment scanner-db

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

start_time=$(date +%s)
"${DIR}/wait-for-deployments-jv.sh" "$num_deployments"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Waiting for deployments completed in ${duration} seconds."

sleep "$run_time"

start_time=$(date +%s)
get_diagnostic_bundle "${results_dir}/diagnostic_bundle_1"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Get the diagnostic bundle completed in ${duration} seconds."

start_time=$(date +%s)
"${DIR}/CheckDB.sh" > "${results_dir}/process_baseline_violations_1.txt"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Checking the DB completed in ${duration} seconds."

rox_api_token="$(logmein "$ROX_ENDPOINT")"
export ROX_API_TOKEN="$rox_api_token"

baseline_time=$(($(date +%s%3N)))
echo "$baseline_time" > "${results_dir}/baseline_time.txt"

if [[ "$lock_baselines" == "true" ]]; then
  start_time=$(date +%s)
  "${PROCESS_BASELINE_SCRIPT}" "$ROX_ENDPOINT"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Baseline script completed in ${duration} seconds."
else
  sleep 8 # Approximately how long it takes to run the script so that the timings are about the same when the baselines are locked and not locked
fi

baseline_end_time=$(($(date +%s%3N)))
echo "$baseline_end_time" > "${results_dir}/baseline_end_time.txt"

sleep 5m

start_time=$(date +%s)
get_diagnostic_bundle "${results_dir}/diagnostic_bundle_2"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Getting the diagnostic bundle completed in ${duration} seconds."

start_time=$(date +%s)
"${DIR}/CheckDB.sh" > "${results_dir}/process_baseline_violations_2.txt"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Checking the DB completed in ${duration} seconds."


start_time=$(date +%s)
mkdir "${DIR}"/performance-results
k6 run "${STACKROX_DIR}"/tests/performance/tests/testK6Integration.js --vus 5 --iterations 10 --out csv=process_baselines_pre_lock.csv &> "$results_dir/k6_load_test.txt"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "K6 load testing completed in ${duration} seconds."

mv "${DIR}"/performance-results "${results_dir}/performance-results-${num_sensors}-${run_time}"

sleep 15m

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
"${DIR}"/prometheus-query.sh "$results_dir/metrics"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Prometheus query script completed in ${duration} seconds."

script_end_time=$(date +%s)
duration=$((script_end_time - script_start_time))
echo "Sript completed in ${duration} seconds."
