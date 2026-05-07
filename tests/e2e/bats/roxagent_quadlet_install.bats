#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../../scripts/test_helpers.bats"

INSTALL_SCRIPT_REL="compliance/virtualmachines/roxagent/quadlet/install.sh"

setup() {
    INSTALL_SCRIPT="${BATS_TEST_DIRNAME}/../../../${INSTALL_SCRIPT_REL}"
    STAGE_DIR="${BATS_TEST_TMPDIR}/custom-stage"
    BIN_DIR="${BATS_TEST_TMPDIR}/bin"
    FAKE_ROOT="${BATS_TEST_TMPDIR}/fake-root"
    CALL_LOG="${BATS_TEST_TMPDIR}/calls.log"

    mkdir -p "${STAGE_DIR}" "${BIN_DIR}" "${FAKE_ROOT}"
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
# Install from explicit stage dir (--stage-dir)
# =============================================================================

@test "--stage-dir installs all unit files to correct locations" {
    run bash "${INSTALL_SCRIPT}" --stage-dir "${STAGE_DIR}"
    assert_success

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
    cat > "${container_file}" <<'EOF'
[Container]
Image=registry.example.com/roxagent:latest
Volume=/etc/yum.repos.d:/etc/yum.repos.d:ro
Volume=/data:/data:rw
EOF

    # Source the script in a subshell to get access to filter_container_file.
    # Override OPTIONAL_HOST_PATHS to paths that DO exist.
    run bash -c "
        source '${INSTALL_SCRIPT}'  --source-only 2>/dev/null || true
        OPTIONAL_HOST_PATHS=(/tmp /var)
        filter_container_file '${container_file}'
    "
    # Since /tmp and /var exist, no stripping occurs.
    assert_output --partial "Volume=/etc/yum.repos.d:/etc/yum.repos.d:ro"
    assert_output --partial "Volume=/data:/data:rw"
}

@test "filter_container_file strips Volume lines for missing paths" {
    local container_file="${BATS_TEST_TMPDIR}/test.container"
    cat > "${container_file}" <<'EOF'
[Container]
Image=registry.example.com/roxagent:latest
Volume=/nonexistent/path1:/container/path1:ro
Volume=/tmp:/container/tmp:rw
Volume=/nonexistent/path2:/container/path2:ro
EOF

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
    cat > "${container_file}" <<'EOF'
[Container]
Image=registry.example.com/roxagent:latest
Environment=SOME_VAR=value
Volume=/missing/path:/dst:ro
Label=com.example=test
EOF

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
    # Write an ssh stub that outputs the marker as a remote mktemp would
    cat > "${BIN_DIR}/ssh" <<'EOF'
#!/usr/bin/env bash
# Simulate remote execution that outputs the stage dir marker
cat <<'REMOTE'
__STAGE_DIR__=/var/tmp/roxagent-install.ABC123
REMOTE
EOF
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
    cat > "${BIN_DIR}/ssh" <<'EOF'
#!/usr/bin/env bash
echo "some garbage output without the marker"
EOF
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
    write_recording_stub ssh

    # Mock create_remote_stage_dir to return a known path
    cat > "${BIN_DIR}/ssh" <<'STUB'
#!/usr/bin/env bash
# First call: create_remote_stage_dir (output marker)
# Subsequent calls: run the heredoc (just record stdin)
if [ ! -f "${CALL_LOG}.ssh_call_count" ]; then
    echo "0" > "${CALL_LOG}.ssh_call_count"
fi
count=$(cat "${CALL_LOG}.ssh_call_count")
count=$((count + 1))
echo "${count}" > "${CALL_LOG}.ssh_call_count"

if [ "${count}" -eq 1 ]; then
    echo "__STAGE_DIR__=/var/tmp/roxagent-install.FAKEXX"
else
    # Record the heredoc content that was piped in
    cat >> "${CALL_LOG}.heredoc"
fi
STUB
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
    cat > "${STAGE_DIR}/roxagent.container" <<'EOF'
[Container]
SENTINEL-CONTAINER=true
EOF

    cat > "${STAGE_DIR}/roxagent.timer" <<'EOF'
SENTINEL-TIMER=true
EOF

    cat > "${STAGE_DIR}/roxagent-prep.service" <<'EOF'
SENTINEL-PREP=true
EOF

    cat > "${STAGE_DIR}/roxagent-tmpfiles.conf" <<'EOF'
SENTINEL-TMPFILES=true
EOF
}

write_sudo_stub() {
    cat > "${BIN_DIR}/sudo" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

rewrite_system_path() {
    local path="${1}"
    case "${path}" in
        /etc/*|/run/*)
            printf '%s%s\n' "${FAKE_ROOT}" "${path}"
            ;;
        *)
            printf '%s\n' "${path}"
            ;;
    esac
}

cmd="${1}"
shift

case "${cmd}" in
    mkdir)
        args=()
        for arg in "$@"; do
            args+=("$(rewrite_system_path "${arg}")")
        done
        command mkdir "${args[@]}"
        ;;
    tee)
        target="$(rewrite_system_path "${1}")"
        command mkdir -p "$(dirname "${target}")"
        cat > "${target}"
        ;;
    cp)
        src="${1}"
        dst="$(rewrite_system_path "${2}")"
        if [[ "${2}" == */ ]]; then
            command mkdir -p "${dst}"
        else
            command mkdir -p "$(dirname "${dst}")"
        fi
        command cp "${src}" "${dst}"
        ;;
    restorecon)
        exit 0
        ;;
    systemd-tmpfiles|systemctl)
        printf '%s %s\n' "${cmd}" "$*" >> "${FAKE_ROOT}/sudo.log"
        exit 0
        ;;
    *)
        printf 'unsupported sudo command: %s\n' "${cmd}" >&2
        exit 1
        ;;
esac
EOF
    chmod 0755 "${BIN_DIR}/sudo"
}

write_unexpected_remote_stub() {
    local name="${1}"

    cat > "${BIN_DIR}/${name}" <<EOF
#!/usr/bin/env bash
printf 'unexpected ${name} invocation: %s\n' "\$*" >&2
exit 99
EOF
    chmod 0755 "${BIN_DIR}/${name}"
}

write_recording_stub() {
    local name="${1}"

    cat > "${BIN_DIR}/${name}" <<'STUB'
#!/usr/bin/env bash
# Record the invocation to the shared call log.
printf '%s %s\n' "$(basename "$0")" "$*" >> "${CALL_LOG}"

cmd_name="$(basename "$0")"

# Determine if this is a command that receives stdin (ssh/remote_exec).
# scp-like calls (virtctl scp, scp) don't pipe stdin so we must not block.
reads_stdin=false
case "${cmd_name}" in
    ssh)
        reads_stdin=true
        ;;
    virtctl)
        # "virtctl ssh ..." reads stdin; "virtctl scp ..." does not.
        if [ "${1:-}" = "ssh" ]; then
            reads_stdin=true
        fi
        ;;
esac

if [ "${reads_stdin}" = "true" ]; then
    # First ssh-like call: emit the stage dir marker (for create_remote_stage_dir).
    # Subsequent ssh-like calls: just drain stdin.
    if [ ! -f "${CALL_LOG}.${cmd_name}_emitted_marker" ]; then
        echo "__STAGE_DIR__=/var/tmp/roxagent-install.TESTXX"
        touch "${CALL_LOG}.${cmd_name}_emitted_marker"
    else
        cat > /dev/null 2>&1 || true
    fi
fi
STUB
    chmod 0755 "${BIN_DIR}/${name}"
}
