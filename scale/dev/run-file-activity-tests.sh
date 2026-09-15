#!/usr/bin/env bash
set -eoux pipefail

# This script runs a series of file activity performance tests
# with varying batch sizes and with/without policy enforcement

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

num_sensors=${1:-1}
run_time=${2:-10m}
results_base_dir=${3:-perf}  # base directory for results; defaults to perf

# Array of batch sizes to test
batch_sizes=(10 50 100 250 500)

# Test each batch size without policy
echo "=== Running tests WITHOUT file activity policy ==="
for batch in "${batch_sizes[@]}"; do
  echo "Testing file-activity-batch-${batch} without policy..."
  "${DIR}/perf-test-file-activity.sh" \
    "$num_sensors" \
    "$run_time" \
    "file-activity-batch-${batch}" \
    "false" \
    "$results_base_dir"

  # Clean up between tests
  echo "Cleaning up before next test..."
  "${DIR}/TeardownTest.sh" || true
  sleep 30
done

# Test each batch size with policy enabled
echo "=== Running tests WITH file activity policy ==="
for batch in "${batch_sizes[@]}"; do
  echo "Testing file-activity-batch-${batch} with policy..."
  "${DIR}/perf-test-file-activity.sh" \
    "$num_sensors" \
    "$run_time" \
    "file-activity-batch-${batch}" \
    "true" \
    "$results_base_dir"

  # Clean up between tests
  echo "Cleaning up before next test..."
  "${DIR}/TeardownTest.sh" || true
  sleep 30
done

echo "All file activity tests completed!"
echo "Results are in ${results_base_dir}/file_activity_results_* directories"
