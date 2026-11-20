#!/usr/bin/env bats

setup() {
    local root="$BATS_TEST_TMPDIR/repo"
    mkdir -p "$root/.openshift-ci" "$root/scripts/ci" "$BATS_TEST_TMPDIR/shared"
    cp "$BATS_TEST_DIRNAME/../../.openshift-ci/begin.sh" "$root/.openshift-ci/"
    # Run the real begin script without cloud credentials or DNS dependencies.
    cat > "$root/scripts/ci/lib.sh" <<'EOF'
info() { :; }
changes_limited_to() { return 1; }
openshift_ci_mods() { :; }
openshift_ci_import_creds() { :; }
retry() { return 0; }
set_ci_shared_export() {
    export "$1=$2"
    printf 'export %s=%q\n' "$1" "$2" >> "$SHARED_DIR/shared_env"
}
EOF
    echo 'setup_gcp() { :; }' > "$root/scripts/ci/gcp.sh"
    export SHARED_DIR="$BATS_TEST_TMPDIR/shared"
    export GITHUB_ENV=/dev/null
    unset MASTER_NODE_COUNT MASTER_NODE_TYPE WORKER_NODE_COUNT WORKER_NODE_TYPE
}

check_topology() {
    local job="$1" workers="$2" worker_type="$3" sno="$4"
    : > "$SHARED_DIR/shared_env"
    run env JOB_NAME="$job" bash "$BATS_TEST_TMPDIR/repo/.openshift-ci/begin.sh"
    [ "$status" -eq 0 ]
    grep -qx "export WORKER_NODE_COUNT=$workers" "$SHARED_DIR/shared_env"
    grep -qx "export WORKER_NODE_TYPE=$worker_type" "$SHARED_DIR/shared_env"
    if [[ "$sno" == true ]]; then
        grep -qx 'export MASTER_NODE_COUNT=1' "$SHARED_DIR/shared_env"
        grep -qx 'export MASTER_NODE_TYPE=e2-standard-16' "$SHARED_DIR/shared_env"
    else
        ! grep -q 'export MASTER_NODE_' "$SHARED_DIR/shared_env"
    fi
}

@test "selected functional presubmits use SNO while postsubmits stay multi-node" {
    for version in 4-22 5-0 next-candidate dev-preview; do
        for suite in operator-e2e-tests nongroovy-e2e-tests scanner-v4-install-tests qa-e2e-tests ui-e2e-tests; do
            check_topology "pull-ci-stackrox-stackrox-master-ocp-$version-$suite" 0 e2-standard-8 true
            check_topology "branch-ci-stackrox-stackrox-master-ocp-$version-merge-$suite" 2 e2-standard-8 false
        done
    done
}

@test "other versions and specialized suites retain existing topology" {
    for version in 4-12 4-14 4-18 4-21 5-1; do
        for suite in operator-e2e-tests nongroovy-e2e-tests scanner-v4-install-tests qa-e2e-tests ui-e2e-tests; do
            check_topology "pull-ci-stackrox-stackrox-master-ocp-$version-$suite" 2 e2-standard-8 false
        done
    done
    for version in 4-22 5-0 next-candidate dev-preview; do
        check_topology "pull-ci-stackrox-stackrox-master-ocp-$version-vm-scanning-e2e-tests" 2 n2-standard-8 false
        for suite in compliance-e2e-tests fips-qa-e2e-tests unknown-e2e-tests; do
            check_topology "pull-ci-stackrox-stackrox-master-ocp-$version-$suite" 2 e2-standard-8 false
        done
    done
}

@test "non-OCP jobs receive no topology overrides" {
    run env JOB_NAME=pull-ci-stackrox-stackrox-master-gke-qa-e2e-tests bash "$BATS_TEST_TMPDIR/repo/.openshift-ci/begin.sh"
    [ "$status" -eq 0 ]
    run grep -Eq 'export (MASTER|WORKER)_NODE_' "$SHARED_DIR/shared_env"
    [ "$status" -eq 1 ]
}

@test "topology logging tolerates exports deferred to later steps" {
    cat >> "$BATS_TEST_TMPDIR/repo/scripts/ci/lib.sh" <<'EOF'
info() { echo "$*"; }
set_ci_shared_export() {
    printf 'export %s=%q\n' "$1" "$2" >> "$SHARED_DIR/shared_env"
}
EOF
    check_topology pull-ci-stackrox-stackrox-master-ocp-4-22-qa-e2e-tests 0 e2-standard-8 true
    [[ "$output" == *"OCP masters=provisioner-default master-type=provisioner-default workers=unset worker-type=unset"* ]]
}
