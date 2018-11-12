#!/usr/bin/env bash
set -eu

function create_cluster {
    LOCAL_API_ENDPOINT="localhost:8000"
    CLUSTER_NAME="$1"
    CLUSTER_TYPE=KUBERNETES_CLUSTER
    CLUSTER_IMAGE="$2"
    CLUSTER_API_ENDPOINT="central.stackrox:443"
    RUNTIME_SUPPORT="true"

    echo "Creating a new cluster"

    export CLUSTER_JSON="{\"name\": \"$CLUSTER_NAME\", \"type\": \"$CLUSTER_TYPE\", \"main_image\": \"$CLUSTER_IMAGE\", \"central_api_endpoint\": \"$CLUSTER_API_ENDPOINT\", \"runtime_support\": $RUNTIME_SUPPORT}"

    TMP=$(mktemp)
    STATUS=$(curl -X POST \
        -d "$CLUSTER_JSON" \
        -k \
        -s \
        -o $TMP \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/clusters)
    >&2 echo "Status: $STATUS"
    if [ "$STATUS" == "500" ]; then
      cat $TMP
      exit 1
    fi

    clusterID="$(cat ${TMP} | jq -r .cluster.id)"
}

function create_identity {
    LOCAL_API_ENDPOINT="localhost:8000"
    CLUSTER_ID=$1
    OUTPUT_DIR="."

    echo "Creating identity for new cluster"
    export ID_JSON="{\"id\": \"$CLUSTER_ID\", \"type\": \"SENSOR_SERVICE\"}"
    TMP=$(mktemp)
    STATUS=$(curl -X POST \
        -d "$ID_JSON" \
        -k \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/serviceIdentities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .certificatePem | base64 -D > "$OUTPUT_DIR/sensor-cert.pem"
    cat "$TMP" | jq -r .privateKeyPem  | base64 -D > "$OUTPUT_DIR/sensor-key.pem"
    rm "$TMP"
    echo
}

# get_authority
# arguments:
#   - central API server endpoint reachable from this host
#   - directory to drop files in
function get_authority {
    LOCAL_API_ENDPOINT="localhost:8000"
    OUTPUT_DIR="."

    echo "Getting CA certificate"
    TMP="$(mktemp)"
    STATUS=$(curl \
        -k \
        -s \
        -o "$TMP" \
        -w "%{http_code}\n" \
        https://$LOCAL_API_ENDPOINT/v1/authorities)
    echo "Status: $STATUS"
    echo "Response: $(cat ${TMP})"
    cat "$TMP" | jq -r .authorities[0].certificatePem | base64 -D > "$OUTPUT_DIR/ca.pem"
    rm "$TMP"
    echo
}

# This launches mock_sensor with the tag defined by `make tag`.
# Any arguments passed to this script are passed on to the mocksensor program.
# Example: ./launch_mock_sensor.sh -max-deployments 100 will launch mocksensor with the args -max-deployments 100.

tag=$(git describe --tags --abbr)
echo "Launching mock sensor with tag: ${tag}"

clusterName=""
declare -a params

for i in "$@"
do
case $i in
    -deployment-name=*)
    sed -i.bak 's@name: DEPLOYMENT_NAME@name: '"${i#*=}"'@' mocksensor.yaml
    ;;
    -secret-name=*)
    secretName="${i#*=}"
    sed -i.bak 's@secretName: SECRET_NAME@secretName: '"${i#*=}"'@' mocksensor.yaml
    ;;
    -cluster-name=*)
    clusterName="${i#*=}"
    ;;
    *)
    sed -i.bak 's@args:@args:\
           - "'"${i}"'"@' mocksensor.yaml
    ;;
esac
done

sed -i.bak 's@image: .*@image: stackrox/scale:'"${tag}"'@' mocksensor.yaml

create_cluster "${clusterName}" "${tag}"
create_identity "${clusterID}"
get_authority

kubectl create secret -n "stackrox" generic $secretName --from-file="./sensor-cert.pem" --from-file="./sensor-key.pem" --from-file="./ca.pem"

kubectl -n stackrox delete deploy/sensor || true
sleep 5
kubectl create -f mocksensor.yaml
git checkout -- mocksensor.yaml