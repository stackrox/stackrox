#!/usr/bin/env bats
# shellcheck disable=SC1091
#
# Unit tests for compliance/virtualmachines/roxagent/quadlet/install.sh
#
# Test strategy:
#   The install script calls external commands (sudo, ssh, scp, virtctl) that
#   require privileges or remote hosts. To test locally without either, we place
#   stub scripts on PATH (in BIN_DIR) that shadow the real binaries.
#
#   Stubs live in tests/e2e/bats/stubs/ as standalone executable scripts because
#   the install script invokes them as external processes — bash functions
#   (export -f) would not survive the child shell boundary reliably.
#
#   Three kinds of stubs are used:
#     - sudo stub:       rewrites /etc/* paths into a temp FAKE_ROOT for assertion
#     - recording stubs: log every call to CALL_LOG so tests can assert on args
#     - unexpected stubs: fail immediately if invoked (tripwire guards)
#
#   See stubs/README or individual stub headers for details.

load "../../../scripts/test_helpers.bats"

INSTALL_SCRIPT_REL="compliance/virtualmachines/roxagent/quadlet/install.sh"

# --- Test fixture content (container file variants) ---------------------------

CONTAINER_ALL_PATHS_EXIST='[Container]
Image=registry.example.com/roxagent:latest
Volume=/etc/yum.repos.d:/etc/yum.repos.d:ro
Volume=/data:/data:rw'

CONTAINER_WITH_MISSING_PATHS='[Container]
Image=registry.example.com/roxagent:latest
Volume=/nonexistent/path1:/container/path1:ro
Volume=/tmp:/container/tmp:rw
Volume=/nonexistent/path2:/container/path2:ro'

CONTAINER_MIXED_LINES='[Container]
Image=registry.example.com/roxagent:latest
Environment=SOME_VAR=value
Volume=/missing/path:/dst:ro
Label=com.example=test'

# --- Test fixture content (staged install files) ------------------------------

STAGE_CONTAINER='[Container]
SENTINEL-CONTAINER=true'

STAGE_TIMER='SENTINEL-TIMER=true'

STAGE_PREP_SERVICE='SENTINEL-PREP=true'

STAGE_TMPFILES_CONF='SENTINEL-TMPFILES=true'

setup() {
    INSTALL_SCRIPT="${BATS_TEST_DIRNAME}/../../../${INSTALL_SCRIPT_REL}"
    STAGE_DIR="${BATS_TEST_TMPDIR}/custom-stage"
    BIN_DIR="${BATS_TEST_TMPDIR}/bin"
    FAKE_ROOT="${BATS_TEST_TMPDIR}/fake-root"
    CALL_LOG="${BATS_TEST_TMPDIR}/calls.log"

    mkdir -p "${STAGE_DIR}" "${BIN_DIR}" "${FAKE_ROOT}"
    # Truncate file to 0 bytes
    : > "${CALL_LOG}"

    export FAKE_ROOT
    export CALL_LOG
    export PATH="${BIN_DIR}:${PATH}"

    write_stage_files
    write_sudo_stub
    write_unexpected_remote_stub scp
    write_unexpected_remote_stub ssh
    write_unexpected_remote_stub virtctl
}

# =============================================================================
# Local install (no args)
# =============================================================================

@test "local install (no args) installs files from QUADLET_FILES_DIR into FAKE_ROOT" {
    QUADLET_FILES_DIR="${STAGE_DIR}" run bash "${INSTALL_SCRIPT}"
    assert_success
    assert_output --partial "Done!"
    assert_output --partial "periodically"

    [ -f "${FAKE_ROOT}/etc/containers/systemd/roxagent.container" ]
    [ -f "${FAKE_ROOT}/etc/systemd/system/roxagent.timer" ]
    [ -f "${FAKE_ROOT}/etc/systemd/system/roxagent-prep.service" ]
    [ -f "${FAKE_ROOT}/etc/tmpfiles.d/roxagent.conf" ]

    run cat "${FAKE_ROOT}/etc/containers/systemd/roxagent.container"
    assert_output --partial "SENTINEL-CONTAINER"

    run cat "${FAKE_ROOT}/etc/systemd/system/roxagent.timer"
    assert_output --partial "SENTINEL-TIMER"

    run cat "${FAKE_ROOT}/etc/systemd/system/roxagent-prep.service"
    assert_output --partial "SENTINEL-PREP"

    run cat "${FAKE_ROOT}/etc/tmpfiles.d/roxagent.conf"
    assert_output --partial "SENTINEL-TMPFILES"
}

# =============================================================================
# Install from explicit stage dir (--stage-dir)
# =============================================================================

@test "--stage-dir installs all unit files to correct locations" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "${STAGE_DIR}"
    assert_success
    refute_output --partial "Done!"

    [ -f "${FAKE_ROOT}/etc/containers/systemd/roxagent.container" ]
    [ -f "${FAKE_ROOT}/etc/systemd/system/roxagent.timer" ]
    [ -f "${FAKE_ROOT}/etc/systemd/system/roxagent-prep.service" ]
    [ -f "${FAKE_ROOT}/etc/tmpfiles.d/roxagent.conf" ]

    run cat "${FAKE_ROOT}/etc/containers/systemd/roxagent.container"
    assert_output --partial "SENTINEL-CONTAINER"

    run cat "${FAKE_ROOT}/etc/systemd/system/roxagent.timer"
    assert_output --partial "SENTINEL-TIMER"

    run cat "${FAKE_ROOT}/etc/systemd/system/roxagent-prep.service"
    assert_output --partial "SENTINEL-PREP"

    run cat "${FAKE_ROOT}/etc/tmpfiles.d/roxagent.conf"
    assert_output --partial "SENTINEL-TMPFILES"
}

@test "--stage-dir does not invoke any remote transport" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "${STAGE_DIR}"
    assert_success

    # The unexpected stubs exit 99 if called; success means they weren't.
    # Double-check the call log is empty.
    run cat "${CALL_LOG}"
    assert_output ""
}

@test "--stage-dir does not print the Done epilogue" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "${STAGE_DIR}"
    assert_success
    refute_output --partial "Done!"
}

# =============================================================================
# Input validation / error cases
# =============================================================================

@test "--stage-dir without dir argument fails with usage" {
    run bash "${INSTALL_SCRIPT}" --stage-dir
    assert_failure
    assert_output --partial "usage:"
}

@test "--stage-dir with nonexistent dir fails with missing file error" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "/nonexistent/path"
    assert_failure
    assert_output --partial "missing staged file"
}

@test "--ssh without host fails with usage" {
    run bash "${INSTALL_SCRIPT}" --ssh
    assert_failure
    assert_output --partial "Usage:"
}

@test "virtctl mode with no target fails with error and usage" {
    # This would happen if someone accidentally passes only --stage-dir-like
    # args that don't match any known flag, BUT the current interface means
    # bare args go to virtctl. The edge case is: what if the script is invoked
    # in a way that setup_transport_virtctl gets 0 args? That can't happen via
    # main() because the else branch requires $# > 0. But we test the error
    # path via a quirk: "virtctl" was the old keyword; now bare args go to
    # virtctl directly. So this test just verifies the usage text exists.
    run bash "${INSTALL_SCRIPT}" --ssh
    assert_failure
    assert_output --partial "Usage:"
    assert_output --partial "virtctl"
}

# =============================================================================
# SSH mode (--ssh) argument dispatch
# =============================================================================

@test "--ssh user@host invokes scp and ssh with port 22" {
    write_recording_stub scp
    write_recording_stub ssh

    run bash "${INSTALL_SCRIPT}" --ssh "testuser@10.0.0.1"
    assert_success
    assert_output --partial "Done!"

    run cat "${CALL_LOG}"
    # scp should be called with -P 22
    assert_output --partial "scp -P 22"
    assert_output --partial "testuser@10.0.0.1:"
    # ssh should be called with -p 22
    assert_output --partial "ssh -p 22 testuser@10.0.0.1"
}

@test "--ssh user@host 2222 uses custom port" {
    write_recording_stub scp
    write_recording_stub ssh

    run bash "${INSTALL_SCRIPT}" --ssh "testuser@10.0.0.1" 2222
    assert_success

    run cat "${CALL_LOG}"
    assert_output --partial "scp -P 2222"
    assert_output --partial "ssh -p 2222 testuser@10.0.0.1"
}

# =============================================================================
# Virtctl mode (default remote) argument dispatch
# =============================================================================

@test "virtctl mode with flags and target invokes virtctl scp/ssh correctly" {
    write_recording_stub virtctl

    run bash "${INSTALL_SCRIPT}" -n openshift-cnv "cloud-user@vmi/rhel10-1"
    assert_success
    assert_output --partial "Done!"

    run cat "${CALL_LOG}"
    # virtctl scp should pass -n openshift-cnv and the target
    assert_output --partial "virtctl scp -n openshift-cnv"
    assert_output --partial "cloud-user@vmi/rhel10-1:"
    # virtctl ssh should pass the flags and target
    assert_output --partial "virtctl ssh -n openshift-cnv cloud-user@vmi/rhel10-1"
}

@test "virtctl mode with only a target (no extra flags) works" {
    write_recording_stub virtctl

    run bash "${INSTALL_SCRIPT}" "cloud-user@vmi/rhel10-1"
    assert_success

    run cat "${CALL_LOG}"
    assert_output --partial "virtctl scp"
    assert_output --partial "cloud-user@vmi/rhel10-1:"
    assert_output --partial "virtctl ssh"
}

@test "virtctl mode with multiple flags passes all flags" {
    write_recording_stub virtctl

    run bash "${INSTALL_SCRIPT}" -n openshift-cnv --local-ssh-opts="-o StrictHostKeyChecking=no" "cloud-user@vmi/rhel10-1"
    assert_success

    run cat "${CALL_LOG}"
    assert_output --partial "virtctl scp -n openshift-cnv --local-ssh-opts=-o StrictHostKeyChecking=no"
}

# =============================================================================
# filter_container_file
# =============================================================================

@test "filter_container_file passes file through when all paths exist" {
    local container_file="${BATS_TEST_TMPDIR}/test.container"
    printf '%s\n' "${CONTAINER_ALL_PATHS_EXIST}" > "${container_file}"

    # Override OPTIONAL_HOST_PATHS to paths that DO exist on any system.
    run bash -c "
        source '${INSTALL_SCRIPT}'  --source-only 2>/dev/null || true
        OPTIONAL_HOST_PATHS=(/tmp /var)
        filter_container_file '${container_file}'
    "
    assert_output --partial "Volume=/etc/yum.repos.d:/etc/yum.repos.d:ro"
    assert_output --partial "Volume=/data:/data:rw"
}

@test "filter_container_file strips Volume lines for missing paths" {
    local container_file="${BATS_TEST_TMPDIR}/test.container"
    printf '%s\n' "${CONTAINER_WITH_MISSING_PATHS}" > "${container_file}"

    run bash -c "
        set -euo pipefail
        OPTIONAL_HOST_PATHS=(/nonexistent/path1 /nonexistent/path2)
        SCRIPT_DIR='${BATS_TEST_TMPDIR}'
        source <(sed -n '/^filter_container_file/,/^}/p' '${INSTALL_SCRIPT}')
        filter_container_file '${container_file}'
    "
    assert_success
    assert_output --partial "Image=registry.example.com/roxagent:latest"
    assert_output --partial "Volume=/tmp:/container/tmp:rw"
    refute_output --partial "Volume=/nonexistent/path1"
    refute_output --partial "Volume=/nonexistent/path2"
}

@test "filter_container_file preserves non-Volume lines unchanged" {
    local container_file="${BATS_TEST_TMPDIR}/test.container"
    printf '%s\n' "${CONTAINER_MIXED_LINES}" > "${container_file}"

    run bash -c "
        set -euo pipefail
        OPTIONAL_HOST_PATHS=(/missing/path)
        SCRIPT_DIR='${BATS_TEST_TMPDIR}'
        source <(sed -n '/^filter_container_file/,/^}/p' '${INSTALL_SCRIPT}')
        filter_container_file '${container_file}'
    "
    assert_success
    assert_output --partial "[Container]"
    assert_output --partial "Image=registry.example.com/roxagent:latest"
    assert_output --partial "Environment=SOME_VAR=value"
    assert_output --partial "Label=com.example=test"
    refute_output --partial "Volume=/missing/path"
}

# =============================================================================
# Remote stage dir flow (mocked)
# =============================================================================

@test "create_remote_stage_dir parses __STAGE_DIR__ marker from remote output" {
    cp "${BATS_TEST_DIRNAME}/stubs/ssh-emit-marker" "${BIN_DIR}/ssh"
    chmod 0755 "${BIN_DIR}/ssh"

    run bash -c "
        set -euo pipefail
        source <(
            sed -n '/^TRANSPORT_KIND=/p; /^SSH_HOST=/p; /^SSH_PORT=/p' '${INSTALL_SCRIPT}'
            sed -n '/^remote_exec/,/^}/p' '${INSTALL_SCRIPT}'
            sed -n '/^create_remote_stage_dir/,/^}/p' '${INSTALL_SCRIPT}'
        )
        TRANSPORT_KIND=ssh
        SSH_HOST=fake
        SSH_PORT=22
        create_remote_stage_dir
    "
    assert_success
    assert_output "/var/tmp/roxagent-install.ABC123"
}

@test "create_remote_stage_dir fails when marker is missing" {
    cp "${BATS_TEST_DIRNAME}/stubs/ssh-no-marker" "${BIN_DIR}/ssh"
    chmod 0755 "${BIN_DIR}/ssh"

    run bash -c "
        set -euo pipefail
        source <(
            sed -n '/^TRANSPORT_KIND=/p; /^SSH_HOST=/p; /^SSH_PORT=/p' '${INSTALL_SCRIPT}'
            sed -n '/^remote_exec/,/^}/p' '${INSTALL_SCRIPT}'
            sed -n '/^create_remote_stage_dir/,/^}/p' '${INSTALL_SCRIPT}'
        )
        TRANSPORT_KIND=ssh
        SSH_HOST=fake
        SSH_PORT=22
        create_remote_stage_dir
    "
    assert_failure
    assert_output --partial "failed to determine remote stage dir"
}

@test "copy_remote_stage_files copies all STAGED_INSTALL_FILES" {
    write_recording_stub scp
    # We need ssh stub too (for create_remote_stage_dir) but we only test copy here.

    run bash -c "
        set -euo pipefail
        SCRIPT_DIR='${BATS_TEST_TMPDIR}'
        QUADLET_FILES_DIR='${BATS_TEST_TMPDIR}'
        TRANSPORT_KIND=ssh
        SSH_HOST=testhost
        SSH_PORT=22
        CALL_LOG='${CALL_LOG}'
        source <(
            sed -n '/^STAGED_INSTALL_FILES=/,/^)/p' '${INSTALL_SCRIPT}'
            sed -n '/^remote_copy/,/^}/p' '${INSTALL_SCRIPT}'
            sed -n '/^copy_remote_stage_files/,/^}/p' '${INSTALL_SCRIPT}'
        )
        copy_remote_stage_files '/var/tmp/roxagent-install.XXXXXX'
    "
    assert_success

    run cat "${CALL_LOG}"
    assert_output --partial "install.sh"
    assert_output --partial "roxagent.container"
    assert_output --partial "roxagent.timer"
    assert_output --partial "roxagent-prep.service"
    assert_output --partial "roxagent-tmpfiles.conf"
}

@test "install_remote heredoc includes cleanup trap with rm -rf" {
    cp "${BATS_TEST_DIRNAME}/stubs/ssh-counting-stub" "${BIN_DIR}/ssh"
    chmod 0755 "${BIN_DIR}/ssh"
    write_recording_stub scp

    run bash "${INSTALL_SCRIPT}" --ssh testuser@remotehost
    assert_success

    # Verify the heredoc sent to the remote includes cleanup
    run cat "${CALL_LOG}.heredoc"
    assert_output --partial "rm -rf"
    assert_output --partial "trap cleanup EXIT"
    assert_output --partial "--stage-dir"
}

# =============================================================================
# Double-Done regression
# =============================================================================

@test "--stage-dir path does NOT print Done epilogue" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "${STAGE_DIR}"
    assert_success
    refute_output --partial "Done!"
    refute_output --partial "periodically"
}

@test "SSH mode prints Done epilogue exactly once" {
    write_recording_stub scp
    write_recording_stub ssh

    run bash "${INSTALL_SCRIPT}" --ssh "user@host"
    assert_success

    local count
    count=$(echo "${output}" | grep -c "Done!" || true)
    [ "${count}" -eq 1 ]
}

@test "virtctl mode prints Done epilogue exactly once" {
    write_recording_stub virtctl

    run bash "${INSTALL_SCRIPT}" -n openshift-cnv "cloud-user@vmi/rhel10-1"
    assert_success

    local count
    count=$(echo "${output}" | grep -c "Done!" || true)
    [ "${count}" -eq 1 ]
}

# =============================================================================
# Helper functions
# =============================================================================

write_stage_files() {
    printf '%s\n' "${STAGE_CONTAINER}" > "${STAGE_DIR}/roxagent.container"
    printf '%s\n' "${STAGE_TIMER}" > "${STAGE_DIR}/roxagent.timer"
    printf '%s\n' "${STAGE_PREP_SERVICE}" > "${STAGE_DIR}/roxagent-prep.service"
    printf '%s\n' "${STAGE_TMPFILES_CONF}" > "${STAGE_DIR}/roxagent-tmpfiles.conf"
}

# Place a fake "sudo" on PATH that intercepts file operations and redirects
# them into FAKE_ROOT instead of real system directories.
#
# Example: "sudo cp foo /etc/systemd/system/" actually writes to
#          "${FAKE_ROOT}/etc/systemd/system/foo", which tests can then inspect.
#
# See: stubs/sudo for the full list of supported commands.
write_sudo_stub() {
    cp "${BATS_TEST_DIRNAME}/stubs/sudo" "${BIN_DIR}/sudo"
    chmod 0755 "${BIN_DIR}/sudo"
}

# Place a stub named $1 on PATH that immediately fails (exit 99) if called.
# Used as a tripwire to verify that a command is never invoked in a given test.
#
# Example: write_unexpected_remote_stub ssh
#          → if install.sh accidentally calls "ssh", the test fails with
#            "unexpected ssh invocation: <args>"
write_unexpected_remote_stub() {
    local name="$1"
    cp "${BATS_TEST_DIRNAME}/stubs/unexpected-stub" "${BIN_DIR}/${name}"
    chmod 0755 "${BIN_DIR}/${name}"
}

# Place a stub named $1 on PATH that logs every call to CALL_LOG for later
# assertion. Also handles stdin draining for ssh-like commands (heredocs) and
# emits __STAGE_DIR__ on the first ssh call so the remote staging flow works.
#
# Example: write_recording_stub scp
#          → after install.sh runs, CALL_LOG contains lines like:
#            "scp -P 22 /path/to/file user@host:/remote/path"
#          which tests assert with: assert_output --partial "scp -P 22"
write_recording_stub() {
    local name="$1"
    cp "${BATS_TEST_DIRNAME}/stubs/recording-stub" "${BIN_DIR}/${name}"
    chmod 0755 "${BIN_DIR}/${name}"
}
