#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

@test "write_github_summary writes the VM access and service summary" {
    summary_file="${BATS_TEST_TMPDIR}/summary.md"

    run env GITHUB_STEP_SUMMARY="$summary_file" bash -c '
        source "$1"
        NAMESPACE="openshift-cnv"
        VM_OS="rhel10"
        VM_PREFIX="rhel10"
        NUM_VMS=2
        MANAGED_VMS=("rhel10-1")
        ADOPTED_VMS=("rhel10-2")
        SKIPPED_VMS=("rhel10-3")
        NATIVE_AGENT_READY_VMS=("rhel10-1" "rhel10-2")
        NATIVE_AGENT_FAILED_VMS=("rhel10-3")
        write_github_summary
    ' _ "${BATS_TEST_DIRNAME}/add-vms.sh"

    assert_success

    run cat "$summary_file"
    assert_output --partial "## Add VMs to Cluster"
    assert_output --partial "### Native agent service verification"
    assert_output --partial "Successfully started on:"
    assert_output --partial "rhel10-1"
    assert_output --partial "rhel10-2"
    assert_output --partial "Needs attention:"
    assert_output --partial "### SSH access"
    assert_output --partial "add-vms-id_ed25519"
    assert_output --partial "virtctl ssh -n openshift-cnv --identity-file ./add-vms-id_ed25519 cloud-user@vmi/rhel10-1"
}
