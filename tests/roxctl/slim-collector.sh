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

curl_central() {
  url="$1"
  shift
  [[ -n "${url}" ]] || die "No URL specified"
  curl --retry 5 -Sskf -u "admin:${ROX_PASSWORD}" "https://${API_ENDPOINT}/${url}" "$@"
}

check_image() {
  image="$1"
  condition="$2"
  if [[ "$condition" == "has -slim" ]]; then
    [[ "$image" == *"-slim"* ]]
    return $?
  fi
  [[ "$image" != *"-slim"* ]]
  return $?
}

# Retrieve API token
API_TOKEN_JSON="$(curl_central v1/apitokens/generate \
  -d '{"name": "test", "role": "Admin"}')" \
  || die "Failed to retrieve Rox API token"
ROX_API_TOKEN="$(echo "$API_TOKEN_JSON" | jq -er .token)" \
  || die "Failed to retrieve token from JSON"
export ROX_API_TOKEN

test_collector_image_references_in_deployment_bundles() {
    SLIM_COLLECTOR_FLAG="$1"
    EXPECTED_IMAGE_CONDITION="$2"

    CLUSTER_NAME="test-cluster-${RANDOM}-${RANDOM}-${RANDOM}"
    echo "Testing correctness of collector image references for clusters generated with $SLIM_COLLECTOR_FLAG (cluster name is $CLUSTER_NAME)"

    # Verify that generating a cluster works.
    if OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor generate k8s --name "$CLUSTER_NAME" "$SLIM_COLLECTOR_FLAG" 2>&1)"; then
        echo "[OK] Generating cluster works"
    else
        eecho "[FAIL] Failed to generate cluster"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        printf "\n\n" >&2
        FAILURES=$((FAILURES + 1))
        return
    fi

    cluster_id="$(curl_central v1/clusters | jq --arg name "${CLUSTER_NAME}" '.clusters | .[] | select(.name==$name).id' -r)"
    if [[ -n "${cluster_id}" ]]; then
        echo "[OK] Got cluster id ${cluster_id}"
    else
        eecho "[FAIL] Failed to retrieve cluster id"
        FAILURES=$((FAILURES + 1))
        return
    fi

    # Verify that generated bundle references the expected collector image.
    COLLECTOR_IMAGE="$(egrep 'image: \S+/collector' "sensor-${CLUSTER_NAME}/collector.yaml" | sed -e 's/[^:]*: "\(.*\)"$/\1/;')"

    if check_image "$COLLECTOR_IMAGE" "$EXPECTED_IMAGE_CONDITION"; then
        echo "[OK] Newly generated bundle collector image satisfies condition: $EXPECTED_IMAGE_CONDITION"
    else
        eecho "[FAIL] Newly generated bundle collector image does not satisfy condition: $EXPECTED_IMAGE_CONDITION (collector image is $COLLECTOR_IMAGE)"
        FAILURES=$((FAILURES + 1))
        return
    fi

    rm -r "sensor-${CLUSTER_NAME}"

    # Verify that refetching deployment bundle for newly created cluster works as expected (i.e. that the bundle references the expected collector image).
    OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor get-bundle --output-dir="sensor-${CLUSTER_NAME}-refetched" "$CLUSTER_NAME" 2>&1)"
    COLLECTOR_IMAGE="$(egrep 'image: \S+/collector' "sensor-${CLUSTER_NAME}-refetched/collector.yaml" | sed -e 's/[^:]*: "\(.*\)"$/\1/;')"

    if check_image "$COLLECTOR_IMAGE" "$EXPECTED_IMAGE_CONDITION"; then
        echo "[OK] Refetched deployment collector image still satisfies condition: $EXPECTED_IMAGE_CONDITION"
    else
        eecho "[FAIL] Refetched deployment bundle collector image does not satisfy condition: $EXPECTED_IMAGE_CONDITION (collector image is $COLLECTOR_IMAGE)"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        FAILURES=$((FAILURES + 1))
        return
    fi

    rm -r "sensor-${CLUSTER_NAME}-refetched"

    curl_central "v1/clusters/${cluster_id}" -X DELETE
    if [[ $? -eq 0 ]]; then
        echo "[OK] Successfully cleaned up cluster"
    else
        eecho "[FAIL] Failed to delete cluster"
        FAILURES=$((FAILURES + 1))
        return
    fi
}

test_collector_image_references_in_deployment_bundles "--slim-collector" "has -slim"
test_collector_image_references_in_deployment_bundles "--slim-collector=auto" "has -slim" # Central is deployed in online mode in CI
test_collector_image_references_in_deployment_bundles "--slim-collector=false" "does not have -slim"

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES tests failed"
  exit 1
fi
