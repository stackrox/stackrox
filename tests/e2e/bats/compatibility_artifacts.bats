#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../../scripts/test_helpers.bats"

# The sourced orchestration invokes these stubs indirectly.
# shellcheck disable=SC2329
setup() {
    source "${BATS_TEST_DIRNAME}/../run-compatibility.sh"

    ROOT="${BATS_TEST_TMPDIR}/repo"
    mkdir -p "${ROOT}/scripts"
    printf '#!/usr/bin/env bash\nexit 0\n' > "${ROOT}/scripts/roxie.sh"
    chmod +x "${ROOT}/scripts/roxie.sh"
    cd "${ROOT}" || return 1
    artifact_pair=""

    # Stub cluster operations; exercise the orchestration's artifact destinations.
    require_environment() { :; }
    export_test_environment() { :; }
    test_preamble() { :; }
    setup_deployment_env() { :; }
    setup_default_TLS_certs() { :; }
    deploy_stackrox_with_custom_central_and_sensor_versions() { artifact_pair="$1/$2"; }
    setup_client_TLS_certs() { :; }
    prepare_for_endpoints_test() { :; }
    wait_for_api() { :; }
    pr_has_label() { return 1; }
    kubectl() { :; }
    make() { :; }
    update_junit_prefix_with_central_and_sensor_version() { :; }

    store_test_results() {
        mkdir -p "${BATS_TEST_TMPDIR}/results/$2"
        printf '%s\n' "${artifact_pair}" > "${BATS_TEST_TMPDIR}/results/$2/pair"
    }
    collect_and_check_stackrox_logs() {
        mkdir -p "${BATS_TEST_TMPDIR}/logs/$2"
        printf '%s\n' "${artifact_pair}" > "${BATS_TEST_TMPDIR}/logs/$2/pair"
    }
}

@test "opposite same-minor version pairs retain separate results and logs" {
    run run_compatibility_tests "5.0.1-rc.0-5-g54c0e50f2d" "5.0.0"
    assert_success
    run run_compatibility_tests "5.0.0" "5.0.1-rc.0-5-g54c0e50f2d"
    assert_success

    for artifact in results logs; do
        run find "${BATS_TEST_TMPDIR}/${artifact}" -name pair -exec cat {} \;
        assert_success
        assert_output --partial "5.0.1-rc.0-5-g54c0e50f2d/5.0.0"
        assert_output --partial "5.0.0/5.0.1-rc.0-5-g54c0e50f2d"
    done
}
