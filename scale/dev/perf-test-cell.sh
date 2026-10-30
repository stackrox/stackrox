#!/usr/bin/env bash
set -eou pipefail

# Unified single-cell file-activity perf test. Replaces perf-test-file-activity.sh
# and perf-test-file-activity-berserker.sh: the load type, workload template, rate
# and policy are all described by a JSON run config, so one script drives both the
# fake and the berserker workloads.
#
# Usage: perf-test-cell.sh <config.json> <rate> <policy_nickname>
#   config.json:     a run config (see configs/*.json)
#   rate:            the actual event rate of one of the config's "rates" entries
#                    (the "rate" field, e.g. 1000). The matching "value" is what gets
#                    substituted into the load template's rate field.
#   policy_nickname: one of the config's "policies[].nickname" entries
#
# Example:
#   ./perf-test-cell.sh configs/berserker-file-activity.json 1000 alerting

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${DIR}/lib/common.sh"

config="${1:?usage: $0 <config.json> <rate> <policy_nickname>}"
rate="${2:?usage: $0 <config.json> <rate> <policy_nickname>}"
policy_nickname="${3:?usage: $0 <config.json> <rate> <policy_nickname>}"

if [[ ! -f "$config" ]]; then
  echo "Error: config not found: ${config}"; exit 1
fi

# Parse the run config.
name="$(jq -r '.name' "$config")"
load_type="$(jq -r '.loadType' "$config")"
load_template="$(jq -r '.loadTemplate' "$config")"
rate_field="$(jq -r '.rateField' "$config")"
num_sensors="$(jq -r '.numSensors' "$config")"
run_time="$(jq -r '.runTime' "$config")"
num_deployments="$(jq -r '.numDeployments // 1' "$config")"
results_base_dir="$(jq -r '.resultsBaseDir' "$config")"

# Validate the rate is declared in the config and resolve the template substitution
# value for it. The cell is identified by the actual event rate (.rates[].rate); the
# value substituted into the template is the paired .rates[].value.
rate_value="$(jq -r --arg r "$rate" \
  '.rates[] | select((.rate|tostring)==$r) | .value' "$config")"
if [[ -z "$rate_value" ]]; then
  echo "Error: rate '${rate}' is not in the config's rates: $(jq -c '[.rates[].rate]' "$config")"
  exit 1
fi

# Resolve the policy file for the nickname ("null"/absent => baseline, no policy).
policy_file="$(jq -r --arg n "$policy_nickname" \
  '.policies[] | select(.nickname==$n) | .file // "null"' "$config")"
if [[ -z "$policy_file" ]]; then
  echo "Error: policy nickname '${policy_nickname}' not found in config"; exit 1
fi

# Load driver for this load type (defines load::render_workload/deploy/cleanup).
# shellcheck source=/dev/null
source "${DIR}/lib/load-${load_type}.sh"

pt::setup_env

# Results dir: <resultsBaseDir>/<name>/rate-<rate>__<policy_nickname>. The base dir
# is anchored under scale/dev when relative so the layout is independent of cwd.
if [[ "$results_base_dir" = /* ]]; then
  run_dir="${results_base_dir}/${name}"
else
  run_dir="${DIR}/${results_base_dir}/${name}"
fi
results_dir="${run_dir}/rate-${rate}__${policy_nickname}"

script_start_time=$(date +%s)

# Tear down any previous install with the workflow `teardown` script (not just a
# namespace delete) so cluster-scoped objects do not linger and conflict with the
# next Helm apply.
teardown || true

rm -rf "$results_dir"
mkdir -p "$results_dir"

# Render the workload from the template for this rate, then deploy the load. The
# template gets the substitution value; the generated filename uses the actual rate.
template_abs="${DIR}/${load_template}"
gen_workload_name="$(load::render_workload "$template_abs" "$rate_value" "$rate_field" "$name" "$rate")"
instance_base="$(load::instance_base "$rate")"

start_time=$(date +%s)
load::deploy "$gen_workload_name" "$num_sensors" "$num_deployments" "$instance_base" "$results_dir"
end_time=$(date +%s)
echo "Load deploy completed in $((end_time - start_time)) seconds."

pt::port_forward_api
pt::acquire_token
pt::disable_all_policies

# Enable the selected policy (if any). The policy's File Path drives what the fact
# agent monitors, so the choice of policy (or baseline/none) defines the scenario.
active_policy_name=""
if [[ "$policy_file" != "null" ]]; then
  active_policy_name="$(jq -r '.name' "${DIR}/${policy_file}")"
  start_time=$(date +%s)
  echo "Enabling policy '${policy_nickname}' (${active_policy_name})..."
  "${DIR}/enable-policy.sh" "${DIR}/${policy_file}"
  end_time=$(date +%s)
  echo "Enabling policy completed in $((end_time - start_time)) seconds."
else
  echo "Running baseline cell with NO policy (fact idle)"
fi

# Wait for workload to stabilize, then generate for the configured run time.
sleep 30

start_time=$(date +%s)
echo "Generating file activity for ${run_time}..."
sleep "$run_time"
end_time=$(date +%s)
echo "File activity generation period completed in $((end_time - start_time)) seconds."

pt::get_diagnostic_bundle "${results_dir}/diagnostic_bundle_1"
[[ -n "$active_policy_name" ]] && pt::checkdb "$active_policy_name" "${results_dir}/file_activity_alerts_1.txt"

# Additional monitoring period.
sleep 5m

pt::get_diagnostic_bundle "${results_dir}/diagnostic_bundle_2"
[[ -n "$active_policy_name" ]] && pt::checkdb "$active_policy_name" "${results_dir}/file_activity_alerts_2.txt"

# K6 load test.
start_time=$(date +%s)
pt::run_k6 "$results_dir" "$num_sensors" "$run_time"
end_time=$(date +%s)
echo "K6 load testing completed in $((end_time - start_time)) seconds."

# Final monitoring period, then scrape Prometheus.
sleep 10m

start_time=$(date +%s)
pt::query_prometheus "${results_dir}/metrics"
end_time=$(date +%s)
echo "Prometheus query completed in $((end_time - start_time)) seconds."

# Tear down the load.
start_time=$(date +%s)
load::cleanup "$num_deployments" "$instance_base"
end_time=$(date +%s)
echo "Load cleanup completed in $((end_time - start_time)) seconds."

script_end_time=$(date +%s)
echo "Cell completed in $((script_end_time - script_start_time)) seconds: ${results_dir}"
