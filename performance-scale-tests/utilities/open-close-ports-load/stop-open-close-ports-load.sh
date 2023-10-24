#!/usr/bin/env bash
set -eoux pipefail

artifacts_dir=$1

export KUBECONFIG="$artifacts_dir"/kubeconfig

kubectl delete deployment open-close-ports-load
kubectl delete configmap open-close-ports-load-configmap
