#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

WD=$(pwd)
cd $DIR

# Create registry-auth secret, used to pull the benchmark image.
if [ -z "$REGISTRY_USERNAME" ]; then
  echo -n "Registry username for StackRox Prevent image: "
  read REGISTRY_USERNAME
  echo
fi
if [ -z "$REGISTRY_PASSWORD" ]; then
  echo -n "Registry password for StackRox Prevent image: "
  read -s REGISTRY_PASSWORD
  echo
fi

# unset the host path so we can get the registry auth locally
OLD_DOCKER_HOST="$DOCKER_HOST"
OLD_DOCKER_CERT_PATH="$DOCKER_CERT_PATH"
OLD_DOCKER_TLS_VERIFY="$DOCKER_TLS_VERIFY"
unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY

docker run --rm --entrypoint=base64 -e REGISTRY_USERNAME="$REGISTRY_USERNAME" -e REGISTRY_PASSWORD="$REGISTRY_PASSWORD" {{.Image}} > registry-auth

export DOCKER_HOST="$OLD_DOCKER_HOST"
export DOCKER_CERT_PATH="$OLD_DOCKER_CERT_PATH"
export DOCKER_TLS_VERIFY="$OLD_DOCKER_TLS_VERIFY"


# Gather client cert bundle if it is present.
if [ -n "$DOCKER_CERT_PATH" ]; then
  cp $DOCKER_CERT_PATH/ca.pem ./docker-ca.pem
  cp $DOCKER_CERT_PATH/key.pem ./docker-key.pem
  cp $DOCKER_CERT_PATH/cert.pem ./docker-cert.pem
fi

# Deploy.
docker stack deploy -c ./sensor.yaml prevent --with-registry-auth

# Clean up temporary files.
rm registry-auth
if [ -n "$DOCKER_CERT_PATH" ]; then
  rm ./docker-*
fi

cd $WD
