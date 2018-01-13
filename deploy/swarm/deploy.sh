#!/usr/bin/env bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [ ! -f "./ca-key.pem" ]; then
    echo "Generating CA key..."
    echo " + Getting cfssl..."
    go get -u github.com/cloudflare/cfssl/cmd/...
    echo " + Generating keypair..."
    echo '{"CN":"CA","key":{"algo":"ecdsa"}}' | cfssl gencert -initca - | cfssljson -bare ca -
fi

FLAGS=""
if [ "$REGISTRY_AUTH" = "true" ]; then
    FLAGS += "--with-registry-auth "
fi

APOLLO_ENDPOINT="${APOLLO_ENDPOINT:-localhost:8080}"
echo "Apollo endpoint set to $APOLLO_ENDPOINT"

APOLLO_IMAGE_TAG="${APOLLO_IMAGE_TAG:-latest}"
echo "Apollo image tag set to $APOLLO_IMAGE_TAG"

echo "Deploying Central..."
docker stack deploy -c $DIR/central.yaml apollo $FLAGS

echo -n "Waiting for Central to respond."
set +e
until $(curl --output /dev/null --silent --fail -k https://$APOLLO_ENDPOINT/v1/ping); do
    echo -n '.'
    sleep 1
done
set -e
echo ""

echo "Creating a new cluster"
CLUSTER_NAME=remote
export CLUSTER="{\"name\": \"$CLUSTER_NAME\", \"type\": \"SWARM_CLUSTER\", \"apollo_image\": \"stackrox/apollo:$APOLLO_IMAGE_TAG\", \"central_api_endpoint\": \"$APOLLO_ENDPOINT\"}"
RESP=$(curl -X POST \
    -d "$CLUSTER" \
    -k \
    -s \
    https://$APOLLO_ENDPOINT/v1/clusters)
echo "Response: $RESP"
echo "$RESP" | jq -r .deploymentYaml > agent-$CLUSTER_NAME-deploy.yaml

echo "Getting identity for new cluster"
RESP=$(curl -X POST \
    -d '{"name": "remote", "type": "SENSOR_SERVICE"}' \
    -k \
    -s \
    https://$APOLLO_ENDPOINT/v1/serviceIdentities)
echo "Response: $RESP"
echo "$RESP" | jq -r .certificate > agent-$CLUSTER_NAME-cert.pem
echo "$RESP" | jq -r .privateKey > agent-$CLUSTER_NAME-key.pem

echo "Getting CA certificate"
RESP=$(curl \
    -k \
    -s \
    https://$APOLLO_ENDPOINT/v1/authorities)
echo "Response: $RESP"
echo "$RESP" | jq -r .authorities[0].certificate > central-ca.pem

docker stack deploy -c agent-$CLUSTER_NAME-deploy.yaml apollo $FLAGS
