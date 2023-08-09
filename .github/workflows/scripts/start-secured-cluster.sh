#!/usr/bin/bash
set -euox pipefail


cd $STACKROX_DIR

bundle=init-bundle.yaml

if [ ! -e roxctl_bin ]; then
    curl https://mirror.openshift.com/pub/rhacs/assets/latest/bin/Linux/roxctl --output roxctl_bin
    chmod 755 roxctl_bin
fi

./roxctl_bin -e https://"$CENTRAL_IP":443 -p "$ROX_ADMIN_PASSWORD" central init-bundles generate long-running-test --output "$bundle"

infractl artifacts "${SECURED_CLUSTER_NAME//./-}" --download-dir secured_cluster_artifacts
export KUBECONFIG=secured_cluster_artifacts/kubeconfig

helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts

image_registry=quay.io/stackrox-io
settings=(
    --namespace stackrox stackrox-secured-cluster-services --create-namespace rhacs/secured-cluster-services
    --values "$bundle"
    --set clusterName=secured-cluster-test
    --set image.collector.registry="$image_registry"
    --set image.collector.name="collector"
    --set image.collector.tag="$TAG"
    --set image.main.registry="$image_registry"
    --set image.main.name="main"
    --set image.main.tag="$TAG"
    --set centralEndpoint=https://"$CENTRAL_IP":443
)

helm install "${settings[@]}"

kubectl -n stackrox create secret generic access-rhacs --from-literal="username=${ROX_ADMIN_USERNAME}" --from-literal="password=${ROX_ADMIN_PASSWORD}" --from-literal="central_url=https://${CENTRAL_IP}":443
