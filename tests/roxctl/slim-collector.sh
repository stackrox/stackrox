#! /usr/bin/env bash

set -uo pipefail

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

[ -n "$API_ENDPOINT" ]
[ -n "$ROX_PASSWORD" ]

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

eecho() {
  echo "$@" >&2
}

die() {
    eecho "$@"
    exit 1
}

# Retrieve API token
API_TOKEN_JSON="$(curl -Sskf \
  -u "admin:$ROX_PASSWORD" \
  -d '{"name": "test", "role": "Admin"}' \
  "https://$API_ENDPOINT/v1/apitokens/generate")" \
  || die "Failed to retrieve Rox API token"
ROX_API_TOKEN="$(echo "$API_TOKEN_JSON" | jq -er .token)" \
  || die "Failed to retrieve token from JSON"
export ROX_API_TOKEN

test_collector_image_references_in_deployment_bundles() {
    SLIM_COLLECTOR_FLAG="$1"
    EXPECTED_IMAGE_TAG="$2"

    CLUSTER_NAME="test-cluster-$RANDOM"
    echo "Testing correctness of collector image references for clusters generated with $SLIM_COLLECTOR_FLAG (cluster name is $CLUSTER_NAME)"

    # Verify that generating a cluster works.
    if OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor generate k8s --name "$CLUSTER_NAME" "$SLIM_COLLECTOR_FLAG" 2>&1)"; then
        echo "[OK] Generating cluster works"
    else
        eecho "[FAIL] Failed to generate cluster"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        FAILURES=$((FAILURES + 1))
    fi

    # Verify that generated bundle references the expected collector image.
    COLLECTOR_IMAGE_TAG="$(grep "image: docker.io/stackrox/collector" "sensor-${CLUSTER_NAME}/sensor.yaml" | sed -e 's/[^:]*: [^:]*:\(.*\)$/\1/;')"
    COLLECTOR_IMAGE_TAG_SUFFIX="$(echo "$COLLECTOR_IMAGE_TAG" | sed -e 's/.*-\([^-]*\)$/\1/;')"

    if [ "$COLLECTOR_IMAGE_TAG_SUFFIX" == "$EXPECTED_IMAGE_TAG" ]; then
        echo "[OK] Newly generated bundle references $EXPECTED_IMAGE_TAG collector image"
    else
        eecho "[FAIL] Newly generated bundle does not reference $EXPECTED_IMAGE_TAG collector image (referenced collector image tag is $COLLECTOR_IMAGE_TAG)"
        FAILURES=$((FAILURES + 1))
    fi

    # Verify that refetching deployment bundle for newly created cluster works as expected (i.e. that the bundle references the expected collector image).
    OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor get-bundle --output-dir="sensor-${CLUSTER_NAME}-refetched" "$CLUSTER_NAME" 2>&1)"
    COLLECTOR_IMAGE_TAG="$(grep "image: docker.io/stackrox/collector" "sensor-${CLUSTER_NAME}-refetched/sensor.yaml" | sed -e 's/[^:]*: [^:]*:\(.*\)$/\1/;')"
    COLLECTOR_IMAGE_TAG_SUFFIX="$(echo "$COLLECTOR_IMAGE_TAG" | sed -e 's/.*-\([^-]*\)$/\1/;')"

    if [ "$COLLECTOR_IMAGE_TAG_SUFFIX" == "$EXPECTED_IMAGE_TAG" ]; then
        echo "[OK] Refetched deployment bundle still references $EXPECTED_IMAGE_TAG collector image"
    else
        eecho "[FAIL] Refetched deployment bundle does not reference $EXPECTED_IMAGE_TAG collector image (referenced collector image tag is $COLLECTOR_IMAGE_TAG)"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        FAILURES=$((FAILURES + 1))
    fi
}

test_collector_image_references_in_deployment_bundles "--slim-collector" "slim"
test_collector_image_references_in_deployment_bundles "--slim-collector=auto" "slim" # Central is deployed in online mode in CI
test_collector_image_references_in_deployment_bundles "--slim-collector=false" "latest"


if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES test failed"
  exit 1
fi
