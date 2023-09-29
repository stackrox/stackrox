#!/usr/bin/env bash
set -eou pipefail

# Given a directory with multiple artifacts for OSCI jobs looks through all of the collector logs
# and produces output for a csv file with the name of the job, the kernel version used in the job,
# and the collection method used by collector.

log_dir=$1

get_info_from_collector_log_file() {
    local log_file="$1"

    dir_name=$(dirname "$log_file" | grep -oP ".*ci-stackrox-stackrox-\K.*")
    kernel_version=$(grep "Kernel Version" "$log_file" | grep -oP 'Kernel Version: \K.*')
    probe_type=""

    if grep -q "Driver loaded into kernel: CO.RE eBPF probe" "$log_file"; then
        probe_type="core_bpf"
    elif grep -q "Driver loaded into kernel: collector-ebpf" "$log_file"; then
        probe_type="ebpf"
    fi

    pattern="^([^/]+)/"
    [[ $dir_name =~ $pattern ]] && extracted="${BASH_REMATCH[1]}"


    if [ -n "$dir_name" ] && [ -n "$kernel_version" ] && [ -n "$probe_type" ]; then
        echo "$extracted,$kernel_version,$probe_type"
    fi
}


cd "$log_dir"

export -f get_info_from_collector_log_file

collector_infos="$(find . -name '*collector.log' -exec bash -c 'get_info_from_collector_log_file "$1"' _ {} \;)"

collector_infos="$(echo "$collector_infos" | sort -u)"

echo "OSCI Job, Kernel Version, Collection method"
echo "$collector_infos"
