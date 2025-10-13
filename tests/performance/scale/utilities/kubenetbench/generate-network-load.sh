#!/bin/bash
set -e

# This script runs the netperf tcp_crr test via cillium/kubenetbench
# with a user configured number of streams.

DIR="$(cd "$(dirname "$0")" && pwd)"

source "$DIR"/../common.sh

artifacts_dir="$1"
test_name="$2"
num_streams="$3"
knb_base_dir="$4"
load_duration=${5:-600}

[[ -n "$artifacts_dir" && -n "$test_name" && -n "${num_streams}" && -n "${knb_base_dir}" ]] \
    || die "Usage: $0 <artifacts-dir> <test-name> <num-streams> <knb-base-dir> [load-duration]"

export KUBECONFIG="${artifacts_dir}/kubeconfig"
log "run netperf tcp_crr with ${num_streams} streams"
"${knb_base_dir}/kubenetbench-master/${test_name}/knb" pod2pod -b netperf --netperf-type tcp_crr --netperf-nstreams "${num_streams}" -t "${load_duration}"
