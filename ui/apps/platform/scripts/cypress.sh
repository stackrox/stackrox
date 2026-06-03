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
roles_json="./cypress/constants/cypressTestRoles.json"
token_prefix=$(jq -r '.tokenNamePrefix' "$roles_json")
default_role=$(jq -r '.defaultRole' "$roles_json")
readarray -t roles < <(jq -r '.roles[]' "$roles_json")

cleanup_tokens() {
  echo "Cleaning up cypress test tokens..."
  ./scripts/cleanup-cypress-tokens.sh 2>&1 || echo "Warning: token cleanup failed"
}

# Clean up tokens before test run, and then again once the script exits
cleanup_tokens
trap cleanup_tokens EXIT

for role in "${roles[@]}"; do
  token_name="${token_prefix}_${role}"
  env_key="ROX_AUTH_TOKEN_$(echo "$role" | tr '[:lower:] ' '[:upper:]_')"

  token_stderr=$(mktemp)
  token=$(UI_API_TOKEN_NAME="$token_name" UI_API_TOKEN_ROLE="$role" ./scripts/get-auth-token.sh 2>"$token_stderr")

  if [[ -n "$token" ]]; then
    export "CYPRESS_${env_key}=${token}"
    echo "Created token for role: $role"

    if [[ "$role" == "$default_role" ]]; then
      export CYPRESS_ROX_AUTH_TOKEN="$token"
    fi
  else
    echo >&2 "ERROR: Failed to create token for role: $role (continuing...)"
    cat >&2 "$token_stderr"
  fi
  rm -f "$token_stderr"
done

if [[ -z "${CYPRESS_ROX_AUTH_TOKEN:-}" ]]; then
  echo >&2 "FATAL: Could not create token for default role ($default_role)"
  exit 1
fi

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
