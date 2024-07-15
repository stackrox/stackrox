#!/usr/bin/env bash
set -eou pipefail

# Given a directory with multiple artifacts for OSCI jobs looks through all of the collector logs
# and produces output for a csv file with the name of the job and the kernel version used in the job.

log_dir=$1

get_info_from_collector_log_file() {
    local log_file="$1"

    dir_name=$(dirname "$log_file" | grep -oP ".*ci-stackrox-stackrox-\K.*")
    kernel_version=$(grep "Kernel Version" "$log_file" | grep -oP 'Kernel Version: \K.*')

    pattern="^([^/]+)/"
    [[ $dir_name =~ $pattern ]] && extracted="${BASH_REMATCH[1]}"


    if [ -n "$dir_name" ] && [ -n "$kernel_version" ]; then
        echo "$extracted,$kernel_version"
    fi
}


cd "$log_dir"

export -f get_info_from_collector_log_file

collector_infos="$(find . -name '*collector.log' -exec bash -c 'get_info_from_collector_log_file "$1"' _ {} \;)"

collector_infos="$(echo "$collector_infos" | sort -u)"

echo "OSCI Job, Kernel Version"
echo "$collector_infos"
