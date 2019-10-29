#!/usr/bin/env bash
set -eu

# This script launches a replay of previous sensor events
export NAME="remote"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

ROX_DIR="${DIR}/../.."
source "$ROX_DIR/deploy/common/deploy.sh"
source "$ROX_DIR/deploy/common/k8sbased.sh"
source "$ROX_DIR/deploy/common/env.sh"
source "$ROX_DIR/deploy/k8s/env.sh"

# set auth
export ROX_ADMIN_PASSWORD="${ROX_PASSWORD:-}"
if [ -z "$ROX_ADMIN_PASSWORD" ]; then
  echo >&2 "Please set ROX_PASSWORD before running this script."
  exit 1
fi

if [ -z "$GOOGLE_CREDENTIALS_RECORD_DB_FETCHER" ]; then
  echo >&2 "Please set GOOGLE_CREDENTIALS_RECORD_DB_FETCHER with the GCP service account for downloading the dump before running this script."
  exit 1
fi

kubectl -n stackrox create secret generic db-serviceaccount --from-literal=serviceaccount.json="$GOOGLE_CREDENTIALS_RECORD_DB_FETCHER"

API_ENDPOINT="${API_ENDPOINT-:localhost:8000}"

get_cluster_zip "${API_ENDPOINT}" "$NAME" KUBERNETES_CLUSTER "$MAIN_IMAGE" "central.stackrox:443" "$DIR" "true" ""

unzip_dir="$DIR/sensor-deploy"
rm -rf "$unzip_dir"
unzip "$DIR/sensor-deploy.zip" -d "$unzip_dir"
rm "$DIR/sensor-deploy.zip"
echo

kubectl delete secret -n stackrox sensor-tls  || true
kubectl create secret -n "stackrox" generic sensor-tls --from-file="$unzip_dir/sensor-cert.pem" \
 --from-file="$unzip_dir/sensor-key.pem" \
 --from-file="$unzip_dir/ca.pem"


echo "Launching replay sensor with tag: ${MAIN_IMAGE_TAG}"
newYAML="$DIR/replay.yaml"

envsubst < "$DIR/replay.yaml.tmp" > "$newYAML"

kubectl apply -f "${newYAML}"

pod="$(kubectl -n stackrox get pod -l app=replay -o jsonpath={.items[].metadata.name} 2>/dev/null || true)"
tries=0
while [[ -z "$pod" && "$tries" -lt 3 ]]; do
	tries=$((tries + 1))
	sleep 5
	pod=$(kubectl -n stackrox get pod -l app=replay -o jsonpath={.items[].metadata.name} 2>/dev/null || true)
done

if [[ -z "$pod" ]]; then
	echo >&2 "No replay pod found"
	exit 1
fi

kubectl -n stackrox wait --for=condition=ready po/${pod} --timeout=60s

# Clean up the replay artifacts
rm -rf $unzip_dir
rm "${newYAML}"
