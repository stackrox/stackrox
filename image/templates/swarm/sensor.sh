#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

WD=$(pwd)
cd $DIR

# Create registry-auth secret, used to pull the benchmark image.
touch registry_auth
chmod 0600 registry_auth
./docker-auth.sh "https://index.docker.io/v1/" | cat >registry-auth

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
