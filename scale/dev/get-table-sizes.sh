#!/usr/bin/env bash
set -eou pipefail

input_dir=$1

for i in 1 2; do
  grep rox_central_postgres_table_size $input_dir/diagnostic_bundle_${i}/metrics-1 | grep process_indicators
  grep rox_central_postgres_table_size $input_dir/diagnostic_bundle_${i}/metrics-1 | grep deployments
  grep rox_central_postgres_table_size $input_dir/diagnostic_bundle_${i}/metrics-1 | grep process_baselines
  grep rox_central_postgres_table_size $input_dir/diagnostic_bundle_${i}/metrics-1 | grep alerts

  echo
  echo
  echo
done
