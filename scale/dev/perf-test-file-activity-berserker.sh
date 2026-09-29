#!/usr/bin/env bash
set -eoux pipefail

get_diagnostic_bundle() {
  diagnostic_bundle_dir=$1

  nc -z 127.0.0.1 8000 || "${DIR}/port-forward-jv.sh" 8000
  until nc -z 127.0.0.1 8000; do
          sleep 1
          echo "Waiting for port forward 8000"
  done
  # download-diagnostics (unlike debug dump, which is Central-only) collects a
  # platform-wide bundle: pod info and logs from Central and the Secured
  # Clusters (Sensor, Collector, Admission Controller).
  roxctl central debug download-diagnostics -e localhost:8000 -p "${ROX_ADMIN_PASSWORD}" --insecure-skip-tls-verify
  ls "$diagnostic_bundle_dir" || mkdir -p "$diagnostic_bundle_dir"
  mv *.zip "$diagnostic_bundle_dir"
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

STACKROX_DIR="$DIR/../.."

num_sensors=$1
run_time=$2
workload_name=$3  # e.g., file-activity-100, file-activity-500
# test_mode selects one of three scenarios (replaces the old with/without-policy pair):
#   alerting   - runtime policy on /tmp/data/** (where berserker writes). fact monitors
#                that path, so it is busy and alerts fire on berserker's activity.
#   baseline   - no file-activity policy at all. With FACT_PATHS no longer set, fact has
#                nothing to monitor and is almost completely inactive (control case).
#   idlepolicy - runtime policy on /tmp/quiet/** (a directory with NO berserker activity).
#                fact is configured to monitor a path but sees no events, isolating the
#                cost of an active monitoring policy from the cost of processing events.
test_mode=${4:-baseline}
results_base_dir=${5:-perf}  # base directory for results; defaults to perf

case "$test_mode" in
  alerting|baseline|idlepolicy) ;;
  *) echo "Error: test_mode must be one of: alerting, baseline, idlepolicy (got '$test_mode')"; exit 1 ;;
esac

# Directory the idlepolicy mode points its policy at. berserker only ever touches
# /tmp/data, so /tmp/quiet stays empty -> the policy is active but no events flow.
IDLE_POLICY_PATH="/tmp/quiet/**"
ALERTING_POLICY_NAME="Berserker File Activity Test Policy"
IDLE_POLICY_NAME="Berserker Idle File Activity Policy"

# Export default value for PAGERDUTY_INTEGRATION_KEY to fix Helm validation error
export PAGERDUTY_INTEGRATION_KEY="${PAGERDUTY_INTEGRATION_KEY:-dummy-key-for-dev-testing}"
export STORAGE="pvc"
export MONITORING_LOAD_BALANCER="none"
export MONITORING_SUPPORT=true
export ROX_SCANNER_V4=false
# Force the Helm-based sensor deploy. The fact container (file-activity monitor)
# is only rendered on the Helm path, where SFA_AGENT=true sets
# collector.famEnabled=true; the roxctl bundle path has no equivalent flag.
# deploy/k8s/sensor.sh only auto-enables Helm for helm v3, so newer helm (v4)
# silently falls back to the bundle path and drops fact. Set this explicitly.
export SENSOR_HELM_DEPLOY=true
# Enable File Activity Monitoring (adds the fact container via the Helm chart).
export SFA_AGENT=true

# Point the secured-cluster Helm deploy at the same image registry as the rest of
# the deployment. The Helm sensor path passes no image overrides, so it uses the
# chart's baked-in default registry (quay.io/rhacs-eng, from Central's image
# flavor) and ignores DEFAULT_IMAGE_REGISTRY/MAIN_IMAGE_REPO. That makes sensor,
# its init containers, collector and fact pull from a repo that may not host them
# (ImagePullBackOff). image.registry cascades to every secured-cluster image in
# the chart. Injected via the deploy script's supported extra-values hook.
image_registry="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory -C "$STACKROX_DIR" default-image-registry)}"
sensor_helm_values="${DIR}/sensor-image-values.yaml"
# NOTE: FACT_PATHS is intentionally NOT set. It is the deprecated, static way of
# telling the fact container what to monitor. The only supported mechanism now is
# the runtime policy's "File Path" field: Sensor derives the monitored paths from
# enabled file-access policies and pushes them to fact via the fact-config ConfigMap
# (sensor/common/filesystem/settings_manager.go), gated by the ROX_SENSITIVE_FILE_ACTIVITY
# feature flag (default enabled). Consequences, per test_mode:
#   baseline   -> no policy -> fact monitors nothing (nearly idle).
#   alerting   -> policy on /tmp/data/** -> fact monitors berserker's directory.
#   idlepolicy -> policy on /tmp/quiet/** -> fact monitors an empty directory.
cat > "$sensor_helm_values" <<EOF
image:
  registry: ${image_registry}
EOF
export ROX_SENSOR_EXTRA_HELM_VALUES_FILE="$sensor_helm_values"

# Tear down any previous install with the workflow `teardown` script rather than
# just deleting the namespace. `kubectl delete ns stackrox` only removes
# namespaced objects; teardown also removes the cluster-scoped ClusterRoles,
# ClusterRoleBindings and ValidatingWebhookConfiguration, which otherwise linger
# and conflict with Helm's server-side apply on the next deploy.
teardown || true

results_dir="${results_base_dir}/berserker_file_activity_results_${num_sensors}_${run_time}_${workload_name}_${test_mode}"
# Start from a clean results dir. rm -rf succeeds even when the dir is absent,
# so mkdir must run unconditionally (not as an || fallback) to ensure it exists.
rm -rf "$results_dir"
mkdir -p "$results_dir"

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
# Record when berserker is up (epoch ms, matching the Prometheus metric
# timestamps). deploy-berserker.sh blocks on the DaemonSet rollout, so control
# returns here only once berserker is ready. Berserker is deployed last (central,
# sensor and collector are already up), so this single timestamp is t=0 for every
# component in the pipeline. Consumed by determine_baseline_timestamp in plot_utils.py.
date +%s%3N > "${results_dir}/berserker_ready_timestamp.txt"
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

# Enable the file-activity policy appropriate to the mode. The policy's File Path is
# what drives fact's monitoring (see the FACT_PATHS note above), so:
#   alerting   -> watch /tmp/data/** (berserker's dir): fact busy, alerts fire.
#   idlepolicy -> watch /tmp/quiet/** (empty): fact configured, no events, ~no alerts.
#   baseline   -> no policy: fact idle.
# active_policy_name is the policy CheckDB later queries for alerts; empty for baseline.
active_policy_name=""
case "$test_mode" in
  alerting)
    active_policy_name="${ALERTING_POLICY_NAME}"
    start_time=$(date +%s)
    echo "Enabling alerting file activity policy (watching /tmp/data/**)..."
    "${DIR}/enable-file-activity-policy-berserker.sh" "/tmp/data/**" "${ALERTING_POLICY_NAME}"
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    echo "Enabling file activity policy completed in ${duration} seconds."
    ;;
  idlepolicy)
    active_policy_name="${IDLE_POLICY_NAME}"
    start_time=$(date +%s)
    echo "Enabling idle file activity policy (watching ${IDLE_POLICY_PATH}, no berserker activity)..."
    "${DIR}/enable-file-activity-policy-berserker.sh" "${IDLE_POLICY_PATH}" "${IDLE_POLICY_NAME}"
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    echo "Enabling idle file activity policy completed in ${duration} seconds."
    ;;
  baseline)
    echo "Running baseline test with NO file activity policy (fact idle)"
    ;;
esac

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

if [[ -n "$active_policy_name" ]]; then
  start_time=$(date +%s)
  "${DIR}/CheckDB-file-activity.sh" "${active_policy_name}" > "${results_dir}/file_activity_alerts_1.txt"
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

if [[ -n "$active_policy_name" ]]; then
  start_time=$(date +%s)
  "${DIR}/CheckDB-file-activity.sh" "${active_policy_name}" > "${results_dir}/file_activity_alerts_2.txt"
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
