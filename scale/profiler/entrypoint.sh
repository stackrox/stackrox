#! /bin/sh

set -eu

# Usage $0 <dir> <port>
DIR=${1:-/pprof}
PORT=${2:-8080}

/profiler --path "$DIR" --port "$PORT" &

/pprof.sh "$DIR" "central.stackrox:443"
