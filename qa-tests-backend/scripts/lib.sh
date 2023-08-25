#!/usr/bin/env bash

# Common functions for deploying a cluster for QA tests

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
if [[ "$(type -t info)" != "function" ]]; then
    # shellcheck source=../../scripts/lib.sh
    source "$ROOT/scripts/lib.sh"
fi
if [[ "$(type -t is_in_PR_context)" != "function" ]]; then
    # shellcheck source=../../scripts/ci/lib.sh
    source "$ROOT/scripts/ci/lib.sh"
fi

set -euo pipefail

deploy_default_psp() {
    info "Deploy Default PSP for stackrox namespace"
    if [[ "$POD_SECURITY_POLICIES" != "false" ]]; then
        "${ROOT}/scripts/ci/create-default-psp.sh"
    else
        info "POD_SECURITY_POLICIES is false, skip Deploy Default PSP for stackrox namespace"
    fi
}

get_ECR_docker_pull_password() {
    info "Get AWS ECR Docker Pull Password"

    aws --version
    local pass
    pass="$(aws --region="${AWS_ECR_REGISTRY_REGION}" ecr get-login-password)"
    ci_export AWS_ECR_DOCKER_PULL_PASSWORD "${pass}"
}

deploy_clair_v4() {
    info "Deploy Clair v4"

    ci_export ROX_CLAIR_V4_SCANNING "${ROX_CLAIR_V4_SCANNING:-true}"
    ci_export CLAIR_V4_ENDPOINT "http://clairv4.qa-clairv4:8080"
    "$ROOT/scripts/ci/clairv4/deploy.sh" qa-clairv4
}

surface_spec_logs() {
    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        info "No place for artifacts, skipping spec logs summary"
        return
    fi

    if [[ ! -d "${ARTIFACT_DIR}/spec-logs" ]]; then
        info "No spec-logs stored"
        return
    fi

    artifact_file="$ARTIFACT_DIR/spec-logs/spec-logs-summary.html"

    cat > "$artifact_file" <<- HEAD
<html>
    <head>
        <title>Groovy Test Logs</title>
        <style>
          body { color: #e8e8e8; background-color: #424242; font-family: "Roboto", "Helvetica", "Arial", sans-serif }
          a { color: #ff8caa }
          a:visited { color: #ff8caa }
        </style>
    </head>
    <body>
    <ul>
HEAD

    local classname
    local url
    for log in "$ARTIFACT_DIR"/spec-logs/*.log; do
        classname="$(basename "${log}")"
        classname="${classname%.log}"
        url="$(get_spec_log_url "${log}")"
        cat >> "$artifact_file" << DETAILS
        <li><a target="_blank" href="${url}">${classname}</a></li>
DETAILS
    done

    cat >> "$artifact_file" <<- FOOT
    </ul>
    <br />
    <br />
  </body>
</html>
FOOT
}

get_spec_log_url() {
    local log="$1"

    # PR logs e.g. 
    # https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/pr-logs/pull/
    # stackrox_stackrox/6974/pull-ci-stackrox-stackrox-master-gke-qa-e2e-tests/1681361747606245376/artifacts/gke-qa-e2e-tests/
    # stackrox-e2e/artifacts/spec-logs/

    # Merge logs e.g.
    # https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/logs/
    # branch-ci-stackrox-stackrox-master-ocp-4-13-merge-qa-e2e-tests/1681375347259478016/artifacts/merge-qa-e2e-tests/
    # stackrox-e2e/artifacts/spec-logs/

    url="https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test"
    if is_in_PR_context; then
        url="${url}/pr-logs/pull/stackrox_stackrox/${PULL_NUMBER}"
    else
        url="${url}/logs"
    fi
    url="${url}/${JOB_NAME}/${BUILD_ID}/artifacts/${JOB_NAME_SAFE}/stackrox-e2e/artifacts"
    url="${url}/${log##"${ARTIFACT_DIR}"/}"

    echo "${url}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
