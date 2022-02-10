#!/usr/bin/env bats

# Allow to run the tests locally provided that bats-helpers are installed in $HOME/bats-core
bats_helpers_root="${HOME}/bats-core"
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  # Location of bats-helpers in the CI image
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"

function setup() {
    source "${BATS_TEST_DIRNAME}/store-artifacts.sh"
    make_fake_CI_env
    mock_gcloud
    mock_gsutil
}

@test "missing source path argument" {
    run store_artifacts
    assert_failure 1
    assert_output --partial 'missing args'
}

@test "non existing source is ignored" {
    run store_artifacts /something-missing
    assert_success 0
    assert_output --partial 'something-missing is missing, nothing to upload'
}

@test "empty source is ignored" {
    local emptydir="${BATS_TEST_TMPDIR}/empty"
    mkdir "$emptydir"
    run store_artifacts "$emptydir"
    assert_success 0
    assert_output --partial 'empty is empty, nothing to upload'
}

@test "stores" {
    run store_artifacts /tmp
    assert_success
    assert_output --partial "Destination: gs://roxci-artifacts/stackrox/12345/theBuildId-job-name/tmp"
}

@test "stores to a different destination" {
    run store_artifacts /tmp different
    assert_success
    assert_output --partial "Destination: gs://roxci-artifacts/stackrox/12345/theBuildId-job-name/ggg"
}

@test "stores to unique destinations" {
    run store_artifacts /tmp unique
    assert_success
    assert_output --partial "Destination: gs://roxci-artifacts/stackrox/12345/theBuildId-job-name/unique-2"
}

@test "stores to unique destinations with many existing" {
    run store_artifacts /tmp many
    assert_success
    assert_output --partial "Destination: gs://roxci-artifacts/stackrox/12345/theBuildId-job-name/many-10"
}

# shellcheck disable=SC2034

make_fake_CI_env() {
    export CI=true
    export OPENSHIFT_CI=true
    export GCLOUD_SERVICE_ACCOUNT_OPENSHIFT_CI_ROX=dummy
    export REPO_NAME="stackrox"
    export BUILD_ID="theBuildId"
    export JOB_NAME="job-name"
    export PULL_PULL_SHA="12345"
    export PATH="$BATS_RUN_TMPDIR:$PATH"
}

mock_gcloud() {
    cat <<EOS >> "$BATS_RUN_TMPDIR/gcloud"
#!/usr/bin/env bash
exit 0
EOS
    chmod 0755 "$BATS_RUN_TMPDIR/gcloud"
}

mock_gsutil() {
    cat <<EOS >> "$BATS_RUN_TMPDIR/gsutil"
#!/usr/bin/env bash
if [[ "\$1" == "ls" ]]; then
    # when checking destination
    if [[ "\$2" =~ many-?[0-9]?\$ ]]; then
        exit 0
    fi
    if [[ "\$2" =~ unique\$ ]]; then
        exit 0
    fi
    exit 1
fi
echo "Destination: \$5"
exit 0
EOS
    chmod 0755 "$BATS_RUN_TMPDIR/gsutil"
}
