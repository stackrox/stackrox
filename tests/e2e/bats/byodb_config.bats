#!/usr/bin/env bats
# shellcheck disable=SC1091

setup() {
    source "${BATS_TEST_DIRNAME}/../lib-yaml.sh"
    source "${BATS_TEST_DIRNAME}/../../byodb/lib.sh"
    export EXTERNAL_DATABASE_HOST=postgres.database
    export EXTERNAL_DB_USER=stackrox
    export EXTERNAL_DATABASE_NAME=stackrox
    export EXTERNAL_DB_PASSWORD=byodb-test-password
    config_file="${BATS_TEST_TMPDIR}/roxie.yaml"
    echo 'operator: {version: test-version}' > "$config_file"
}

retrying_kubectl() {
    echo "$*" >> "${BATS_TEST_TMPDIR}/kubectl-args"
    if [[ "$1" == apply ]]; then
        cat > "${BATS_TEST_TMPDIR}/namespace.yaml"
    else
        cat > "${BATS_TEST_TMPDIR}/secret.yaml"
    fi
}

@test "BYODB supplies external database settings without putting the password in roxie config" {
    run configure_byodb "$config_file" stackrox
    [ "$status" -eq 0 ]

    [ "$(yq '.central.namespace' "$config_file")" = stackrox ]
    [ "$(yq '.central.spec.central.db.connectionString' "$config_file")" = \
        'host=postgres.database client_encoding=UTF8 user=stackrox dbname=stackrox statement_timeout=1200000' ]
    [ "$(yq '.central.spec.central.db.passwordSecret.name' "$config_file")" = central-external-db-password ]
    [ "$(yq '.operator.version' "$config_file")" = test-version ]
    run grep -q "$EXTERNAL_DB_PASSWORD" "$config_file"
    [ "$status" -eq 1 ]
}

@test "BYODB creates its password secret after the namespace and owns it independently of compat cleanup" {
    run configure_byodb "$config_file" stackrox
    [ "$status" -eq 0 ]

    [ "$(cat "${BATS_TEST_TMPDIR}/kubectl-args")" = $'apply -f -\n-n stackrox apply -f -' ]
    [ "$(yq '.kind' "${BATS_TEST_TMPDIR}/namespace.yaml")" = Namespace ]
    [ "$(yq '.metadata.name' "${BATS_TEST_TMPDIR}/namespace.yaml")" = stackrox ]
    local secret_file="${BATS_TEST_TMPDIR}/secret.yaml"
    [ "$(yq '.kind' "$secret_file")" = Secret ]
    [ "$(yq '.metadata.name' "$secret_file")" = "$(yq '.central.spec.central.db.passwordSecret.name' "$config_file")" ]
    [ "$(yq '.metadata.labels."app.kubernetes.io/managed-by"' "$secret_file")" = byodb-tests ]
    [ "$(yq '.data.password | @base64d' "$secret_file")" = "$EXTERNAL_DB_PASSWORD" ]
}
