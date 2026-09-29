#!/usr/bin/env bash
set -eoux pipefail

# This script runs a series of file activity performance tests
# with varying batch sizes and with/without policy enforcement

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

num_sensors=${1:-1}
run_time=${2:-10m}
results_base_dir=${3:-perf}  # base directory for results; defaults to perf
workload_type=${4:-fake}     # fake or berserker

# Select the per-test script based on workload type. The two workloads also use
# different workload-name conventions (see workload_name below).
if [[ "$workload_type" == "berserker" ]]; then
  perf_test_script="${DIR}/perf-test-file-activity-berserker.sh"
  results_dir_prefix="berserker_file_activity_results"
else
  perf_test_script="${DIR}/perf-test-file-activity.sh"
  results_dir_prefix="file_activity_results"
fi

# Array of batch sizes to test
batch_sizes=(10 50 100 250 500)

# The set of scenarios to run per batch size, and the results-dir suffix each maps to.
# berserker uses three modes (see perf-test-file-activity-berserker.sh); the fake
# workload keeps the original with/without-policy pair.
#   berserker: mode name IS the suffix        -> ..._<mode>
#   fake:      legacy true/false with a prefix -> ..._policy_<mode>
if [[ "$workload_type" == "berserker" ]]; then
  test_modes=(baseline alerting idlepolicy)
else
  test_modes=(false true)
fi

suffix_for_mode() {
  local mode=$1
  if [[ "$workload_type" == "berserker" ]]; then
    echo "${mode}"
  else
    echo "policy_${mode}"
  fi
}

# Run a single test for the given batch size and test mode.
run_batch_test() {
  local batch=$1
  local mode=$2

  # Fake workload names by batch size (file-activity-batch-10); berserker names
  # by event rate, which is batch size * 10 (file-activity-100).
  local workload_name
  if [[ "$workload_type" == "berserker" ]]; then
    workload_name="file-activity-$((batch * 10))"
  else
    workload_name="file-activity-batch-${batch}"
  fi

  # Resume support: skip a case whose results dir already exists, so an
  # interrupted run can be restarted by re-invoking with the same arguments
  # (each test takes many minutes). This must mirror the results_dir name the
  # per-test scripts build (results_dir_prefix differs by workload). Delete a
  # dir to force its test to re-run; delete a partial dir before resuming.
  local mode_suffix
  mode_suffix="$(suffix_for_mode "$mode")"
  local results_dir="${results_base_dir}/${results_dir_prefix}_${num_sensors}_${run_time}_${workload_name}_${mode_suffix}"
  if [[ -d "$results_dir" ]]; then
    echo "Results dir already exists, skipping: ${results_dir}"
    return 0
  fi

  "$perf_test_script" \
    "$num_sensors" \
    "$run_time" \
    "$workload_name" \
    "$mode" \
    "$results_base_dir"

  # Clean up between tests
  echo "Cleaning up before next test..."
  "${DIR}/TeardownTest.sh" || true
  sleep 30
}

# Run every batch size for each mode. Iterate mode-outer so all rates for one
# scenario complete before moving to the next.
for mode in "${test_modes[@]}"; do
  echo "=== Running ${workload_type} tests: mode=${mode} ==="
  for batch in "${batch_sizes[@]}"; do
    echo "Testing batch ${batch} (mode=${mode})..."
    run_batch_test "$batch" "$mode"
  done
done

echo "All file activity tests completed!"
if [[ "$workload_type" == "berserker" ]]; then
  echo "Results are in ${results_base_dir}/berserker_file_activity_results_* directories"
else
  echo "Results are in ${results_base_dir}/file_activity_results_* directories"
fi
