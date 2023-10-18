#!/usr/bin/env bash
set -eou pipefail

kube_burner_path=$1
config=$2
uuid=$3
artifacts_dir=$4
load_duration=$5

export KUBECONFIG="$artifacts_dir"/kubeconfig

echo "Starting kube-burner"

"$kube_burner_path" init --uuid="$uuid" --config="$config" --skip-tls-verify --timeout="$load_duration"s
