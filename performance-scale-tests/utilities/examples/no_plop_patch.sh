#!/usr/bin/env bash
set -eou pipefail

artifacts_dir=$1

DIR="$(cd "$(dirname "$0")" && pwd)"

echo "no plop patch"

export KUBECONFIG="$artifacts_dir"/kubeconfig

kubectl set env ds/collector ROX_PROCESSES_LISTENING_ON_PORT=false --namespace stackrox
kubectl set env deployment/central ROX_PROCESSES_LISTENING_ON_PORT=false --namespace stackrox
kubectl set env deployment/sensor ROX_PROCESSES_LISTENING_ON_PORT=false --namespace stackrox

sleep 10
