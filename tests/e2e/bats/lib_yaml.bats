#!/usr/bin/env bats

# This file contains tests for the YAML manipulation functions in lib-yaml.sh.
# These functions are used in the roxie-based deployment scripts.
#
# This test suite can be run using:
#
#   bats tests/e2e/bats/lib_yaml.bats

# shellcheck disable=SC1091
load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib-yaml.sh"
    test_file="${BATS_TEST_TMPDIR}/test.yaml"
}

# -- merge_yaml --

@test "merge_yaml into empty file" {
    touch "$test_file"
    echo 'foo: bar' | merge_yaml "$test_file"
    run yq eval '.foo' "$test_file"
    assert_success
    assert_output "bar"
}

@test "merge_yaml overwrites existing key" {
    echo 'foo: old' > "$test_file"
    echo 'foo: new' | merge_yaml "$test_file"
    run yq eval '.foo' "$test_file"
    assert_success
    assert_output "new"
}

@test "merge_yaml preserves existing keys" {
    echo -e 'foo: bar\nbaz: qux' > "$test_file"
    echo 'foo: new' | merge_yaml "$test_file"
    run yq eval '.baz' "$test_file"
    assert_success
    assert_output "qux"
}

@test "merge_yaml deep merges nested keys" {
    cat > "$test_file" <<'EOF'
parent:
  child1: one
  child2: two
EOF
    cat <<EOF | merge_yaml "$test_file"
parent:
  child2: updated
  child3: three
EOF
    run yq eval '.parent.child1' "$test_file"
    assert_output "one"
    run yq eval '.parent.child2' "$test_file"
    assert_output "updated"
    run yq eval '.parent.child3' "$test_file"
    assert_output "three"
}

# -- patch_yaml --

@test "patch_yaml sets a value" {
    echo 'foo: bar' > "$test_file"
    patch_yaml "$test_file" '.foo = "patched"'
    run yq eval '.foo' "$test_file"
    assert_success
    assert_output "patched"
}

@test "patch_yaml adds a new key" {
    echo 'foo: bar' > "$test_file"
    patch_yaml "$test_file" '.newkey = "hello"'
    run yq eval '.newkey' "$test_file"
    assert_success
    assert_output "hello"
}

# -- init_yaml_path_as_list --

@test "init_yaml_path_as_list creates list at missing path" {
    echo '{}' > "$test_file"
    init_yaml_path_as_list "$test_file" ".items"
    run yq eval '.items | length' "$test_file"
    assert_success
    assert_output "0"
}

@test "init_yaml_path_as_list preserves existing list" {
    cat > "$test_file" <<EOF
items:
  - one
  - two
EOF
    init_yaml_path_as_list "$test_file" ".items"
    run yq eval '.items | length' "$test_file"
    assert_success
    assert_output "2"
}

# -- set_custom_env --

@test "set_custom_env adds env var to component" {
    echo '{}' > "$test_file"
    set_custom_env "$test_file" "central" "MY_VAR" "my_value"
    run yq eval '.central.spec.customize.envVars[0].name' "$test_file"
    assert_output "MY_VAR"
    run yq eval '.central.spec.customize.envVars[0].value' "$test_file"
    assert_output "my_value"
}

@test "set_custom_env appends multiple env vars" {
    echo '{}' > "$test_file"
    set_custom_env "$test_file" "central" "VAR1" "val1"
    set_custom_env "$test_file" "central" "VAR2" "val2"
    run yq eval '.central.spec.customize.envVars | length' "$test_file"
    assert_output "2"
    run yq eval '.central.spec.customize.envVars[0].name' "$test_file"
    assert_output "VAR1"
    run yq eval '.central.spec.customize.envVars[0].value' "$test_file"
    assert_output "val1"
    run yq eval '.central.spec.customize.envVars[1].name' "$test_file"
    assert_output "VAR2"
    run yq eval '.central.spec.customize.envVars[1].value' "$test_file"
    assert_output "val2"
}
