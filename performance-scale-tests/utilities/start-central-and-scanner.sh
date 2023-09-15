#!/usr/bin/env bash
set -eou pipefail

echo "Starting central and scanner related pods"

artifacts_dir=$1

export KUBECONFIG="$artifacts_dir"/kubeconfig
admin_password="$(cat "$artifacts_dir"/kubeadmin-password)"

settings=(
    --namespace stackrox stackrox-central-services --create-namespace rhacs/central-services
    --set central.exposure.route.enabled=true
    --set central.adminPassword.value="$admin_password"
    --set enableOpenShiftMonitoring=true
    --set central.db.enabled=true
)

if [[ -n ${DOCKER_USERNAME:-} ]]; then
    settings+=(--set imagePullSecrets.username="$DOCKER_USERNAME")
fi

if [[ -n ${DOCKER_PASSWORD:-} ]]; then
    settings+=(--set imagePullSecrets.password="$DOCKER_PASSWORD")
fi

if [[ -n ${CENTRAL_IMAGE_REGISTRY:-} ]]; then
    settings+=(--set central.image.registry="$CENTRAL_IMAGE_REGISTRY")
fi

if [[ -n ${CENTRAL_IMAGE_NAME:-} ]]; then
    settings+=(--set central.image.name="$CENTRAL_IMAGE_NAME")
fi

if [[ -n ${CENTRAL_IMAGE_TAG:-} ]]; then
    settings+=(--set central.image.tag="$CENTRAL_IMAGE_TAG")
fi

if [[ -n ${SCANNER_DBIMAGE_REGISTRY:-} ]]; then
    settings+=(--set scanner.dbImage.registry="$SCANNER_DBIMAGE_REGISTRY")
fi

if [[ -n ${SCANNER_DBIMAGE_NAME:-} ]]; then
    settings+=(--set scanner.dbImage.name="$SCANNER_DBIMAGE_NAME")
fi

if [[ -n ${SCANNER_DBIMAGE_TAG:-} ]]; then
    settings+=(--set scanner.dbImage.tag="$SCANNER_DBIMAGE_TAG")
fi

echo "Running: helm install ${settings[@]}"

helm install "${settings[@]}"
