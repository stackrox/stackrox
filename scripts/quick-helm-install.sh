#!/usr/bin/env bash
set -euo pipefail

echo "Adding the stackrox/helm-charts/opensource repository to Helm."

helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/

echo "Generating STACKROX_ADMIN_PASSWORD"

STACKROX_ADMIN_PASSWORD="$(openssl rand -base64 20 | tr -d '/=+')"

echo "Installing stackrox-central-services"

helm install -n stackrox --create-namespace stackrox-central-services stackrox/stackrox-central-services --set central.adminPassword.value="$STACKROX_ADMIN_PASSWORD"

kubectl -n stackrox rollout status deploy/central --timeout=3m

echo "Generating an init bundle with shared secrets."

kubectl -n stackrox exec -it deploy/central -- roxctl --insecure-skip-tls-verify \
  --password "${STACKROX_ADMIN_PASSWORD}" \
  central init-bundles generate stackrox-init-bundle --output - 0>/dev/null 1> stackrox-init-bundle.yaml 2>/dev/null

echo "Installing the first secured cluster"

helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  -f stackrox-init-bundle.yaml \
  --set clusterName="First-Secured-Cluster"

echo "You can add more secured clusters on different kube contexts using the following command:
helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \\
  -f stackrox-init-bundle.yaml \\
  --set clusterName=\"\$CLUSTER_NAME\"

STACKROX_ADMIN_PASSWORD = $STACKROX_ADMIN_PASSWORD
Above is your automatically generated stackrox admin password. Please store it securely, as you will need it during further configuration"

