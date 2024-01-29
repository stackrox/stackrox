#!/usr/bin/env bash
set -eou pipefail

num_ports=${1:-}
num_per_second=${2:-}
num_concurrent=${3:-}

if [[ -z $num_ports || -z $num_per_second || -z $num_concurrent ]]; then
    echo "Usage: num_ports num_per_second num_concurrent"
    exit 1
fi

run_open_close_ports_load_forever() {
    local start_port=$1
    local end_port=$2
    local num_per_second=$3

    while true; do
        /open-close-ports-load "$start_port" "$end_port" "$num_per_second" || true
    done
}

increment=$((num_ports / num_concurrent))
start_port=1
end_port=$increment

for ((i = 0; i < num_concurrent; i = i + 1)); do
    $(run_open_close_ports_load_forever "$start_port" "$end_port" "$num_per_second") &
    nohup bash -c "$(declare -f run_open_close_ports_load_forever)" &
    start_port=$((start_port + increment))
    end_port=$((end_port + increment))
done
