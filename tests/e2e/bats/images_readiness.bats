#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

setup() {
    source "${BATS_TEST_DIRNAME}/../../upgrade/lib.sh"
    export IMAGE_REPROCESSING_READINESS_TIMEOUT=30
    export graphql_calls="${BATS_TEST_TMPDIR}/calls"
    echo 0 >"${graphql_calls}"
}

info() {
    echo "$*"
}

die() {
    echo "$*" >&2
    exit 1
}

# sleep replaces the readiness loop's poll delay. Advancing SECONDS, which the
# loop uses for its deadline, lets timeout tests reach that deadline without
# waiting in real time.
sleep() {
    SECONDS=$((SECONDS + $1))
}

# next_graphql_call prints the zero-based index of the current GraphQL call.
next_graphql_call() {
    local call
    call="$(cat "${graphql_calls}")"
    echo "$((call + 1))" >"${graphql_calls}"
    echo "$call"
}

@test "image preflight retries until Central serves images for the cluster" {
    central_graphql_query() {
        local call
        call="$(next_graphql_call)"
        if [[ "$1" == *summary_counts* ]]; then
            echo '{"data":{"imageCount":3}}'
        else
            case "$call" in
                1) echo '{"data":{"images":[]}}' ;;
                *) echo '{"data":{"images":[{"id":"sha256:abc"}]}}' ;;
            esac
        fi
    }

    run wait_for_images_in_central 268c98c6-e983-4f4e-95d2-9793cebddfd7
    assert_success
    assert_output --partial "Images in Central: 3, images for cluster 268c98c6-e983-4f4e-95d2-9793cebddfd7: 0"
    assert_output --partial "Central serves images for cluster 268c98c6-e983-4f4e-95d2-9793cebddfd7"
    [[ "$(cat "${graphql_calls}")" == 4 ]]
}

@test "image preflight requires a non-zero image count" {
    central_graphql_query() {
        if [[ "$1" == *summary_counts* ]]; then
            echo '{"data":{"imageCount":0}}'
        else
            echo '{"data":{"images":[{"id":"sha256:abc"}]}}'
        fi
    }

    run wait_for_images_in_central 268c98c6-e983-4f4e-95d2-9793cebddfd7
    assert_failure
    assert_output --partial "Images in Central: 0, images for cluster 268c98c6-e983-4f4e-95d2-9793cebddfd7: 1"
    assert_output --partial "within 30s"
}

@test "image preflight treats missing GraphQL data as empty" {
    central_graphql_query() {
        echo '{"data":null}'
    }

    run wait_for_images_in_central 268c98c6-e983-4f4e-95d2-9793cebddfd7
    assert_failure
    assert_output --partial "Images in Central: 0, images for cluster 268c98c6-e983-4f4e-95d2-9793cebddfd7: 0"
    assert_output --partial "within 30s"
}

@test "image preflight reports GraphQL errors instead of counting zero" {
    central_graphql_query() {
        echo '{"errors":[{"message":"database unavailable"}]}'
    }

    run wait_for_images_in_central 268c98c6-e983-4f4e-95d2-9793cebddfd7
    assert_failure
    assert_output --partial 'Unexpected GraphQL response from Central: {"errors":[{"message":"database unavailable"}]}'
    assert_output --partial "within 30s"
}

@test "cluster image count queries images scoped to the cluster" {
    central_graphql_query() {
        if ! [[ "$1" == *'"Cluster ID:268c98c6-e983-4f4e-95d2-9793cebddfd7"'* ]]; then
            echo "unexpected payload: $1" >&2
            return 1
        fi
        echo '{"data":{"images":[{"id":"sha256:abc"},{"id":"sha256:def"}]}}'
    }

    run central_cluster_image_count 268c98c6-e983-4f4e-95d2-9793cebddfd7
    assert_success
    assert_output "2"
}
