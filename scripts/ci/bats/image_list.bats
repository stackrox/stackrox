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
  USE_KONFLUX_IMAGES=true DEPLOY_STACKROX_VIA_OPERATOR=true MAIN_IMAGE_TAG="$tag" CI_JOB_NAME="gke-db-backup-restore-tests-konflux" populate_stackrox_image_list "$image_list"
  run cat "${image_list}"
  assert_success
  assert_output --partial "release-main ${tag}"
  assert_output --partial "release-central-db ${tag}"
  assert_output --partial "release-roxctl ${tag}"
  assert_output --partial "release-operator ${tag}"
  assert_output --partial "release-operator-bundle v${tag}"
  assert_output --partial "stackrox-operator-index v${tag}"
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
