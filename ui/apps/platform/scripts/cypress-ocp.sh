#!/usr/bin/env bash

env | grep -o '^CLUSTER.*'
set -x

# Opens cypress with environment variables for feature flags and auth
OPENSHIFT_API_ENDPOINT="${OPENSHIFT_API_ENDPOINT:-http://localhost:9000}"
API_PROXY_BASE_URL="${OPENSHIFT_API_ENDPOINT}/api/proxy/plugin/advanced-cluster-security/api-service"

if [[ -z "$OPENSHIFT_CONSOLE_USERNAME" || -z "$OPENSHIFT_CONSOLE_PASSWORD" ]]; then
    echo "OPENSHIFT_CONSOLE_USERNAME and OPENSHIFT_CONSOLE_PASSWORD must be set"
    exit 1
fi

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

if [[ -n "$OPENSHIFT_CONSOLE_PASSWORD" ]]; then
  readarray -t arr < <(curl -sk --config <(curl_cfg user "$OPENSHIFT_CONSOLE_USERNAME:$OPENSHIFT_CONSOLE_PASSWORD") "${API_PROXY_BASE_URL}"/v1/featureflags | jq -cr '.featureFlags[] | {name: .envVar, enabled: .enabled}')
  for i in "${arr[@]}"; do
    name=$(echo "$i" | jq -rc .name)
    val=$(echo "$i" | jq -rc .enabled)
    export CYPRESS_"${name}"="${val}"
  done
fi

# eventually it should be in cypress.config.js: https://github.com/cypress-io/cypress/issues/5218
artifacts_dir="${TEST_RESULTS_OUTPUT_DIR:-cypress/test-results}/ocp-artifacts"
export CYPRESS_VIDEOS_FOLDER="${artifacts_dir}/videos"
export CYPRESS_SCREENSHOTS_FOLDER="${artifacts_dir}/screenshots"
if [[ -n "${OPENSHIFT_CONSOLE_URL}" ]]; then
  export CYPRESS_BASE_URL="${OPENSHIFT_CONSOLE_URL}"
fi

export CYPRESS_SPEC_PATTERN='cypress/integration-ocp/**/*.test.{js,ts}'

# be able to skip tests that are not relevant, for example: openshift
export CYPRESS_ORCHESTRATOR_FLAVOR="${ORCHESTRATOR_FLAVOR}"

export CYPRESS_OCP_BRIDGE_AUTH_DISABLED="${OCP_BRIDGE_AUTH_DISABLED}"
export CYPRESS_CLUSTER_USERNAME="${OPENSHIFT_CONSOLE_USERNAME}"
export CYPRESS_CLUSTER_PASSWORD="${OPENSHIFT_CONSOLE_PASSWORD}"

# exit if ORCHESTRATOR_FLAVOR is not 'openshift'
if [ "${ORCHESTRATOR_FLAVOR}" != "openshift" ]; then
    echo "ORCHESTRATOR_FLAVOR is not 'openshift', skipping cypress-ocp"
    exit 0
fi

if [ "$2" == "--spec" ]; then
    if [ $# -ne 3 ]; then
        echo "usage: npm run cypress-spec <spec-file>"
        exit 1
    fi
    cypress run --spec "cypress/integration-ocp/$3"
else
    DEBUG="cypress*" NO_COLOR=1 cypress "$@" 2> /dev/null
fi
