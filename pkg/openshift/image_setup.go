package openshift

// ImageSetup is the script that pulls and then pushes to the internal OpenShift registry
const ImageSetup = `
#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [ -z "$PREVENT_IMAGE_REGISTRY" ]; then
  echo -n "Enter StackRox Prevent Registry (e.g. stackrox.io or docker.io): "
  read PREVENT_IMAGE_REGISTRY
fi

if [ -z "$PREVENT_IMAGE_TAG" ]; then
  echo -n "Enter StackRox Prevent image tag: "
  read PREVENT_IMAGE_TAG
fi

if [ -z "$CLAIRIFY_IMAGE_TAG" ]; then
  echo -n "Enter StackRox Clairify image tag: "
  read CLAIRIFY_IMAGE_TAG
fi

if [ "$PREVENT_IMAGE_REGISTRY" = "stackrox.io" ]; then
	PREVENT_IMAGE_REPO="prevent"
	CLAIRIFY_IMAGE_REPO="clairify"
elif [ "$PREVENT_IMAGE_REGISTRY" = "docker.io" ]; then
	PREVENT_IMAGE_REPO="stackrox/prevent"
	CLAIRIFY_IMAGE_REPO="stackrox/clairify"
fi

if [ -z "$PREVENT_IMAGE_REPO" ]; then
	echo -n "Enter StackRox Prevent Repo: "
	read PREVENT_IMAGE_REPO
fi

if [ -z "$CLAIRIFY_IMAGE_REPO" ]; then
	echo -n "Enter StackRox Clairify Repo: "
	read CLAIRIFY_IMAGE_REPO
fi

PREVENT_IMAGE="${PREVENT_IMAGE_REGISTRY}/${PREVENT_IMAGE_REPO}:${PREVENT_IMAGE_TAG}"
CLAIRIFY_IMAGE="${PREVENT_IMAGE_REGISTRY}/${CLAIRIFY_IMAGE_REPO}:${CLAIRIFY_IMAGE_TAG}"

echo "Images to pull: ${PREVENT_IMAGE} and ${CLAIRIFY_IMAGE}. Does that look correct? Hit any key to continue. "
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

if [ -z "$PRIVATE_REGISTRY" ]; then
  PRIVATE_REGISTRY=$(oc get route -n default | grep docker-registry | tr -s ' ' | cut -d' ' -f2)
fi

echo "Private registry: $PRIVATE_REGISTRY"

oc get sa pusher > /dev/null || oc create sa pusher
oc policy add-role-to-user system:image-builder "system:serviceaccount:$OC_PROJECT:pusher"

sleep 2

set +x
TOKEN=$(oc serviceaccounts get-token pusher)
sudo docker login -u "anything" -p "$TOKEN" "$PRIVATE_REGISTRY"
set -x

# Pushing whatever we pulled except the <none> ones
echo "Sync the images to registry: $PRIVATE_REGISTRY"

sudo docker pull "${PREVENT_IMAGE}"
sudo docker tag "${PREVENT_IMAGE}" "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"
sudo docker push "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"

sudo docker pull "${CLAIRIFY_IMAGE}"
sudo docker tag "${CLAIRIFY_IMAGE}" "$PRIVATE_REGISTRY/$OC_PROJECT/clairify:$CLAIRIFY_IMAGE_TAG"
sudo docker push "$PRIVATE_REGISTRY/$OC_PROJECT/clairify:$CLAIRIFY_IMAGE_TAG"

oc project "$OC_PROJECT"
`
