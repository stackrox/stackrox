#!/bin/bash
set -e

# This script runs the netperf tcp_crr test via cillium/kubenetbench
# with a user configured number of streams.

DIR="$(cd "$(dirname "$0")" && pwd)"

source "$DIR"/../common.sh

artifacts_dir="$1"
knb_base_dir="$2"

[[ -n "$artifacts_dir" && -n "$knb_base_dir" ]] \
    || die "Usage: $0 <artifacts-dir> <knb-base-dir>"

log "teardown knb-monitor"
kubectl delete ds/knb-monitor || true
kubectl delete pod knb-cli || true
kubectl delete pod knb-srv || true


# $knb_dir contains test results that may be useful
#rm -rf "${knb_base_dir}"
