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

        if [[ "$cmd" == *"systemctl is-active roxagent.service"* ]]; then
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

stub_central_image() {
    KUBECTL_CENTRAL_IMAGE="$1"
    kubectl() {
        [[ "$*" == *'deploy/central'* ]] || {
            echo "unexpected kubectl: $*" >&2
            return 1
        }
        [[ "$*" == *'name=="central"'* ]] || {
            echo "jsonpath did not select the central container: $*" >&2
            return 1
        }
        printf '%s\n' "${KUBECTL_CENTRAL_IMAGE}"
    }
}

@test "detect_image_tag keeps a provided tag without calling kubectl" {
    kubectl() {
        echo "kubectl should not be called" >&2
        return 1
    }

    run detect_image_tag

    assert_success
    assert_output --partial "Using provided image tag: 4.12.x-123-gdeadbeef"
}

@test "detect_image_tag rejects a provided value that is not a Docker tag" {
    IMAGE_TAG='foo&bar'

    run detect_image_tag

    assert_failure
    assert_output --partial "Invalid image tag"
}

@test "detect_image_tag reads a tag from a Central image reference" {
    IMAGE_TAG=""
    stub_central_image "quay.io/stackrox-io/main:4.8.0"

    run detect_image_tag

    assert_success
    assert_output --partial "Detected image tag: 4.8.0"
}

@test "detect_image_tag keeps the tag when the Central image is digest-pinned" {
    IMAGE_TAG=""
    stub_central_image "quay.io/stackrox-io/main:4.8.0@sha256:deadbeef"

    run detect_image_tag

    assert_success
    assert_output --partial "Detected image tag: 4.8.0"
}

@test "detect_image_tag reads a tag from a host:port reference" {
    IMAGE_TAG=""
    stub_central_image "localhost:5000/stackrox/main:abc"

    run detect_image_tag

    assert_success
    assert_output --partial "Detected image tag: abc"
}

@test "detect_image_tag rejects a digest reference with no tag" {
    IMAGE_TAG=""
    stub_central_image "quay.io/stackrox-io/main@sha256:0123456789abcdef"

    run detect_image_tag

    assert_failure
    assert_output --partial "Cannot parse tag from Central image"
}

@test "detect_image_tag rejects a host:port reference with no tag" {
    IMAGE_TAG=""
    stub_central_image "localhost:5000/stackrox/main"

    run detect_image_tag

    assert_failure
    assert_output --partial "Cannot parse tag from Central image"
}

@test "detect_image_tag rejects a parsed tag sed would treat as syntax" {
    IMAGE_TAG=""
    stub_central_image "quay.io/stackrox-io/main:foo&bar"

    run detect_image_tag

    assert_failure
    assert_output --partial "Invalid image tag"
}
