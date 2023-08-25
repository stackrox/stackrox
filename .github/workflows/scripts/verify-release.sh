#!/usr/bin/env bash
#
# Verifies that for a given release all artifacts
# have been published.
#
set -euo pipefail

check_args() {
    RELEASE_PATCH="$1"
    LATEST_VERSION="$2"
    RELEASE="$3"
    PROJECT="$4"
    ERRATA_NAME="$5"

    check_not_empty \
        RELEASE_PATCH \
        LATEST_VERSION \
        RELEASE \
        PROJECT \
        ERRATA_NAME
}

mark_failed() {
    touch failed_validation
}

trap_for_failed_validation() {
    if [ -f failed_validation ]; then
        exit 1
    fi
}

check_dir_not_empty() {
    DIR="$1"
    if [ -d "$DIR" ]; then
        if [ -n "$(find "$DIR" -maxdepth 0 -empty)" ]; then
            mark_failed
            gh_log error "The required directory ${DIR} is empty."
        fi
    else
        mark_failed
        gh_log error "The required directory ${DIR} does not exist."
    fi
}

check_url_page_exists() {
    URL="$1"
    curl -sSLf "$URL" --output /dev/null || {
        mark_failed
        gh_log error "Retrieving $URL failed."
    }
}

check_url_yaml_contains() {
    URL="$1"
    QUERY="$2"
    curl -sSLf "$URL" 2>/dev/null | yq -e "$QUERY" >/dev/null || {
        mark_failed
        gh_log error "The Helm index does not contain the new version."
    }
}

check_docker_image() {
    IMAGE="$1"

    DOCKER_CLI_EXPERIMENTAL=enabled docker manifest inspect "$IMAGE" >/dev/null || {
        mark_failed
        gh_log error "The required image $IMAGE does not exist."
    }
}

validate_helm_charts() {
    RELEASE_PATCH="$1"
    LATEST_VERSION="$2"
    git clone --quiet https://github.com/stackrox/helm-charts
    check_dir_not_empty "helm-charts/${RELEASE_PATCH}"
    if [ "${LATEST_VERSION}" == "true" ]; then
        if ! grep -q "${RELEASE_PATCH}" < "helm-charts/latest/central-services/Chart.yaml"; then
            mark_failed
            gh_log error "The symbolic link to the latest chart does not point to the ${RELEASE_PATCH} version."
        fi
    fi

    check_url_yaml_contains "https://charts.stackrox.io/index.yaml" ".entries.central-services[] | select( .appVersion == \"${RELEASE_PATCH}\")"
    check_url_yaml_contains "https://charts.stackrox.io/index.yaml" ".entries.secured-cluster-services[] | select( .appVersion == \"${RELEASE_PATCH}\")"
}

validate_images() {
    RELEASE_PATCH="$1"
    check_docker_image "registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8:${RELEASE_PATCH}"
    check_docker_image "quay.io/stackrox-io/main:${RELEASE_PATCH}"
    check_docker_image "quay.io/rhacs-eng/main:${RELEASE_PATCH}"
}

validate_docs() {
    RELEASE="$1"
    check_url_page_exists "https://docs.openshift.com/acs/${RELEASE}/welcome/index.html"
}

validate_jira_release() {
    PROJECT="$1"
    RELEASE_PATCH="$2"

    IS_RELEASED=$(curl -sSLf \
        -H "Authorization: Bearer $JIRA_TOKEN" \
        "https://issues.redhat.com/rest/api/2/project/${PROJECT}/versions" \
    | jq -r ".[] | select(.name == \"${RELEASE_PATCH}\") | .released")
    if [ "${IS_RELEASED}" != "true" ]; then
        mark_failed
        gh_log error "JIRA Release $RELEASE_PATCH has not been marked as done."
    fi
}

validate_errata() {
    ERRATA_NAME="$1"
    check_url_page_exists "https://access.redhat.com/errata/${ERRATA_NAME}"
}

main() {
    RELEASE_PATCH="$1"
    LATEST_VERSION="$2"
    RELEASE="$3"
    PROJECT="$4"
    ERRATA_NAME="$5"

    TMP_DIR="$(mktemp -d)"
    pushd "$TMP_DIR" >/dev/null

    validate_helm_charts "$RELEASE_PATCH" "$LATEST_VERSION"
    validate_images "$RELEASE_PATCH"
    validate_docs "$RELEASE"
    validate_jira_release "$PROJECT" "$RELEASE_PATCH"
    validate_errata "$ERRATA_NAME"

    trap_for_failed_validation
    popd >/dev/null 2>&1
    rm -rf "$TMP_DIR"
}

check_args "$@"
main "$@"
