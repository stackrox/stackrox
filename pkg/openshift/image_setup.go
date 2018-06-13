package openshift

// ImageSetup is the script that pulls and then pushes to the internal OpenShift registry
const ImageSetup = `
#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [ -z "$PREVENT_IMAGE_REGISTRY" ]; then
  echo -n "Enter StackRox Prevent Registry (e.g. stackrox.io or docker.io): "
  read PREVENT_IMAGE_REGISTRY
  echo
fi

if [ -z "$PREVENT_IMAGE_REPO" ]; then
  echo -n "Enter StackRox Prevent Repo (e.g. prevent or stackrox/prevent): "
  read PREVENT_IMAGE_REPO
  echo
fi

if [ -z "$PREVENT_IMAGE_TAG" ]; then
  echo -n "Enter StackRox Prevent image tag: "
  read PREVENT_IMAGE_TAG
  echo
fi

FULL_IMAGE="${PREVENT_IMAGE_REGISTRY}/${PREVENT_IMAGE_REPO}:${PREVENT_IMAGE_TAG}"
echo "Full image to pull is ${FULL_IMAGE}. Does that look correct? Hit any key to continue. "
read -s -n 1

if [ -z "$REGISTRY_USERNAME" ]; then
  echo -n "Registry username for StackRox Prevent image from $PREVENT_IMAGE_REGISTRY: "
  read REGISTRY_USERNAME
  echo
fi
if [ -z "$REGISTRY_PASSWORD" ]; then
  echo -n "Registry password for StackRox Prevent image from $PREVENT_IMAGE_REGISTRY: "
  read -s REGISTRY_PASSWORD
  echo
fi

OC_PROJECT="${OC_PROJECT:-stackrox}"
echo "docker login to $PREVENT_IMAGE_REGISTRY"

set +x
sudo docker login -u "$REGISTRY_USERNAME" -p "$REGISTRY_PASSWORD" "$PREVENT_IMAGE_REGISTRY"
set -x

oc new-project "$OC_PROJECT" || true
oc project "$OC_PROJECT"

PRIVATE_REGISTRY=$(oc get route -n default | grep docker-registry | tr -s ' ' | cut -d' ' -f2)
echo "Private registry: $PRIVATE_REGISTRY"

sudo docker pull "${FULL_IMAGE}"
sudo docker tag "${FULL_IMAGE}" "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"

# Pushing whatever we pulled except the <none> ones
echo "Sync the images to registry: $PRIVATE_REGISTRY"

oc get sa pusher > /dev/null || oc create sa pusher
oc policy add-role-to-user system:image-builder "system:serviceaccount:$OC_PROJECT:pusher"

sleep 2

set +x
TOKEN=$(oc serviceaccounts get-token pusher)
sudo docker login -u "anything" -p "$TOKEN" "$PRIVATE_REGISTRY"
set -x

sudo docker push "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"

oc project "$OC_PROJECT"
`
