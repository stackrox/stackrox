#!/usr/bin/env bash
set -eo pipefail

helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/
helm repo update

kubectx kind-kind
kubectl -n stackrox apply -f central-htpasswd

kind create cluster --config=hostpath-kind.yaml

# REVOKE
# ROX_USERNAME=1-simon roxctl central init-bundles revoke simon-bundle --insecure-skip-tls-verify -p 123 -e localhost:8443

# Issue init-bundles
ROX_USERNAME=1-simon roxctl central init-bundles generate simon-bundle --insecure-skip-tls-verify --output=init-bundle-simon.yaml -p 123 -e localhost:8443 || true
helm upgrade --install --create-namespace -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  --set clusterName="simon-tenant-cluster" \
  --set centralEndpoint="central.stackrox.svc:443" \
  -f init-bundle-simon.yaml

#kind create cluster --config=hostpath-kind.yaml kyle-cluster
#ROX_USERNAME=2-kyle roxctl central init-bundles generate kyle-bundle --insecure-skip-tls-verify --output=init-bundle-kyle.yaml -p 123 -e localhost:8443 || true
#kubectx kind-kyle
#helm upgrade --install --create-namespace -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
#  --set clusterName="kyle-tenant-cluster" \
#  --set centralEndpoint="localhost:8443" \
#  -f init-bundle-kyle.yaml
#

