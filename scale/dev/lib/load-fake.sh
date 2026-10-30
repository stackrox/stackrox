#!/usr/bin/env bash
# Fake-workload load driver. Source after lib/common.sh.
#
# Implements the load::* interface used by perf-test.sh. It wraps the existing
# run-many-jv.sh (central + sensor with the built-in fake workload generator) rather
# than reimplementing the deploy. The rate is the fileActivityWorkload batchSize.

FAKE_WORKLOAD_DIR="${PT_STACKROX_DIR}/scale/workloads"

# Render the fake-workload yaml template for a given rate into a generated workload
# file and echo the workload name (basename without extension). launch_sensor-jv.sh
# (via run-many-jv.sh) resolves it from scale/workloads/<name>.yaml.
#   $1 template (abs)  $2 rate value  $3 rate field  $4 config name  $5 rate
load::render_workload() {
  local template=$1 rate_value=$2 rate_field=$3 config_name=$4 rate=$5
  local gen_name="gen-${config_name}-${rate}"
  local out_file="${FAKE_WORKLOAD_DIR}/${gen_name}.yaml"
  sed "s|${rate_field}|${rate_value}|g" "$template" > "$out_file"
  echo "$gen_name"
}

# The fake workload has no per-instance k8s objects to name, so there is no instance
# base. Returned empty for interface symmetry with the berserker driver.
load::instance_base() {
  echo ""
}

# Deploy central + sensor with the fake workload baked in.
#   $1 gen workload name  $2 num_sensors  $3 num_deployments (unused)
#   $4 instance base (unused)  $5 results_dir (unused)
load::deploy() {
  local gen_workload_name=$1 num_sensors=$2
  local start_time end_time duration error_code

  # A previous fake run may have left a stray namespace behind.
  kubectl delete ns stackrox1 || true

  echo "Deploying ACS for fake-workload testing..."
  start_time=$(date +%s)
  error_code=1
  while [[ "$error_code" != 0 ]]; do
    "${PT_DEV_DIR}/run-many-jv.sh" "${gen_workload_name}" "$num_sensors"
    error_code=$?
    echo "error_code= $error_code"
  done
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Deploying ACS completed in ${duration} seconds."

  pt::delete_scanner
}

# The fake workload rides inside the sensor deployment; there is nothing to clean up
# per cell (the orchestrator tears the whole install down between cells).
#   $1 num_deployments (unused)  $2 instance base (unused)
load::cleanup() {
  :
}
