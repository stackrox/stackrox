#!/bin/bash
#
# File modified from https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/marketplace/deployer_helm_base/create_manifests.sh
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eox pipefail

for i in "$@"
do
case $i in
  --mode=*)
    mode="${i#*=}"
    shift
    ;;
  *)
    >&2 echo "Unrecognized flag: $i"
    exit 1
    ;;
esac
done

[[ -z "$NAME" ]] && echo "NAME must be set" && exit 1
[[ -z "$NAMESPACE" ]] && echo "NAMESPACE must be set" && exit 1

echo "Creating the manifests for the kubernetes resources that build the application \"$NAME\""

data_dir="/data"
manifest_dir="$data_dir/manifest-expanded"
mkdir -p "$manifest_dir"

if [[ "$mode" = "test" ]]; then
  test_data_dir="/data-test"
  mkdir -p "/data-test"
fi

function extract_manifest() {
  data=$1
  extracted="$data/extracted"
  data_chart="$data/chart"
  mkdir -p "$extracted"


  # Expand the chart template.
  if [[ -d "$data_chart" ]]; then
    for chart in $(find "$data_chart" -maxdepth 1 -type f -name "*.tar.gz"); do
      chart_manifest_file=$(basename "$chart" | sed 's/.tar.gz$//')
      mkdir "$extracted/$chart_manifest_file"
      tar xfC "$chart" "$extracted/$chart_manifest_file"
    done
  fi
}

extract_manifest "$data_dir"

# Overwrite the templates using the test templates
if [[ "$mode" = "test" ]]; then
  extract_manifest "$test_data_dir"

  overlay_test_files.py \
    --manifest "$data_dir/extracted" \
    --test_manifest "$test_data_dir/extracted"
fi

# Log information and, at the same time, catch errors early and separately.
# This is a work around for the fact that process and command substitutions
# do not propagate errors.
echo "=== values.yaml ==="
/bin/print_config.py --output=yaml
echo "==================="

# Run helm expansion.
for chart in "$data_dir/extracted"/*; do
  chart_manifest_file=$(basename "$chart" | sed 's/.tar.gz$//').yaml
  ## BEGIN STACKROX MODIFICATIONS
  helm template "$chart/chart" \
    --name="$NAME" \
    --namespace="$NAMESPACE" \
    --values=<(/bin/print_config.py --output=yaml) \
    --set "namespace=$NAMESPACE" \
    > "$manifest_dir/$chart_manifest_file"
    ## END STACKROX MODIFICATIONS

  if [[ "$mode" != "test" ]]; then
    process_helm_hooks.py \
      --manifest "$manifest_dir/$chart_manifest_file"
  else
    process_helm_hooks.py --deploy_tests \
     --manifest "$manifest_dir/$chart_manifest_file"
  fi

  ensure_k8s_apps_labels.py \
    --manifest "$manifest_dir/$chart_manifest_file" \
    --appname "$NAME"
done
