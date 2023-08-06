#!/usr/bin/bash
set -euox pipefail


cd $STACKROX_DIR

bundle=bundle-7.yaml


curl https://mirror.openshift.com/pub/rhacs/assets/latest/bin/Linux/roxctl --output roxctl_bin
chmod 755 roxctl_bin

./roxctl_bin -e https://"$CENTRAL_IP":443 -p "$ROX_ADMIN_PASSWORD" central init-bundles generate long-running-test --output "$bundle"

ls

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
)


helm install "${settings[@]}"

echo "::add-mask::$ROX_ADMIN_PASSWORD"
kubectl -n stackrox create secret generic access-rhacs --from-literal="username=${ROX_ADMIN_USERNAME}" --from-literal="password=${ROX_ADMIN_PASSWORD}" --from-literal="central_url=https://${CENTRAL_IP}":443

export KUBE_BURNER_VERSION=1.4.3

mkdir -p ./kube-burner

curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

kube_burner_config_file="$STACKROX_DIR"/.github/workflows/other-configs/cluster-density-kube-burner.yml 
kube_burner_gen_config_file="$STACKROX_DIR"/.github/workflows/other-configs/cluster-density-kube-burner_gen.yml 

sed "s|STACKROX_DIR|$STACKROX_DIR|" "$kube_burner_config_file" > "$kube_burner_gen_config_file" 

nohup "$STACKROX_DIR"/.github/workflows/scripts/repeate-kube-burner.sh ./kube-burner/kube-burner "$kube_burner_gen_config_file" &
