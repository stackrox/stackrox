#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/install-agent-native.sh"
}

@test "roxagent-prep.service prepares the curated roxroot tree" {
    run cat "$SYSTEMD_DIR/roxagent-prep.service"

    assert_success
    assert_output --partial "ExecStartPre=/bin/rm -rf /tmp/roxroot"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/etc/pki/entitlement"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/lib/dnf"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/cache/dnf"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /run/lock/roxagent"
    assert_output --partial "ExecStart=/bin/rm -rf /tmp/roxagent-rpm"
    assert_output --partial "ExecStart=/bin/cp -a /var/lib/rpm /tmp/roxagent-rpm"
}

@test "create_native_serve_file generates long-running serve unit" {
    run create_native_serve_file

    assert_success
    assert_output --partial "Requires=roxagent-prep.service"
    assert_output --partial "Type=simple"
    assert_output --partial "Restart=on-failure"
    assert_output --partial "BindPaths=/tmp/roxagent-rpm:/tmp/roxroot/var/lib/rpm"
    assert_output --partial "BindReadOnlyPaths=-/etc/os-release:/tmp/roxroot/etc/os-release"
    assert_output --partial "ExecStart=/usr/local/bin/roxagent serve --port 818 --host-path /tmp/roxroot"
    assert_output --partial "WantedBy=multi-user.target"
}
