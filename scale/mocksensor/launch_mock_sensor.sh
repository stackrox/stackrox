#!/usr/bin/env bash
set -eu

# This launches mock_sensor with the tag defined by `make tag`.
# Any arguments passed to this script are passed on to the mocksensor program.
# Example: ./launch_mock_sensor.sh 1 -max-deployments 100 will launch a deployment "mock-sensor-1" with args -max-deployments 100.

if [ $# -lt 1 ]; then
  echo "usage: $0 <mock sensor suffix> [<mock sensor args ...>]"
  echo "e.g. ./launch_mock_sensor.sh 1"
  exit 1
fi

export NAME="$1"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

ROX_DIR="${DIR}/../.."
# shellcheck source=../../deploy/common/deploy.sh
source "$ROX_DIR/deploy/common/deploy.sh"
# shellcheck source=../../deploy/common/k8sbased.sh
source "$ROX_DIR/deploy/common/k8sbased.sh"
# shellcheck source=../../deploy/common/env.sh
source "$ROX_DIR/deploy/common/env.sh"
# shellcheck source=../../deploy/k8s/env.sh
source "$ROX_DIR/deploy/k8s/env.sh"

# set auth
export ROX_ADMIN_PASSWORD="${ROX_PASSWORD:-}"
if [ -z "$ROX_ADMIN_PASSWORD" ]; then
  echo >&2 "Please set ROX_PASSWORD before running this script."
  exit 1
fi

API_ENDPOINT="${API_ENDPOINT-:localhost:8000}"

get_cluster_zip "${API_ENDPOINT}" "mock-cluster-$1" KUBERNETES_CLUSTER "$MAIN_IMAGE" "central.stackrox:443" "$DIR" "default" ""

unzip_dir="$DIR/sensor-deploy"
rm -rf "$unzip_dir"
unzip "$DIR/sensor-deploy.zip" -d "$unzip_dir"
rm "$DIR/sensor-deploy.zip"
echo

kubectl create secret -n "stackrox" generic sensor-tls-$1 --from-file="$unzip_dir/sensor-cert.pem" \
 --from-file="$unzip_dir/sensor-key.pem" \
 --from-file="$unzip_dir/ca.pem"


echo "Launching mock sensor with tag: ${MAIN_IMAGE_TAG}"
newYAML="$DIR/rendered-mock-sensor.yaml"

# Add the extra arguments
if [ $# -gt 1 ]; then
  shift
  export ARGS="$(printf '%s\n' "$@" | jq -R . | jq -cs .)"
else
  export ARGS="[]"
fi
envsubst < "$DIR/mocksensor.yaml.tmpl" > "$newYAML"

kubectl -n stackrox delete deploy/sensor 2>/dev/null || true
kubectl create -f "${newYAML}"

# Clean up the mock sensor artifacts
rm -rf $unzip_dir
rm "${newYAML}"
