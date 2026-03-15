#!/usr/bin/env bash

# Only runs OpenShift Console plugin E2E tests when:
# - ORCHESTRATOR_FLAVOR is 'openshift'
# - OpenShift version is a supported version
#
# Version detection from CLUSTER_FLAVOR_VARIANT format: openshift-4-ocp/<prefix>-<major>.<minor>
# See scripts/ci/jobs/ocp_qa_e2e_tests.py
# Example: openshift-4-ocp/gcp-4.16 -> major=4, minor=16
#
# Edge cases default to RUNNING tests to avoid silently skipping them:
# - If version cannot be parsed, tests run
# - If CLUSTER_FLAVOR_VARIANT is empty, tests run
# Only skips if we definitively know the version is too old to prevent breakages in CI resulting in no test coverage.

set -eo pipefail

export CYPRESS_ORCHESTRATOR_FLAVOR="${ORCHESTRATOR_FLAVOR}"

if [[ "${ORCHESTRATOR_FLAVOR}" != "openshift" ]]; then
    echo "ORCHESTRATOR_FLAVOR is not 'openshift', skipping cypress-ocp tests"
    exit 0
fi

MIN_MAJOR=4
MIN_MINOR=19

should_run=true
major=""
minor=""

if [[ -z "${CLUSTER_FLAVOR_VARIANT:-}" ]]; then
    echo "CLUSTER_FLAVOR_VARIANT is not set, running cypress-ocp (no version check possible)"
elif [[ "${CLUSTER_FLAVOR_VARIANT}" =~ openshift-4-ocp/[^-]+-([0-9]+)\.([0-9]+) ]]; then
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    export CYPRESS_OSCI_OPENSHIFT_VERSION="${major}.${minor}"

    echo "Detected OpenShift version: ${CYPRESS_OSCI_OPENSHIFT_VERSION}"

    # Only skip if we know the version is too old
    if [[ "${major}" -lt ${MIN_MAJOR} ]] || { [[ "${major}" -eq ${MIN_MAJOR} ]] && [[ "${minor}" -lt ${MIN_MINOR} ]]; }; then
        should_run=false
    fi
else
    echo "CLUSTER_FLAVOR_VARIANT format not recognized: ${CLUSTER_FLAVOR_VARIANT}"
    echo "Running anyway (defaulting to run on parse failure)"
fi

if [[ "${should_run}" == "false" ]]; then
    echo "OpenShift ${CYPRESS_OSCI_OPENSHIFT_VERSION} < ${MIN_MAJOR}.${MIN_MINOR}, skipping cypress-ocp tests"
    exit 0
fi

# Opens cypress with environment variables for feature flags and auth
OPENSHIFT_CONSOLE_URL="${OPENSHIFT_CONSOLE_URL:-http://localhost:9000}"
API_PROXY_BASE_URL="${OPENSHIFT_CONSOLE_URL}/api/proxy/plugin/advanced-cluster-security/api-service/proxy/central"

if [[ -z "$OCP_BRIDGE_AUTH_DISABLED" && ( -z "$OPENSHIFT_CONSOLE_USERNAME" || -z "$OPENSHIFT_CONSOLE_PASSWORD" ) ]]; then
    echo "OPENSHIFT_CONSOLE_USERNAME and OPENSHIFT_CONSOLE_PASSWORD must be set if OCP_BRIDGE_AUTH_DISABLED is not true"
    exit 1
fi

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

if [[ -n "$OPENSHIFT_CONSOLE_PASSWORD" || "$OCP_BRIDGE_AUTH_DISABLED" == "true" ]]; then
  readarray -t arr < <(curl -sk --config <(curl_cfg user "$OPENSHIFT_CONSOLE_USERNAME:$OPENSHIFT_CONSOLE_PASSWORD") "${API_PROXY_BASE_URL}"/v1/featureflags | jq -cr '.featureFlags[] | {name: .envVar, enabled: .enabled}')
  for i in "${arr[@]}"; do
    name=$(echo "$i" | jq -rc .name)
    val=$(echo "$i" | jq -rc .enabled)
    export CYPRESS_"${name}"="${val}"
  done
fi

# eventually it should be in cypress.config.js: https://github.com/cypress-io/cypress/issues/5218
artifacts_dir="${TEST_RESULTS_OUTPUT_DIR:-cypress/test-results}/artifacts/ocp-console-plugin"
export CYPRESS_VIDEOS_FOLDER="${artifacts_dir}/videos"
export CYPRESS_SCREENSHOTS_FOLDER="${artifacts_dir}/screenshots"
if [[ -n "${OPENSHIFT_CONSOLE_URL}" ]]; then
  export CYPRESS_BASE_URL="${OPENSHIFT_CONSOLE_URL}"
fi

export CYPRESS_SPEC_PATTERN='cypress/integration-ocp/**/*.test.{js,ts}'


export CYPRESS_OCP_BRIDGE_AUTH_DISABLED="${OCP_BRIDGE_AUTH_DISABLED}"
export CYPRESS_OPENSHIFT_CONSOLE_USERNAME="${OPENSHIFT_CONSOLE_USERNAME}"
export CYPRESS_OPENSHIFT_CONSOLE_PASSWORD="${OPENSHIFT_CONSOLE_PASSWORD}"

if [ "$2" == "--spec" ]; then
    if [ $# -lt 3 ]; then
        echo "usage: npm run cypress-spec <spec-file> [<spec-file> ...]"
        exit 1
    fi
    # Collect all spec arguments (everything after --spec)
    shift 2  # Remove script name and --spec
    all_specs="$*"

    # Support multiple comma-separated spec files by prefixing each individually
    IFS=',' read -ra specs <<< "$all_specs"
    prefixed_specs=()
    for spec in "${specs[@]}"; do
        # Trim whitespace from each spec
        spec=$(echo "$spec" | xargs)
        prefixed_specs+=("cypress/integration-ocp/$spec")
    done
    # Join with commas for Cypress
    spec_list=$(IFS=,; echo "${prefixed_specs[*]}")
    cypress run --spec "$spec_list"
else
    DEBUG="cypress*" NO_COLOR=1 cypress "$@" 2> /dev/null
fi
