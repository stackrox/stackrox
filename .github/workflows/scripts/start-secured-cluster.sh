#!/usr/bin/bash
set -euox pipefail


cd $STACKROX_DIR

export CLUSTER_API_ENDPOINT=https://"${CENTRAL_IP}":443
export API_ENDPOINT="${CENTRAL_IP}":443
export MAIN_IMAGE_TAG=$TAG
export CLUSTER=secured-cluster

./deploy/k8s/sensor.sh

kubectl -n stackrox create secret generic access-rhacs --from-literal="username=${ROX_ADMIN_USERNAME}" --from-literal="password=${ROX_ADMIN_PASSWORD}" --from-literal="central_url=https://${CENTRAL_IP}":443
