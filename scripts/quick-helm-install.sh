#!/usr/bin/env bash
set -e

echo "Adding the stackrox/helm-charts/opensource repository to Helm."

helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/

echo "Generating stackrox-admin-password.txt"

openssl rand -base64 20 | tr -d '/=+' > stackrox-admin-password.txt
STACKROX_ADMIN_PASSWORD=`cat stackrox-admin-password.txt`

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

echo "You can add more secured clusters using the following command:
helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \\
  -f stackrox-init-bundle.yaml \\
  --set clusterName=\"\$CLUSTER_NAME\""

