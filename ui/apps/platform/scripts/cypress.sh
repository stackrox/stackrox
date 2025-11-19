#!/usr/bin/env bash

# Opens cypress with environment variables for feature flags and auth
api_endpoint="${UI_BASE_URL:-https://localhost:8000}"

if [[ -z "$ROX_USERNAME" || -z "$ROX_ADMIN_PASSWORD" ]]; then
  # basic auth creds weren't set (e.g. by CI), assume local k8s deployment
  # shellcheck source=../../../../scripts/k8s/export-basic-auth-creds.sh
  source ../../../scripts/k8s/export-basic-auth-creds.sh ../../../deploy/k8s
fi

curl_cfg() { # Use built-in echo to not expose $2 in the process list.
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

if [[ -n "$ROX_ADMIN_PASSWORD" ]]; then
  readarray -t arr < <(curl -sk --config <(curl_cfg user "admin:$ROX_ADMIN_PASSWORD") "${api_endpoint}"/v1/featureflags | jq -cr '.featureFlags[] | {name: .envVar, enabled: .enabled}')
  for i in "${arr[@]}"; do
    name=$(echo "$i" | jq -rc .name)
    val=$(echo "$i" | jq -rc .enabled)
    export CYPRESS_"${name}"="${val}"
  done
fi
export CYPRESS_ROX_AUTH_TOKEN=$(./scripts/get-auth-token.sh)

# eventually it should be in cypress.config.js: https://github.com/cypress-io/cypress/issues/5218
artifacts_dir="${TEST_RESULTS_OUTPUT_DIR:-cypress/test-results}/artifacts"
export CYPRESS_VIDEOS_FOLDER="${artifacts_dir}/videos"
export CYPRESS_SCREENSHOTS_FOLDER="${artifacts_dir}/screenshots"
if [[ -n "${UI_BASE_URL}" ]]; then
  export CYPRESS_BASE_URL="${UI_BASE_URL}"
fi

export CYPRESS_SPEC_PATTERN='cypress/integration/**/*.test.{js,ts}'

# be able to skip tests that are not relevant, for example: openshift
export CYPRESS_ORCHESTRATOR_FLAVOR="${ORCHESTRATOR_FLAVOR}"

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
        prefixed_specs+=("cypress/integration/$spec")
    done
    # Join with commas for Cypress
    spec_list=$(IFS=,; echo "${prefixed_specs[*]}")
    cypress run --spec "$spec_list"
else
    DEBUG="cypress*" NO_COLOR=1 cypress "$@" 2> /dev/null
fi
