#!/usr/bin/env bash
set -eou pipefail

kube_burner_bin=$1
config=$2

while [ true ]; do
    "$kube_burner_bin" init --uuid=long-running-collector-kube-burner --config="$config" --skip-tls-verify --timeout=168h || true
done
