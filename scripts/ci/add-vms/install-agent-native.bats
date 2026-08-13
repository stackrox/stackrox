#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/install-agent-native.sh"
}

@test "roxagent-prep.service prepares the curated roxroot tree without copying RPM DB" {
    run cat "$SYSTEMD_DIR/roxagent-prep.service"

    assert_success
    assert_output --partial "ExecStartPre=/bin/rm -rf /tmp/roxroot"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/etc/pki/entitlement"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/lib/dnf"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/lib/rpm"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/usr/lib/sysimage/libdnf5"
    assert_output --partial "ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/cache/dnf"
    refute_output --partial "/run/lock/roxagent"
    assert_output --partial "ExecStart=/bin/true"
    refute_output --partial "roxagent-rpm"
    refute_output --partial "cp -a /var/lib/rpm"
}

@test "create_native_serve_file mounts live RPM and DNF inputs read-only into roxroot" {
    run create_native_serve_file

    assert_success
    assert_output --partial "Requires=roxagent-prep.service"
    assert_output --partial "Type=simple"
    assert_output --partial "Restart=on-failure"
    assert_output --partial "BindReadOnlyPaths=-/etc/os-release:/tmp/roxroot/etc/os-release"
    assert_output --partial "BindReadOnlyPaths=-/var/lib/rpm:/tmp/roxroot/var/lib/rpm"
    assert_output --partial "BindReadOnlyPaths=-/usr/lib/sysimage/libdnf5:/tmp/roxroot/usr/lib/sysimage/libdnf5"
    assert_output --partial "ExecStart=/usr/local/bin/roxagent serve --port 818 --host-path /tmp/roxroot"
    assert_output --partial "WantedBy=multi-user.target"
    refute_output --partial "BindPaths=/tmp/roxagent-rpm"
}
