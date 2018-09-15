#!/bin/bash

set -e

if [ -z "$PREVENT_IMAGE_REGISTRY" ]; then
  echo "This script pulls the StackRox image and pushes it to the OpenShift registry."
  echo "We first need to know which image to pull, and from where."
  echo "Most users can use the defaults."
  echo
  echo "The options are: 'stackrox.io' or 'docker.io'."
  echo -n "Which registry will you deploy from? (default: stackrox.io): "
  read PREVENT_IMAGE_REGISTRY
  PREVENT_IMAGE_REGISTRY="${PREVENT_IMAGE_REGISTRY:-stackrox.io}"
fi

if [ -z "$PREVENT_IMAGE_TAG" ]; then
  echo -n "Enter StackRox Prevent image tag (default: {{.ImageTag}}): "
  read PREVENT_IMAGE_TAG
  PREVENT_IMAGE_TAG="${PREVENT_IMAGE_TAG:-{{.ImageTag}}}"
fi

if [ "$PREVENT_IMAGE_REGISTRY" = "stackrox.io" ]; then
	PREVENT_IMAGE_REPO="prevent"
elif [ "$PREVENT_IMAGE_REGISTRY" = "docker.io" ]; then
	PREVENT_IMAGE_REPO="stackrox/prevent"
fi

if [ -z "$PREVENT_IMAGE_REPO" ]; then
	echo -n "Enter StackRox Prevent Repo (default: prevent): "
	read PREVENT_IMAGE_REPO
  PREVENT_IMAGE_REPO="${PREVENT_IMAGE_REPO:-prevent}"
fi

PREVENT_IMAGE="${PREVENT_IMAGE_REGISTRY}/${PREVENT_IMAGE_REPO}:${PREVENT_IMAGE_TAG}"

echo "Image to pull: ${PREVENT_IMAGE}"
echo -n "Does that look correct? Hit any key to continue, or ctrl-C to exit. "
read -s -n 1
echo

# Set USE_SUDO=true or USE_SUDO=false to skip auto-detection.
DOCKER=()
if [ -z "$USE_SUDO" ]; then
  echo "Detecting whether sudo is required for use of Docker commands..."
  docker version > /dev/null || DOCKER+=("sudo")
else
  if [ "${USE_SUDO:0:1}" = "t" ] || [ "${USE_SUDO:0:1}" = "T" ]; then
    echo "Sudo enabled by USE_SUDO=$USE_SUDO"
    DOCKER+=("sudo")
  else
    echo "Sudo disabled by USE_SUDO=$USE_SUDO"
  fi
fi
DOCKER+=("docker")
echo "Testing: ${DOCKER[*]} version"
"${DOCKER[@]}" version --format 'Running Docker {{`{{.Server.Version}}`}}'

echo "Please enter your credentials to login to $PREVENT_IMAGE_REGISTRY"
# To use this script without an interactive shell, set REGISTRY_USERNAME and REGISTRY_PASSWORD.
if [ -n "$REGISTRY_USERNAME" ] && [ -n "$REGISTRY_PASSWORD" ]; then
    "${DOCKER[@]}" login -u "$REGISTRY_USERNAME" -p "$REGISTRY_PASSWORD" "$PREVENT_IMAGE_REGISTRY"
else
    "${DOCKER[@]}" login "$PREVENT_IMAGE_REGISTRY"
fi

"${DOCKER[@]}" pull "${PREVENT_IMAGE}"

OC_PROJECT="${OC_PROJECT:-{{.Namespace}}}"
oc new-project "$OC_PROJECT" || true
oc project "$OC_PROJECT"

PRIVATE_REGISTRY="${PRIVATE_REGISTRY:-$(oc get route -n default docker-registry --output=jsonpath='{.spec.host}')}"
echo "OpenShift registry: $PRIVATE_REGISTRY"

oc get sa pusher > /dev/null || oc create sa pusher
oc policy add-role-to-user system:image-builder "system:serviceaccount:$OC_PROJECT:pusher"

sleep 2

TOKEN="$(oc serviceaccounts get-token pusher)"
"${DOCKER[@]}" login -u "anything" -p "$TOKEN" "$PRIVATE_REGISTRY"

echo "Pulling and pushing images to $PRIVATE_REGISTRY"
"${DOCKER[@]}" tag "${PREVENT_IMAGE}" "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"
"${DOCKER[@]}" push "$PRIVATE_REGISTRY/$OC_PROJECT/prevent:$PREVENT_IMAGE_TAG"
