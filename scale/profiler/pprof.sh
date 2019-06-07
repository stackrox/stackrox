#! /bin/sh

# Collect pprof profiles
#
# Runs pprof against the specified URL
#
# Usage:
#   pprof.sh OUTPUT_DIR ENDPOINT
#
# Example:
# $ ./pprof.sh . localhost:8000 5

set -e

usage() {
    echo "usage: ./pprof.sh <output dir> <endpoint> <num_iterations (optional)>"
}

curl_central() {
    if [[ -n $ROX_API_TOKEN ]]; then
        curl -sk -H "Authorization: Bearer $ROX_API_TOKEN" $@
    else
        curl -sk -u "admin:$ROX_PASSWORD" $@
    fi
}

pull_profiles() {
  echo "Iteration $1"
  formatted_date="$(date +%Y-%m-%d-%H-%M-%S)"
  curl_central "https://$ENDPOINT/debug/heap" > "$DIR/heap_${formatted_date}.tar.gz"
  curl_central "https://$ENDPOINT/debug/goroutine" > "$DIR/goroutine_${formatted_date}.tar.gz"
  curl_central "https://$ENDPOINT/debug/pprof/profile" > "$DIR/cpu_${formatted_date}.tar.gz"
}

if [[ -z $ROX_PASSWORD && -z $ROX_API_TOKEN ]]; then
  >&2 echo "Need to specify either ROX_PASSWORD or ROX_API_TOKEN"
  exit 1
fi

if [ $# -lt 2 ]; then
  usage
  exit 1
fi

DIR="$1"
ENDPOINT="$2"
NUM_ITERATIONS=${3:--1}

# Run the first one outside of the loop so that we don't have an extraneous sleep
pull_profiles 1

count=1
while [ $count -ne "$NUM_ITERATIONS" ]
do
  pull_profiles $((count+1))
  sleep 30
  count=$((count+1))
done
