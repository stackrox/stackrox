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

@test "native_agent_service_verified tracks successful starts" {
    virtctl() {
        local cmd=""
        while (($#)); do
            case "$1" in
                --command)
                    cmd="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        if [[ "$cmd" == *"systemctl show roxagent.service"* ]]; then
            printf 'success\nenabled\nactive\n'
            return 0
        fi

        return 1
    }

    run native_agent_service_verified "rhel10-1"

    assert_success
}

@test "native_agent_service_verified rejects unhealthy service state" {
    virtctl() {
        local cmd=""
        while (($#)); do
            case "$1" in
                --command)
                    cmd="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done

        if [[ "$cmd" == *"systemctl show roxagent.service"* ]]; then
            printf 'failed\nenabled\ninactive\n'
            return 0
        fi

        return 1
    }

    run native_agent_service_verified "rhel10-1"

    assert_failure
}
