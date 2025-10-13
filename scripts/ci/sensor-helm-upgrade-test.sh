#!/usr/bin/env bash

# This script takes the 5 most recent major releases from the sensor helm chart of the stackrox/helm-charts
# repository, deploys the sensor from this version, and performs an upgrade to the current secured-cluster-services
# chart.

set -eo pipefail

function curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

function roxcurl() {
  local url="$1"
  shift
  curl --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") -k "https://${API_ENDPOINT}${url}" "$@"
}

function kcr() {
  kubectl -n stackrox "$@"
}

# wait_for_stabilize waits until all pods are ready and not being deleted for three consecutive seconds. It fails if
# this condition is not met after 120 seconds.
function wait_for_stabilize() {
  local offending_pods=""
  local successes=0
  for i in {1..120}; do
    local new_offending_pods
    new_offending_pods="$(
      kcr get po -o json |
        jq '[(.items[] // []) | select(.metadata.labels.app | (. == "sensor" or . == "collector" or . == "admission-control")) | select(.metadata.deletionTimestamp != null or ((.status.containerStatuses // []) | all(.ready) | not)) | .metadata.name] | sort | join(", ")' -r
        )"
    if [[ -z "$new_offending_pods" ]]; then
      successes=$((successes + 1))
      if (( successes >= 3 )); then
        echo >&2
        return 0
      fi
    else
      if [[ "$new_offending_pods" != "$offending_pods" ]]; then
        echo >&2
        echo >&2 "Offending pods: ${new_offending_pods}"
        offending_pods="$new_offending_pods"
      fi
      successes=0
    fi
    sleep 1
    echo >&2 -n "."
  done

  die "Failed to wait for deployment stabilization!"
}

function die() {
  echo >&2 "$@"
  exit 1
}

# Perform the actual test
echo "Deleting any existing cluster"
for cluster_id in $(roxcurl /v1/clusters | jq '.clusters[] | .id' -r); do
  roxcurl "/v1/clusters/${cluster_id}" -XDELETE
done

echo "Disabling sensor auto-upgrade"
roxcurl /v1/sensorupgrades/config -d'{"config":{"enableAutoUpgrades":false}}'

echo "Generating an API token"
api_token="$(
  roxcurl /v1/apitokens/generate -d '{"name": "helm-upgrade-test", "roles": ["Sensor Creator"]}' | jq -r '.token'
  )"

echo "Generating new Helm chart"
new_helm_chart_dir="$(mktemp -d)"
roxctl helm output secured-cluster-services --remove --output-dir "$new_helm_chart_dir"

echo "Cloning upstream Helm charts repo"
helm_charts_repo="$(mktemp -d)"
git clone git@github.com:stackrox/helm-charts.git "$helm_charts_repo"

echo "Waiting for stabilization before initial deployment ..."
wait_for_stabilize

for i in 50.0 51.0 52.0 54.0 53.0; do
  cd "${helm_charts_repo}/3.0.${i}"
  [[ ! -d sensor/ ]] || cd sensor

  # We need to modify the values.yaml file in order for the deployment script to work
  echo "Patching values.yaml"
  mv values.yaml values.yaml.orig
  yq e -j values.yaml.orig | jq '.cluster.name="test" | .image.registry.main="quay.io" | .image.repository.main="rhacs-eng/main" | .image.registry.collector="quay.io" | .image.repository.collector="rhacs-eng/collector" | .config.slimCollector=true' | yq e -P - >values.yaml

  echo "Installing version ${i} of the old Helm chart"
  ROX_API_TOKEN="$api_token" ./scripts/setup.sh -e "$API_ENDPOINT" -f values.yaml
  helm -n stackrox install sensor . -f values.yaml
  wait_for_stabilize

  echo "Upgrading to current version of the Helm chart"
  helm -n stackrox upgrade sensor "$new_helm_chart_dir" --reuse-values -f <("${new_helm_chart_dir}/scripts/fetch-secrets.sh")
  wait_for_stabilize

  echo "Uninstalling Helm chart"
  helm -n stackrox uninstall sensor
  wait_for_stabilize
  kcr delete --ignore-not-found secrets/sensor-tls secrets/collector-tls secrets/admission-control-tls

  echo "Deleting cluster"
  roxcurl "/v1/clusters/$(roxcurl /v1/clusters | jq '.clusters[0].id' -r)" -XDELETE
done
