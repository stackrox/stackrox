#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ -z $DOCKER_USER ]]; then
    echo "DOCKER_USER env required"
    exit 1
fi

if [[ -z DOCKER_PASS ]]; then
    echo "DOCKER_PASS env required"
    exit 1
fi

OC_PROJECT="${OC_PROJECT:-stackrox}"
PREVENT_IMAGE_TAG="${PREVENT_IMAGE_TAG:-1.0}"
ROX_IMAGE_REPO_REGISTRY=${ROX_IMAGE_REPO_REGISTRY:-docker.io}
echo "docker login to $ROX_IMAGE_REPO_REGISTRY"

set +x
sudo docker login -u "$DOCKER_USER" -p "$DOCKER_PASS"
set -x

oc new-project "$OC_PROJECT" || true
oc project "$OC_PROJECT"

PRIVATE_REGISTRY=$(oc get route -n default | grep docker-registry | tr -s ' ' | cut -d' ' -f2)
echo "Private registry: $PRIVATE_REGISTRY"

sudo docker pull "$ROX_IMAGE_REPO_REGISTRY/stackrox/prevent:$PREVENT_IMAGE_TAG"
sudo docker tag "$ROX_IMAGE_REPO_REGISTRY/stackrox/prevent:$PREVENT_IMAGE_TAG" "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"

# Pushing whatever we pulled except the <none> ones
echo "Sync the images to registry: $PRIVATE_REGISTRY"

oc create sa pusher || true
oc policy add-role-to-user system:image-builder "system:serviceaccount:$OC_PROJECT:pusher"

sleep 2

set +x
TOKEN=$(oc serviceaccounts get-token pusher)
sudo docker login -u "anything" -p "$TOKEN" "$PRIVATE_REGISTRY"
set -x

sudo docker push "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"

echo "Creating secrets..."
oc delete secrets stackrox || true
set +x
oc secrets new-dockercfg stackrox --docker-server="$PRIVATE_REGISTRY" --docker-username="anything" --docker-password="$TOKEN" --docker-email="support@stackrox.com"
set -x

oc project "$OC_PROJECT"
