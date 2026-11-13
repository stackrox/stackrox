#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../../scripts/test_helpers.bats"

# The compatibility layer invokes these stubs indirectly.
# shellcheck disable=SC2329
setup() {
    source "${BATS_TEST_DIRNAME}/../lib-yaml.sh"
    source "${BATS_TEST_DIRNAME}/../lib-compat.sh"
    info() { :; }
    ci_export() { :; }
    retrying_kubectl() { cat >/dev/null; }

    unset ROXDEPLOY_CONFIG_FILE_MAP ROX_PLAINTEXT_ENDPOINTS EXTERNAL_DB
    unset ROX_DEFAULT_TLS_KEY_FILE ROX_DEFAULT_TLS_CERT_FILE TRUSTED_CA_FILE
    unset STORAGE_CLASS SCANNER_V4_DB_STORAGE_CLASS
    export CI_JOB_NAME=gke-nongroovy-e2e-tests
    export LOAD_BALANCER=lb ROX_SCANNER_V4=true SFA_AGENT=false
    test_file="${BATS_TEST_TMPDIR}/config.yaml"
    cp "${TEST_ROOT}/deploy/roxie-config-resources.yaml" "$test_file"
    db_overlay=$(yq -o=json -I=0 '.central.spec.overlays[] | select(.name == "central-db")' "$test_file")
    db_resources=$(yq -o=json -I=0 '.central.spec.central.db.resources' "$test_file")
}

assert_db_config_preserved() {
    run yq -o=json -I=0 '.central.spec.overlays[] | select(.name == "central-db")' "$test_file"
    assert_success
    assert_output "$db_overlay"
    run yq -o=json -I=0 '.central.spec.central.db.resources' "$test_file"
    assert_success
    assert_output "$db_resources"
}

@test "non-Groovy compatibility settings preserve DB tuning alongside endpoints" {
    export ROXDEPLOY_CONFIG_FILE_MAP="${BATS_TEST_TMPDIR}/endpoints.yaml"
    export ROX_PLAINTEXT_ENDPOINTS=8080,8081
    cat > "$ROXDEPLOY_CONFIG_FILE_MAP" <<'EOF'
endpoints:
  - listen: ":8082"
    tls:
      disable: true
EOF

    roxie_config_from_environment_compat "$test_file" stackrox

    assert_db_config_preserved
    run yq '.central.spec.overlays[] | select(.name == "central-endpoints") | .patches[0].verbatim' "$test_file"
    assert_success
    assert_output "$(cat "$ROXDEPLOY_CONFIG_FILE_MAP")"
    run yq '.central.spec.customize.envVars[] | select(.name == "ROX_PLAINTEXT_ENDPOINTS") | .value' "$test_file"
    assert_success
    assert_output "8080,8081"
}

@test "QA and UI compatibility settings preserve DB tuning without endpoint overrides" {
    roxie_config_from_environment_compat "$test_file" stackrox

    assert_db_config_preserved
    run yq '.central.spec.overlays | length' "$test_file"
    assert_success
    assert_output "1"
}

@test "BYODB compatibility settings retain an optional managed DB overlay" {
    export EXTERNAL_DB=true EXTERNAL_DATABASE_HOST=postgres.example
    export EXTERNAL_DB_USER=stackrox EXTERNAL_DATABASE_NAME=central
    export EXTERNAL_DB_PASSWORD=test-password

    roxie_config_from_environment_compat "$test_file" stackrox

    assert_db_config_preserved
    run yq '.central.spec.overlays[] | select(.name == "central-db") | .optional' "$test_file"
    assert_success
    assert_output "true"
    run yq '.central.spec.central.db.connectionString' "$test_file"
    assert_success
    assert_output --partial "host=postgres.example"
}
