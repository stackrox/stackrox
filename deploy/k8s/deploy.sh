#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
COMMON_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )"/../common && pwd)"

source $COMMON_DIR/deploy.sh

export NAMESPACE="${NAMESPACE:-stackrox}"
echo "Kubernetes namespace set to $NAMESPACE"
kubectl create ns "$NAMESPACE" || true

export CLUSTER_API_ENDPOINT="${CLUSTER_API_ENDPOINT:-apollo.stackrox:443}"
echo "In-cluster Apollo endpoint set to $CLUSTER_API_ENDPOINT"
echo

set -u

generate_ca "$K8S_DIR"

echo "Creating image pull secrets..."
PULL_SECRET_NAME="stackrox"
kubectl delete secret "$PULL_SECRET_NAME" --namespace "$NAMESPACE" || true
set +x
kubectl create secret docker-registry \
    "$PULL_SECRET_NAME" --namespace "$NAMESPACE" \
    --docker-server=https://index.docker.io/v1/ \
    --docker-username="$DOCKER_USER" \
    --docker-password="$DOCKER_PASS" \
    --docker-email="does-not-matter@stackrox.io"
echo

echo "Deploying Central..."
kubectl delete secret -n "$NAMESPACE" central-tls || true
kubectl create secret -n "$NAMESPACE" generic central-tls --from-file="$K8S_DIR/ca.pem" --from-file="$K8S_DIR/ca-key.pem"
kubectl delete -f "$K8S_DIR/central.yaml" || true
cat "$K8S_DIR/central.yaml" | sed "s|stackrox/apollo:latest|$APOLLO_IMAGE|" | kubectl create -f -
echo

echo -n "Waiting for Apollo pod to be ready."
until [ "$(kubectl get pod -n $NAMESPACE --selector 'app=apollo' | grep Running | wc -l)" -eq 1 ]; do
    echo -n .
    sleep 1
done
echo

pkill -f "kubectl port-forward -n ${NAMESPACE}" || true
export CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=apollo' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" 8000:443 &
PID="$!"
echo "Port-forward launched with PID: $PID"
LOCAL_API_ENDPOINT=localhost:8000
echo "Set local API endpoint to: $LOCAL_API_ENDPOINT"

wait_for_central "$LOCAL_API_ENDPOINT"
CLUSTER="remote"
create_cluster "$LOCAL_API_ENDPOINT" "$CLUSTER" KUBERNETES_CLUSTER "$APOLLO_IMAGE" "$CLUSTER_API_ENDPOINT" "$K8S_DIR" "\"namespace\": \"$NAMESPACE\", \"imagePullSecret\": \"stackrox\""
get_identity "$LOCAL_API_ENDPOINT" "$CLUSTER" "$K8S_DIR"
get_authority "$LOCAL_API_ENDPOINT" "$K8S_DIR"

echo "Deploying Sensor..."
kubectl delete secret -n "$NAMESPACE" sensor-tls || true
kubectl create secret -n "$NAMESPACE" generic sensor-tls --from-file="$K8S_DIR/sensor-$CLUSTER-cert.pem" --from-file="$K8S_DIR/sensor-$CLUSTER-key.pem" --from-file="$K8S_DIR/central-ca.pem"
kubectl create -f "$K8S_DIR/sensor-$CLUSTER_NAME-deploy.yaml"
echo

echo "Successfully deployed!"
echo "Access the UI at: https://$LOCAL_API_ENDPOINT"
