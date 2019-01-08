#! /bin/sh

if [ $# -eq 0 ]; then
  echo "Port to connect to on profiler is required as first argument"
  echo "usage: ./port-forward.sh 8080 <optional remote port>"
  exit 1
fi

local_port="$1"
remote_port="${2:-$local_port}"

echo "LOCAL PORT: ${local_port}"
echo "REMOTE PORT: ${remote_port}"

POD=$(kubectl -n stackrox get po -o jsonpath='{.items[?(@.metadata.labels.app=="profiler")].metadata.name}')
if [[ -z $POD ]]; then
  echo "Could not find profiler pod"
  exit 1
fi

kubectl port-forward -n stackrox "$POD" "${local_port}:${remote_port}" > /dev/null &
