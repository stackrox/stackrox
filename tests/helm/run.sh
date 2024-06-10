#!/bin/bash

set -euo pipefail

# TODO: Parameterize
pushd charts/
../../../bin/linux_amd64/roxctl helm output central-services --image-defaults=development_build --remove --debug
../../../bin/linux_amd64/roxctl helm output secured-cluster-services --image-defaults=development_build --remove --debug
popd

# --disable-openapi-validation \
# --set central.persistence.none=true \

# TODO: Check syntax / common footguns on this use of find
for configuration_dir in $(find . -type f -name 'input_values.yaml' -print0 | xargs --null -n 1 dirname); do
  helm template --debug \
    -n stackrox \
    stackrox-central-services \
    charts/stackrox-central-services-chart \
    --values=./default_public_values.yaml \
    --values=./default_private_values.yaml \
    --values="${configuration_dir}"/input_values.yaml \
    > "${configuration_dir}"/central-services-installation.yaml


done
#
#echo "Rendering a dry run installation of the stackrox-secured-cluster-services helm charts as Kubernetes manifests:"
#for VERSION in "old" "new"; do
#  helm install --debug --dry-run \
#    -n stackrox \
#    --disable-openapi-validation \
#    --set clusterName=mergers-and-acquisitions \
#    --set ca.cert=a1 \
#    --set serviceTLS.cert=b2 \
#    --set serviceTLS.key=c3 \
#    --set createSecrets=false \
#    stackrox-secured-cluster-services \
#    "${TMP_ROOT}"/secured-cluster-services-${VERSION} \
#    > "${TMP_ROOT}"/secured-cluster-services-${VERSION}-installation.yaml