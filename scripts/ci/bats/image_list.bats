#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    unset OPENSHIFT_CI
    unset PULL_BASE_REF
    unset CLONEREFS_OPTIONS
    unset GITHUB_ACTION
    unset GITHUB_BASE_REF
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "get images for upgrade test" {
  local tag
  tag="$(make --quiet --no-print-directory tag)"
  local image_list
  image_list="$(mktemp)"
  CI_JOB_NAME="branch-ci-stackrox-stackrox-master-merge-gke-upgrade-tests" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "main ${tag}"
  assert_output --partial "roxctl ${tag}"
  assert_output --partial "central-db ${tag}"
}

@test "get Konflux images for DB backup and restore test" {
  local tag
  tag="$(make --quiet --no-print-directory tag)-fast"
  local image_list
  image_list="$(mktemp)"
  USE_KONFLUX_IMAGES=true MAIN_IMAGE_TAG="$tag" CI_JOB_NAME="gke-db-backup-restore-tests-konflux" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "release-main ${tag}"
  assert_output --partial "release-central-db ${tag}"
  assert_output --partial "release-roxctl ${tag}"
}

@test "get images for VM scanning test" {
  local tag
  tag="$(make --quiet --no-print-directory tag)"
  local image_list
  image_list="$(mktemp)"
  CI_JOB_NAME="ocp-vm-scanning-tests" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "main ${tag}"
  assert_output --partial "central-db ${tag}"
  assert_output --partial "collector ${tag}"
  assert_output --partial "fact ${tag}"
  assert_output --partial "scanner-v4 ${tag}"
  assert_output --partial "scanner-v4-db ${tag}"
  assert_output --partial "roxctl ${tag}"
  refute_output --partial "release-"
}

@test "get Konflux images for VM scanning test" {
  local tag
  tag="$(make --quiet --no-print-directory tag)-fast"
  local operator_tag
  operator_tag="$(BUILD_TAG="$tag" make -C operator --quiet --no-print-directory tag)"
  local image_list
  image_list="$(mktemp)"
  # VM scanning always deploys via the operator, same as the real job
  # (scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py sets DEPLOY_STACKROX_VIA_OPERATOR=true).
  USE_KONFLUX_IMAGES=true DEPLOY_STACKROX_VIA_OPERATOR=true MAIN_IMAGE_TAG="$tag" CI_JOB_NAME="ocp-vm-scanning-tests-konflux" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "release-main ${tag}"
  assert_output --partial "release-central-db ${tag}"
  assert_output --partial "release-collector ${tag}"
  assert_output --partial "release-fact ${tag}"
  assert_output --partial "release-scanner-v4 ${tag}"
  assert_output --partial "release-scanner-v4-db ${tag}"
  assert_output --partial "release-roxctl ${tag}"
  assert_output --partial "release-operator ${operator_tag}"
  assert_output --partial "release-operator-bundle v${operator_tag}"
  # operator-index is always published under the plain (non "release-") name
  # by Konflux too, see .tekton/operator-index-build.yaml.
  assert_output --partial "stackrox-operator-index v${operator_tag}"
}

@test "get Konflux images for VM scanning test, tag without -fast suffix" {
  # This tests actual release build logic, versus pre-release build testing.
  # Konflux's operator-index build always forces a "-fast" suffix on the index
  # tag, even for tags that otherwise wouldn't carry one (e.g. release tags),
  # to avoid colliding with GHA-built index images of the same name. See the
  # determine-index-tag task in .tekton/operator-index-build.yaml.
  local tag
  tag="$(make --quiet --no-print-directory tag)"
  local operator_tag
  operator_tag="$(BUILD_TAG="$tag" make -C operator --quiet --no-print-directory tag)"
  local image_list
  image_list="$(mktemp)"
  USE_KONFLUX_IMAGES=true DEPLOY_STACKROX_VIA_OPERATOR=true MAIN_IMAGE_TAG="$tag" CI_JOB_NAME="ocp-vm-scanning-tests-konflux" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "release-main ${tag}"
  assert_output --partial "stackrox-operator-index v${operator_tag}-fast"
}

@test "get images for rcd test" {
  local tag
  tag="$(make --quiet --no-print-directory tag)"
  local image_list
  image_list="$(mktemp)"
  # Simulate the real job: gke_race_condition_qa_e2e_tests.py sets MAIN_IMAGE_TAG with -rcd
  MAIN_IMAGE_TAG="${tag}-rcd" CI_JOB_NAME="master-race-condition-qa-e2e-tests" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "main ${tag}-rcd"
  refute_output --partial "-rcd-rcd"
  assert_output --partial "roxctl ${tag}"
  assert_output --partial "central-db ${tag}"
}
