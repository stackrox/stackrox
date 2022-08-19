#!/bin/sh
set -eu

# Compare the metrics found in different zipped stackrox_debug dumps. This is
# intended to run in CI to compare a BASELINE versus a system under test run.

usage() {
    echo "$0 <baseline.zip> <to_compare.zip> <table-output.html>"
    echo "e.g. $0 stackrox_debug_2020_05_20_09_51_46.zip stackrox_debug_2020_05_21_09_54_08.zip out.html"
}

metrics_of_interest=\
'rox_central_sensor_event_queue,'\
'rox_central_sensor_event_duration,'\
'process_cpu_seconds_total,'\
'rox_central_index_op_duration,'\
'rox_central_rocksdb_op_duration,'\
'rox_central_function_segment_duration,'\
'rox_central_datastore_function_duration,'\
'rox_central_k8s_event_processing_duration'

main() {
    if [ "$#" -ne 3 ]; then
        usage
        exit 1
    fi

    baseline="$1"
    to_compare="$2"
    comparison_out="$3"

    unzip -o "${baseline}" -d baseline
    unzip -o "${to_compare}" -d to_compare

    prometheus-metric-parser compare --old-file baseline/metrics-2 --new-file to_compare/metrics-2 \
        --metrics ${metrics_of_interest} --warn 15 --error 25 \
        --format=html-table > "${comparison_out}" || true

    prometheus-metric-parser compare --old-file baseline/metrics-2 --new-file to_compare/metrics-2 \
        --metrics ${metrics_of_interest} --warn 15 --error
}

main "$@"
