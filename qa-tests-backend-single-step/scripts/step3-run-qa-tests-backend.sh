#!/bin/bash
# Run E2E tests (Groovy + Spock + Fabric8 + Gradle)
set -eux
source "scripts/common.sh"
source "scripts/config.sh"

SCRIPT_ROOT=$(realpath "$(dirname "$0")")  # brew install coreutils
QA_TESTS_BACKEND_DIR="$GOPATH/src/github.com/stackrox/stackrox/qa-tests-backend"
echo "SCRIPT_ROOT          : $SCRIPT_ROOT"
echo "QA_TESTS_BACKEND_DIR : $QA_TESTS_BACKEND_DIR"

echo "Creating $QA_TESTS_BACKEND_DIR/qa-test-settings.properties"
pass show qa-test-settings.properties.v2 \
    > "$QA_TESTS_BACKEND_DIR/qa-test-settings.properties"

cd "$QA_TESTS_BACKEND_DIR"
REGISTRY_USERNAME="$(pass quay-io-ro-username)"; export REGISTRY_USERNAME
REGISTRY_PASSWORD="$(pass quay-io-ro-password)"; export REGISTRY_PASSWORD

# Disabling build to accelerate dev loop -- takes 3-5 minutes on my laptop
if false; then
  make style proto-generated-srcs
else
  echo "SKIPPING BUILD TO SPEEDUP DEV LOOP"
fi

export CLUSTER="OPENSHIFT"
export AWS_ECR_REGISTRY_NAME="051999192406"
export AWS_ECR_REGISTRY_REGION="us-east-2"

AWS_ECR_DOCKER_PULL_PASSWORD="$(aws ecr get-login-password)" || true
export AWS_ECR_DOCKER_PULL_PASSWORD

CENTRAL_USERNAME="admin"
CENTRAL_PASSWORD=$(cat "$GOPATH/src/github.com/stackrox/stackrox/deploy/openshift/central-deploy/password")
export ROX_PASSWORD="$CENTRAL_PASSWORD"

QUAY_USERNAME="$(pass quay-io-ro-username)"
QUAY_PASSWORD="$(pass quay-io-ro-password)"
export QUAY_USERNAME QUAY_PASSWORD

port-forward-central
PASSWORD_FILE_PATH=$(join_by ""  \
  "$GOPATH/src/github.com/stackrox/stackrox/" \
  "deploy/openshift/central-deploy/password")
CENTRAL_USERNAME="admin"
CENTRAL_PASSWORD=$(cat "$PASSWORD_FILE_PATH")
echo "Access Central console at localhost:8443"
echo "Login with ($CENTRAL_USERNAME, $CENTRAL_PASSWORD)"

gradle test --tests='ImageScanningTest'
#gradle test --tests='ImageScanningTest.Image metadata from registry test'
