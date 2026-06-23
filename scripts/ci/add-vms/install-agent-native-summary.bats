#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/install-agent-native.sh"

    NAMESPACE="openshift-cnv"
    SSH_USER="cloud-user"
    AUTOMATION_SSH_PRIVKEY="${BATS_TEST_TMPDIR}/id_ed25519"
    touch "${AUTOMATION_SSH_PRIVKEY}"
}

# mock_virtctl_serve_status stubs virtctl to return the given
# serve_enabled and serve_active values when the command contains
# "systemctl is-enabled roxagent-serve.service".
mock_virtctl_serve_status() {
    local serve_enabled="$1" serve_active="$2"
    export _MOCK_SERVE_ENABLED="$serve_enabled"
    export _MOCK_SERVE_ACTIVE="$serve_active"

    virtctl() {
        local cmd=""
        while (($#)); do
            case "$1" in
                --command) cmd="$2"; shift 2 ;;
                *)         shift ;;
            esac
        done

        if [[ "$cmd" == *"roxagent-serve.service"* ]]; then
            printf '%s\n%s\n' "$_MOCK_SERVE_ENABLED" "$_MOCK_SERVE_ACTIVE"
            return 0
        fi
        return 1
    }
}

@test "native_agent_service_verified tracks successful starts" {
    mock_virtctl_serve_status "enabled" "active"

    run native_agent_service_verified "rhel10-1"

    assert_success
}

@test "native_agent_service_verified rejects unhealthy service state" {
    mock_virtctl_serve_status "enabled" "inactive"

    run native_agent_service_verified "rhel10-1"

    assert_failure
}
