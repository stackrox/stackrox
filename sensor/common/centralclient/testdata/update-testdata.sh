#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
GIT_ROOT=$(git rev-parse --show-toplevel)

# This script will update the testdata for the TLSChallenge tests.
# Usage: ./update-testdata.sh
#
# Prerequisites:
# - jq
# - openssl

if [[ "$UPDATE_CENTRAL_CA" == "true" ]]; then
    if ! kubectl -n stackrox get secrets central-tls; then
        echo "Central CA not found. Running StackRox instance with provisioned Central CA cert required."
        exit 1
    fi
    # Receive new StackRox CA certificate from currently running instance. Safe it as testdata to be used in the test case.
    centralTLS=$(kubectl -n stackrox get secret central-tls -o json)
    files=("ca.pem" "ca-key.pem" "jwt-key.pem" "cert.pem" "key.pem")
    for key in "${files[@]}"
    do
        echo "$centralTLS" | jq -r ".data[\"$key\"]" | base64 --decode > "$SCRIPT_DIR/central/central-$key"
    done
fi

# Generate an new private key and CA certificate which is used as an additional CA in Central.
openssl genrsa -out "$SCRIPT_DIR"/additionalCA.key 2048
openssl req -x509 -new -nodes -key "$SCRIPT_DIR"/additionalCA.key -sha256 -out "$SCRIPT_DIR"/additionalCAs/ca.pem -days 100000 -subj '/CN=Root LoadBalancer Certificate Authority'

# The token is a random cryptographically generated number. Generation is done in sensor/common/centralclient/client.go:generateChallengeToken
challenge_token="h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
trustInfoResponse=$(go run "$GIT_ROOT/central/metadata/service/testdata/exec_tlschallenge.go" "$challenge_token" "$SCRIPT_DIR/central" "$SCRIPT_DIR/additionalCAs")

# Update signature and trustInfoSerialized example file
echo "$trustInfoResponse" | jq ".trustInfoSerialized" -r > "$SCRIPT_DIR/trust_info_serialized.example"
echo "$trustInfoResponse" | jq .signature -r > "$SCRIPT_DIR/signature.example"

echo "Run go unit tests..."
go test "$SCRIPT_DIR/../"
