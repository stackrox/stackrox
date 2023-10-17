#!/usr/bin/env bash
set -eou pipefail

name=$1
artifacts_dir=${2:-/tmp/artifacts}
collector_image_registry=${3:-quay.io/rhacs-eng}
collector_image_tag=${4:-3.7.3}

DIR="$(cd "$(dirname "$0")" && pwd)"

"$DIR"/wait-for-cluster.sh "$name"
infractl artifacts "$name" --download-dir "$artifacts_dir"
export KUBECONFIG="$artifacts_dir"/kubeconfig
"$DIR"/start-central-and-scanner.sh "$artifacts_dir"
"$DIR"/wait-for-pods.sh "$artifacts_dir"
"$DIR"/get-bundle.sh "$artifacts_dir"
"$DIR"/start-secured-cluster.sh "$artifacts_dir" "$collector_image_tag" "$collector_image_registry"
"$DIR"/wait-for-pods.sh "$artifacts_dir"
"$DIR"/turn-on-monitoring.sh "$artifacts_dir"
