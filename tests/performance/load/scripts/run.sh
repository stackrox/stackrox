#!/usr/bin/env bash
set -eou pipefail

if ! command -v npm &> /dev/null; then
   echo "npm is missing. Install it"
   exit 1
fi

if ! command -v k6 &> /dev/null; then
   echo "k6 is missing. Install it"
   exit 1
fi

if [ $# -lt 1 ]; then
  echo "Usage: $0 <out_dir> [STACKROX_DIR]"
  exit 1
fi

out_dir=$1
STACKROX_DIR=${2:-}

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"


if [ -z "${STACKROX_DIR:-}" ]; then
  if ! (cd "$DIR" && git rev-parse --is-inside-work-tree > /dev/null 2>&1); then
     echo "$DIR is not in a git repository. It must be in order to run"
     exit 1
  fi
  STACKROX_DIR="$(cd "$DIR" && git rev-parse --show-toplevel)"
fi


SCRIPT_DIR="$HOME"

cd "${STACKROX_DIR}/tests/performance/load"

mkdir -p "$out_dir" || true

"${STACKROX_DIR}/scale/dev/port-forward.sh" 8000
buffer_time=30
num_iter=5

for (( i = 0; i < num_iter ; i=i+1 )); do
  for vus in 1 5; do
    "${SCRIPT_DIR}/monitor-top-pod.sh" &> "${out_dir}/k6-test-result--default-5--vus-${vus}--run-${i}--top-pod.txt" &
    PID=$!
    sleep "$buffer_time"
    npm test -- --out csv="${out_dir}/k6-test-result--default-5--vus-${vus}--run-${i}.csv" --quiet --iterations 5 --vus "${vus}" --duration 25m
    sleep "$buffer_time"
    kill -9 "$PID"
  done
done


"${SCRIPT_DIR}/get-stackrox-info.sh" "$out_dir"
