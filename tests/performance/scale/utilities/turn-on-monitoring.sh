#!/usr/bin/env bash
set -eo pipefail

artifacts_dir=$1
DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"

export KUBECONFIG="$artifacts_dir/kubeconfig"

oc apply -f "$DIR/../config/collector-monitoring.yml"

name=namespace/stackrox
if oc label "$name" openshift.io/cluster-monitoring="true" --overwrite=true | grep "${name} labeled"; then
  echo "Turning on monitoring and sleeping for 120 seconds"
  sleep 120
else
  echo "Monitoring is on for $name"
fi
