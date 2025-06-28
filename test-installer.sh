#!/bin/bash
set -e

# Read namespace from installer.yaml
NAMESPACE=$(yq .namespace installer.yaml)
if [ -z "$NAMESPACE" ]; then
    NAMESPACE="stackrox"
fi

echo "Using namespace: $NAMESPACE"

echo "Building installer..."
make bin/installer

echo "Cleaning up existing resources..."
kubectl delete validatingwebhookconfiguration stackrox --ignore-not-found
kubectl delete namespace $NAMESPACE --ignore-not-found

echo "Deploying central..."
./bin/installer apply central

echo "Waiting for central deployment to be ready..."
kubectl wait --for=condition=available deployment/central -n $NAMESPACE --timeout=300s

echo "Waiting extra time for central to be fully ready..."
sleep 10

echo "Deploying CRS..."
./bin/installer apply crs

echo "Deploying secured cluster..."
./bin/installer apply securedcluster

echo "Waiting for all deployments to be ready..."
kubectl wait --for=condition=available deployment --all -n $NAMESPACE --timeout=300s

echo "Testing webhook validation..."
if kubectl create deployment test-nginx --image=nginx --dry-run=server > /dev/null 2>&1; then
    echo "✓ Webhook validation successful"
else
    echo "✗ Webhook validation failed"
    kubectl create deployment test-nginx --image=nginx --dry-run=server
    exit 1
fi

echo "All tests passed!"
