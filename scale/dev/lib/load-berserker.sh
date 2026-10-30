#!/usr/bin/env bash
# Berserker load driver. Source after lib/common.sh.
#
# Implements the load::* interface used by perf-test.sh. It wraps the existing
# deploy-berserker.sh / cleanup-berserker.sh scripts rather than reimplementing the
# DaemonSet plumbing; this driver only renders the workload from a template and
# handles the central+sensor deploy specific to berserker (Helm path + fact agent).

BERSERKER_WORKLOAD_DIR="${PT_STACKROX_DIR}/scale/berserker/workloads"

# Render the .ber template for a given rate into a generated workload file and echo
# the workload name (basename without extension) for deploy-berserker.sh to resolve.
#   $1 template (abs)  $2 rate value  $3 rate field  $4 config name  $5 rate
load::render_workload() {
  local template=$1 rate_value=$2 rate_field=$3 config_name=$4 rate=$5
  local gen_name="gen-${config_name}-${rate}"
  local out_file="${BERSERKER_WORKLOAD_DIR}/${gen_name}.ber"
  sed "s|${rate_field}|${rate_value}|g" "$template" > "$out_file"
  echo "$gen_name"
}

# Derive a DNS-1123-safe k8s instance base from the rate (dots -> dashes). The
# instance name becomes part of DaemonSet/label names, which cannot contain dots.
#   $1 rate
load::instance_base() {
  local rate=$1
  echo "r${rate//./-}"
}

# Deploy central + one sensor (Helm path, with the fact agent) and the berserker
# workload(s). Writes berserker_ready_timestamp.txt into results_dir.
#   $1 gen workload name  $2 num_sensors (unused)  $3 num_deployments
#   $4 instance base  $5 results_dir
load::deploy() {
  local gen_workload_name=$1 num_deployments=$3 instance_base=$4 results_dir=$5
  local start_time end_time duration i instance error_code image_registry sensor_helm_values

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
  # chart's baked-in default registry and ignores DEFAULT_IMAGE_REGISTRY. image.registry
  # cascades to every secured-cluster image (sensor, collector, fact). Injected via
  # the deploy script's supported extra-values hook.
  image_registry="${DEFAULT_IMAGE_REGISTRY:-$(make --quiet --no-print-directory -C "$PT_STACKROX_DIR" default-image-registry)}"
  sensor_helm_values="${PT_DEV_DIR}/sensor-image-values.yaml"
  cat > "$sensor_helm_values" <<EOF
image:
  registry: ${image_registry}
EOF
  export ROX_SENSOR_EXTRA_HELM_VALUES_FILE="$sensor_helm_values"

  # NOTE: FACT_PATHS is intentionally NOT set. The only supported mechanism now is
  # the runtime policy's "File Path" field: Sensor derives the monitored paths from
  # enabled file-access policies and pushes them to fact via the fact-config ConfigMap
  # (sensor/common/filesystem/settings_manager.go). So the enabled policy (or absence
  # of one for baseline) controls what fact monitors.

  echo "Deploying ACS for berserker testing..."
  start_time=$(date +%s)
  error_code=1
  while [[ "$error_code" != 0 ]]; do
    if ! kubectl -n stackrox get deploy/central; then
      "${PT_DEV_DIR}/launch_central-jv.sh"
      kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
    else
      kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
      killpf 8000 || true
      "${PT_DEV_DIR}/port-forward-jv.sh" 8000
      sleep 5
    fi

    "${PT_STACKROX_DIR}/deploy/k8s/sensor.sh"

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

  pt::delete_scanner

  echo "Deploying ${num_deployments} berserker deployment(s) for workload: ${gen_workload_name}"
  start_time=$(date +%s)
  for i in $(seq 1 "$num_deployments"); do
    instance="$(load::_instance_name "$instance_base" "$i" "$num_deployments")"
    echo "Deploying berserker instance ${i}/${num_deployments}: ${instance}"
    "${PT_DEV_DIR}/deploy-berserker.sh" "${gen_workload_name}" "stackrox" "${instance}"
  done
  # Record when berserker is up (epoch ms). deploy-berserker.sh blocks on each
  # DaemonSet rollout, so control returns here only once every instance is ready.
  # Berserker is deployed last, so this single timestamp is t=0 for every component.
  # Consumed by determine_baseline_timestamp in plot_utils.py.
  date +%s%3N > "${results_dir}/berserker_ready_timestamp.txt"
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  echo "Berserker deployment completed in ${duration} seconds."
}

# Instance (k8s object) name for the i-th deployment. A single deployment keeps the
# plain instance base; multiple deployments get a -dN suffix so each is a distinct
# StackRox deployment (alerts hash by deployment ID, sharding onto separate workers).
#   $1 instance base  $2 index  $3 num_deployments
load::_instance_name() {
  local instance_base=$1 idx=$2 num_deployments=$3
  if [[ "$num_deployments" -eq 1 ]]; then
    echo "${instance_base}"
  else
    echo "${instance_base}-d${idx}"
  fi
}

# Tear down the berserker workload(s) - one call per deployed instance.
#   $1 num_deployments  $2 instance base
load::cleanup() {
  local num_deployments=$1 instance_base=$2 i
  echo "Cleaning up ${num_deployments} berserker deployment(s)..."
  for i in $(seq 1 "$num_deployments"); do
    "${PT_DEV_DIR}/cleanup-berserker.sh" "$(load::_instance_name "$instance_base" "$i" "$num_deployments")" "stackrox"
  done
}
