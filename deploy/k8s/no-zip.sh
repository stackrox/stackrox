#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export NAMESPACE="${NAMESPACE:-stackrox}"
echo "Kubernetes namespace set to $NAMESPACE"
kubectl create ns "$NAMESPACE" || true

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-central.stackrox:443}"
echo "In-cluster Central endpoint set to $CLUSTER_API_ENDPOINT"
echo

export PREVENT_IMAGE="stackrox/prevent:${PREVENT_IMAGE_TAG:-$(git describe --tags --abbrev=10 --dirty)}"

set -u

echo "Creating image pull secrets..."
PULL_SECRET_NAME="stackrox"
kubectl delete secret "$PULL_SECRET_NAME" --namespace "$NAMESPACE" || true
set +x
kubectl create secret docker-registry \
    "$PULL_SECRET_NAME" --namespace "$NAMESPACE" \
    --docker-server=https://index.docker.io/v1/ \
    --docker-username="$REGISTRY_USERNAME" \
    --docker-password="$REGISTRY_PASSWORD" \
    --docker-email="does-not-matter@stackrox.io"
echo

echo "Generating central config..."
docker run "$PREVENT_IMAGE" -t k8s -n "$NAMESPACE" -i "$PREVENT_IMAGE" > $K8S_DIR/central.zip
UNZIP_DIR="$K8S_DIR/central-deploy/"
rm -rf "$UNZIP_DIR"
unzip "$K8S_DIR/central.zip" -d "$UNZIP_DIR"
echo

echo "Deploying Central..."
kubectl delete secret -n "$NAMESPACE" central-tls || true
kubectl delete -f "$UNZIP_DIR/deploy.yaml" || true
$UNZIP_DIR/deploy.sh
echo

echo -n "Waiting for Central pod to be ready."
until [ "$(kubectl get pod -n $NAMESPACE --selector 'app=central' | grep Running | wc -l)" -eq 1 ]; do
    echo -n .
    sleep 1
done
echo

pkill -f "kubectl port-forward -n ${NAMESPACE}" || true
export CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" 8000:443 &
PID="$!"
echo "Port-forward launched with PID: $PID"
LOCAL_API_ENDPOINT=localhost:8000
echo "Set local API endpoint to: $LOCAL_API_ENDPOINT"

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
CLUSTER_ID=$(create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" KUBERNETES_CLUSTER "$PREVENT_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "\"namespace\": \"$NAMESPACE\", \"imagePullSecret\": \"stackrox\"")
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER_ID" "$K8S_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$K8S_DIR"

echo "Deploying Sensor..."
kubectl delete secret -n "$NAMESPACE" sensor-tls || true
$K8S_DIR/sensor-deploy.sh
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
