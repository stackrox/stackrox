#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/install-agent-quadlet.sh"

    NAMESPACE="openshift-cnv"
    SSH_USER="cloud-user"
    IMAGE_TAG="4.12.x-123-gdeadbeef"
    AUTOMATION_SSH_PRIVKEY="${BATS_TEST_TMPDIR}/id_ed25519"
    touch "${AUTOMATION_SSH_PRIVKEY}"

    VIRTCTL_INSTALL_COMPLETE="true"
    VIRTCTL_INSTALLED_IMAGE="Image=quay.io/stackrox-io/main:${IMAGE_TAG}"

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

        if [[ "$cmd" == *"systemctl is-enabled roxagent.timer"* ]]; then
            [[ "${VIRTCTL_INSTALL_COMPLETE}" == "true" ]]
            return
        fi

        if [[ "$cmd" == *"grep -h '^Image='"* ]]; then
            printf '%s\n' "${VIRTCTL_INSTALLED_IMAGE}"
            return
        fi

        echo "unexpected virtctl command: ${cmd}" >&2
        return 1
    }
}

@test "installed_image_tag_matches rejects partial installs even when image matches" {
    VIRTCTL_INSTALL_COMPLETE="false"

    run installed_image_tag_matches "rhel10-1"

    assert_failure
}

@test "installed_image_tag_matches rejects mismatched image tags" {
    VIRTCTL_INSTALLED_IMAGE="Image=quay.io/stackrox-io/main:wrong-tag"

    run installed_image_tag_matches "rhel10-1"

    assert_failure
}

@test "installed_image_tag_matches accepts complete installs with matching image tags" {
    run installed_image_tag_matches "rhel10-1"

    assert_success
}
