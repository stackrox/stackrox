#! /bin/sh

set -eu

# Launch a profiler that collects pprof profiles periodically
#
# Usage:
#   ./launch-profiler.sh <dir> <port>
#
# Example:
# $ ./launch-profiler.sh  - launches a profiler on port :8080 and writes to /pprof
# $ /.launch-profiler.sh /dir 9999  - launches a profiler on port :9999 and writes to /dir

usage() {
    echo "usage: ./launch-profiler.sh <dir> <port>"
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [ $# -gt 2 ]; then
  usage
  exit 1
fi

if [ -z ${ROX_PASSWORD+x} ]; then
  echo "ROX_PASSWORD must be set to authenticate against profiling endpoints"
  exit 1
fi

export OUTPUT_DIR=${1:-/pprof}
export PORT=${2:-"8080"}

TAG=${MAIN_IMAGE_TAG:-}
if [ -z $TAG ]; then
  TAG=$(git -C "$DIR" describe --tags --abbrev=10 --dirty)
fi
export TAG

envsubst < "$DIR/deploy.yaml" | kubectl apply -f -
