#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
  # remove binaries from the previous runs
  delete-outdated-binaries "$(roxctl-development version)"

  echo "Testing roxctl version: '$(roxctl-development version)'" >&3
  command -v yq > /dev/null || skip "Tests in this file require yq"
  command -v expect || skip "Tests in this file require expect"
}

setup() {
  export out_dir="$(mktemp -d -u)"
  sleep 1
}

teardown() {
  rm -rf "$out_dir"
  true
}

bitfield_to_failure() {
  local status="$1"
  ((status == 0)) && return 0
  if ((status & 8)); then
    fail "ERROR: Missing some questions in the interactive mode"
  fi
  msg="ERROR: Missing hints about default value for:"
  if ((status & 4)); then
    msg="$msg main"
  fi
  if ((status & 2)); then
    msg="$msg scanner-db"
  fi
  if ((status & 1)); then
    msg="$msg scanner"
  fi
  fail "$msg - status $status"
}

assert_flavor_prompt_development() {
  assert_line --partial 'Default container images settings (development_build, rhacs, opensource); it controls repositories from where to download the images, image names and tags format (default: "development_build"):'
}

assert_flavor_prompt_release() {
  assert_line --partial 'Default container images settings (rhacs, opensource); it controls repositories from where to download the images, image names and tags format (default: "rhacs"):'
}

assert_prompts_development() {
  # partial line matching allows to avoid problems with leading an trailing whitespaces
  # main/scanner/scanner-db are constants from code
  assert_line --regexp '^Main .* "quay.io/rhacs-eng/main:'
  assert_line --regexp '^Scanner-db .* "quay.io/rhacs-eng/scanner-db:'
  assert_line --regexp '^Scanner .* "quay.io/rhacs-eng/scanner:'
}

assert_prompts_opensource() {
  assert_line --regexp '^Main .* "quay.io/stackrox-io/main:'
  assert_line --regexp '^Scanner-db .* "quay.io/stackrox-io/scanner-db:'
  assert_line --regexp '^Scanner .* "quay.io/stackrox-io/scanner:'
}

assert_prompts_rhacs() {
  assert_line --regexp '^Main .* "registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8:'
  assert_line --regexp '^Scanner-db .* "registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-rhel8:'
  assert_line --regexp '^Scanner .* "registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8:'
}

@test "roxctl-development central generate interactive flavor=dummy should ask for valid value" {
  roxctl_bin="$(roxctl-development-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive-dummy.expect.tcl" -- "$roxctl_bin" "$out_dir"
  assert_success
}

@test "roxctl-development central generate interactive flavor=development_build" {
  roxctl_bin="$(roxctl-development-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- "$roxctl_bin" development_build "$out_dir" "quay.io/rhacs-eng"
  bitfield_to_failure "$status"
  assert_success
  assert_prompts_development
  assert_flavor_prompt_development
  sleep 2 # due to frequent flakes of missing yaml files
  assert_components_registry "$out_dir/central" "quay.io/rhacs-eng" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "quay.io/rhacs-eng" "$any_version" 'scanner' 'scanner-db'
}

@test "roxctl-development central generate interactive flavor=opensource" {
  roxctl_bin="$(roxctl-development-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- "$roxctl_bin" opensource "$out_dir" "quay.io/stackrox-io"
  bitfield_to_failure "$status"
  assert_success
  assert_prompts_opensource
  assert_flavor_prompt_development
  sleep 2 # due to frequent flakes of missing yaml files
  assert_components_registry "$out_dir/central" "quay.io/stackrox-io" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "quay.io/stackrox-io" "$any_version" 'scanner' 'scanner-db'
}

@test "roxctl-development central generate interactive flavor=rhacs" {
  roxctl_bin="$(roxctl-development-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- "$roxctl_bin" rhacs "$out_dir" "registry.redhat.io/advanced-cluster-security"
  bitfield_to_failure "$status"
  assert_success
  assert_prompts_rhacs
  assert_flavor_prompt_development
  sleep 2 # due to frequent flakes of missing yaml files
  assert_components_registry "$out_dir/central" "registry.redhat.io" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "registry.redhat.io" "$any_version" 'scanner' 'scanner-db'
}

# RELEASE

@test "roxctl-release central generate interactive flavor=dummy should ask for valid value" {
  roxctl_bin="$(roxctl-release-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive-dummy.expect.tcl" -- "$roxctl_bin" "$out_dir"
  assert_success
}

@test "roxctl-release central generate interactive flavor=opensource" {
  roxctl_bin="$(roxctl-release-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- "$roxctl_bin" opensource "$out_dir" "quay.io/stackrox-io"
  bitfield_to_failure "$status"
  assert_success
  assert_prompts_opensource
  assert_flavor_prompt_release
  sleep 2 # due to frequent flakes of missing yaml files
  assert_components_registry "$out_dir/central" "quay.io/stackrox-io" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "quay.io/stackrox-io" "$any_version" 'scanner' 'scanner-db'
}

@test "roxctl-release central generate interactive flavor=rhacs" {
  roxctl_bin="$(roxctl-release-cmd)"
  run expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- "$roxctl_bin" rhacs "$out_dir" "registry.redhat.io/advanced-cluster-security"
  bitfield_to_failure "$status"
  assert_success
  assert_prompts_rhacs
  assert_flavor_prompt_release
  sleep 2 # due to frequent flakes of missing yaml files
  assert_components_registry "$out_dir/central" "registry.redhat.io" "$any_version" 'main'
  assert_components_registry "$out_dir/scanner" "registry.redhat.io" "$any_version" 'scanner' 'scanner-db'
}
