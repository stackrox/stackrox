#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  echo "Testing roxctl version: '$(roxctl-release version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
  # remove binaries from the previous runs
  rm -f "$(roxctl-development-cmd)" "$(roxctl-development-release)"
}

setup() {
  out_dir="$(mktemp -d -u)"
}

teardown() {
  rm -rf "$out_dir"
}

@test "roxctl-release generate netpol should return error on empty or non-existing directory" {
  run roxctl-release generate netpol "$out_dir"
  assert_failure
  assert_line --partial "Error synthesizing policies from folder: no deployment objects discovered in the repository"

  run roxctl-release generate netpol
  assert_failure
  assert_line --partial "missing <folder-path> argument"
}

@test "roxctl-release generate netpol generates network policies" {
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/frontend.yaml"
  assert_file_exist "${test_data}/np-guard/scenario-minimal-service/backend.yaml"
  ofile="$(mktemp)"
  echo "Writing network policies to ${ofile}" >&3
  run roxctl-release generate netpol "${test_data}/np-guard/scenario-minimal-service"
  assert_success
  echo "$output" > "$ofile"
  assert_file_exist "$ofile"
  yaml_valid "$ofile"

  # There must be at least 2 yaml documents in the output
  run yq e 'document_index' "${ofile}"
  assert_line '0'
  assert_line '1'

  # Ensure that both yaml docs are of kind 'NetworkPolicy'
  run yq e '.kind | ({"match": ., "doc": document_index})' "${ofile}"
  assert_line --index 0 'match: NetworkPolicy'
  assert_line --index 1 'doc: 0'
  assert_line --index 2 'match: NetworkPolicy'
  assert_line --index 3 'doc: 1'


}


