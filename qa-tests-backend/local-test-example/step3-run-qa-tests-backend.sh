#!/bin/bash
set -eu
source "local-test-example/common.sh"
source "local-test-example/config.sh"
cd "$STACKROX_SOURCE_ROOT"  # all paths should be relative to here

SCRIPT_ROOT=$(realpath "$(dirname "$0")")  # brew install coreutils
echo "SCRIPT_ROOT          : $SCRIPT_ROOT"
echo "QA_TESTS_BACKEND_DIR : $QA_TESTS_BACKEND_DIR"

echo "Creating $QA_TESTS_BACKEND_DIR/qa-test-settings.properties"
pass show qa-test-settings.properties \
    > "$QA_TESTS_BACKEND_DIR/qa-test-settings.properties"

cd "$QA_TESTS_BACKEND_DIR"
REGISTRY_USERNAME="$(pass quay-io-ro-username)"; export REGISTRY_USERNAME
REGISTRY_PASSWORD="$(pass quay-io-ro-password)"; export REGISTRY_PASSWORD

# Disabling build to accelerate dev loop -- takes 3-5 minutes on my laptop
if false; then
    go mod tidy
    make style proto-generated-srcs
else
    echo "SKIPPING BUILD TO SPEEDUP DEV LOOP"
fi

export CLUSTER="OPENSHIFT"
export AWS_ECR_REGISTRY_NAME="051999192406"
export AWS_ECR_REGISTRY_REGION="us-east-2"

AWS_ECR_DOCKER_PULL_PASSWORD="$(aws ecr get-login-password)" || true
export AWS_ECR_DOCKER_PULL_PASSWORD

QUAY_USERNAME="$(pass quay-io-ro-username)"
QUAY_PASSWORD="$(pass quay-io-ro-password)"
export QUAY_USERNAME QUAY_PASSWORD

# Required vars for Groovy e2e api tests
export API_HOSTNAME="localhost"
export API_PORT="8443"

export KUBECONFIG="/tmp/kubeconfig"
pkill -f 'port-forward.*svc/central' || true
sleep 2
kubectl port-forward -n stackrox svc/central "$API_PORT:443" &> /tmp/central.log &
sleep 3

# Verify API connectivity
nc -vz "$API_HOSTNAME" "$API_PORT" \
    || error "FAILED: [nc -vz $API_HOSTNAME $API_PORT]"

PASSWORD_FILE_PATH="$GOPATH/src/github.com/stackrox/stackrox/deploy/openshift/central-deploy/password"
ROX_USERNAME="admin"
ROX_PASSWORD=$(cat "$PASSWORD_FILE_PATH")
export ROX_USERNAME ROX_PASSWORD

echo "Access Central console at http://$API_HOSTNAME:$API_PORT"
echo "Login with ($ROX_USERNAME, $ROX_PASSWORD)"

gradle build -x test
#gradle test --tests='LocalQaPropsTest'
gradle test --tests='ReconciliationTest'
