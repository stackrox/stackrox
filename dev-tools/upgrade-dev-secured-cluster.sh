#!/usr/bin/env bash
set -eo pipefail

# Upgrade the secured cluster services deployed by the ./deployed scripts.
# Usage: ./upgrade-dev-secured-cluster <helm-flags...>

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

custom_helm_args=("$@")
# ROX-37649: Sensor chart now at deploy/sensor-deploy since deploy/k8s was removed
secured_cluster_chart_path="$DIR/../deploy/sensor-deploy/chart"

roxctl helm output secured-cluster-services --image-defaults=development_build --output-dir "$secured_cluster_chart_path" --remove --debug

if [[ ! -d "$secured_cluster_chart_path" ]]; then
  echo "Could not find chart: $secured_cluster_chart_path"
  exit 1
fi

helm_args=(
  --reuse-values
)

default_init_bundle_path="$secured_cluster_chart_path/../init-bundle.yaml"
if [[ -f  "$default_init_bundle_path" ]]; then
  echo "Using init bundle: $default_init_bundle_path"
  helm_args+=(
    -f "$default_init_bundle_path"
  )
fi

MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-$(make --quiet --no-print-directory -C "$DIR/.." tag)}"

helm_args+=(
  --set "image.main.tag=$MAIN_IMAGE_TAG"
  "${custom_helm_args[@]}"
)

helm -n stackrox upgrade --install stackrox-secured-cluster-services "$secured_cluster_chart_path" "${helm_args[@]}"
