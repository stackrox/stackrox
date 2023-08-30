#!/bin/bash
set -eou pipefail

log_dir=$1
cd $log_dir

# Find collector.log files and extract kernel version and probe type
#find | grep collector.log | while IFS= read -r -d $'\0' log_file; do
for log_file in `find | grep collector.log`; do
    dir_name=$(dirname "$log_file" | grep -oP ".*-rehearse-.*-pull-ci-stackrox-stackrox-\K.*")
    kernel_version=$(grep "Kernel Version" "$log_file" | grep -oP 'Kernel Version: \K.*')
    probe_type=""

    if grep -q " CO.RE eBPF probe" "$log_file"; then
        probe_type="core_bpf"
    elif grep -q "collector-ebpf" "$log_file"; then
        probe_type="ebpf"
    fi

    pattern="^([^/]+)/"
    [[ $dir_name =~ $pattern ]] && extracted="${BASH_REMATCH[1]}"


    #dir_name="$(echo $dir_name | sed 's|\\.*||')"
    if [ -n "$dir_name" ] && [ -n "$kernel_version" ] && [ -n "$probe_type" ]; then
        echo -e "$extracted\t$kernel_version\t$probe_type"
    fi
done | sort -u

