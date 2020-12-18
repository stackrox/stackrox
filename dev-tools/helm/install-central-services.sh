#!/usr/bin/env bash
set -eo pipefail

# Install central services based on the new helm charts.
# Usage:
#
# $ cdrox # change to rox root directory
# $ roxctl helm output central-services # generate helm chart
# $ ./dev-tools/helm/install-central-services.sh

SCRIPT="$(python -c 'import os, sys; print(os.path.realpath(sys.argv[1]))' "${BASH_SOURCE[0]}")"
source "$(dirname "$SCRIPT")/common-vars.sh"

CENTRAL_PASSWORD="$(cat ./deploy/k8s/central-deploy/password)"

helm upgrade --install -n stackrox stackrox-central-services --create-namespace  ./stackrox-central-services-chart \
    -f ./dev-tools/helm/central-services/docker-values-public.yaml \
    --set-file licenseKey=./deploy/common/dev-license.lic \
    --set imagePullSecrets.username="${DOCKER_USER}" \
    --set imagePullSecrets.password="${DOCKER_PASSWORD}" \
    --set central.adminPassword.value="${CENTRAL_PASSWORD}" \
    --set central.image.tag="${MAIN_IMAGE_TAG}"

echo "Deployed image: ${MAIN_IMAGE_TAG}"
echo "Admin password is: ${CENTRAL_PASSWORD}"
