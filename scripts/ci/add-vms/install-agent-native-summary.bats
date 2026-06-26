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

# mock_virtctl_service_status stubs virtctl to return the given
# service_result, timer_enabled, and timer_active values when the
# command contains "systemctl show roxagent.service".
mock_virtctl_service_status() {
    local service_result="$1" timer_enabled="$2" timer_active="$3"
    # Export so the subshell created by `run` can see them.
    export _MOCK_SERVICE_RESULT="$service_result"
    export _MOCK_TIMER_ENABLED="$timer_enabled"
    export _MOCK_TIMER_ACTIVE="$timer_active"

    virtctl() {
        local cmd=""
        while (($#)); do
            case "$1" in
                --command) cmd="$2"; shift 2 ;;
                *)         shift ;;
            esac
        done

        if [[ "$cmd" == *"systemctl show roxagent.service"* ]]; then
            printf '%s\n%s\n%s\n' "$_MOCK_SERVICE_RESULT" "$_MOCK_TIMER_ENABLED" "$_MOCK_TIMER_ACTIVE"
            return 0
        fi
        return 1
    }
}

@test "native_agent_service_verified tracks successful starts" {
    mock_virtctl_service_status "success" "enabled" "active"

    run native_agent_service_verified "rhel10-1"

    assert_success
}

@test "native_agent_service_verified rejects unhealthy service state" {
    mock_virtctl_service_status "failed" "enabled" "inactive"

    run native_agent_service_verified "rhel10-1"

    assert_failure
}
