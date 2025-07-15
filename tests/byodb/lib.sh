#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test utility functions for upgrades
deploy_external_postgres() {
    pwd
    EXTERNAL_DB_PASSWORD="$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c12 || true)"
    EXTERNAL_DB_USER=stackrox
    EXTERNAL_DATABASE_NAME=stackrox
    EXTERNAL_DATABASE_HOST=postgres.database
    ci_export "EXTERNAL_DB_PASSWORD" "$EXTERNAL_DB_PASSWORD"
    ci_export "EXTERNAL_DB_USER" "$EXTERNAL_DB_USER"
    ci_export "EXTERNAL_DATABASE_NAME" "$EXTERNAL_DATABASE_NAME"
    ci_export "EXTERNAL_DATABASE_HOST" "$EXTERNAL_DATABASE_HOST"
    ci_export "EXTERNAL_DB" true

    kubectl create namespace database
    envsubst < ./tests/byodb/simple-postgres.yaml | kubectl apply -f -

    kubectl wait --for=condition=Ready pod -l app=postgres -n database --timeout=180s
}

preamble() {
    info "Starting test preamble"

    if is_darwin; then
        HOST_OS="darwin"
    elif is_linux; then
        HOST_OS="linux"
    else
        die "Only linux or darwin are supported for this test"
    fi

    case "$(uname -m)" in
        x86_64) TEST_HOST_PLATFORM="${HOST_OS}_amd64" ;;
        aarch64) TEST_HOST_PLATFORM="${HOST_OS}_arm64" ;;
        arm64) TEST_HOST_PLATFORM="${HOST_OS}_arm64" ;;
        ppc64le) TEST_HOST_PLATFORM="${HOST_OS}_ppc64le" ;;
        s390x) TEST_HOST_PLATFORM="${HOST_OS}_s390x" ;;
        *) die "Unknown architecture" ;;
    esac

    require_executable "$TEST_ROOT/bin/${TEST_HOST_PLATFORM}/roxctl"

    if is_CI; then
        if ! command -v yq >/dev/null 2>&1; then
            sudo wget https://github.com/mikefarah/yq/releases/download/v4.4.1/yq_linux_amd64 -O /usr/bin/yq
            sudo chmod 0755 /usr/bin/yq
        fi
    else
        require_executable yq
    fi
}
