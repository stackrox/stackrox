#!/usr/bin/env bats

setup() {
    root="$(cd "$BATS_TEST_DIRNAME/../../.." && pwd)"
    export install="$BATS_TEST_TMPDIR/stackrox"
    mkdir -p "$install/bin"
    sed "s|/stackrox/|$install/|g" "$root/image/rhel/static-bin/start-central.sh" >"$install/start-central.sh"
    cp "$root/image/rhel/static-bin/debug" "$install/"
    cat >"$install/bin/migrator" <<'SCRIPT'
#!/usr/bin/env bash
exit "${MIGRATOR_STATUS:-0}"
SCRIPT
    cat >"$install/central" <<'SCRIPT'
#!/usr/bin/env bash
touch "$install/central-started"
SCRIPT
    chmod +x "$install/bin/migrator" "$install/central"
}

@test "Central starts only after successful migration" {
    run bash "$install/start-central.sh"
    [ "$status" -eq 0 ]
    [ -f "$install/central-started" ]
}

@test "migration rejection prevents Central startup" {
    export MIGRATOR_STATUS=1
    run bash "$install/start-central.sh"
    [ "$status" -eq 1 ]
    [ ! -f "$install/central-started" ]
}

@test "SIGILL retains the CPU diagnostic" {
    export MIGRATOR_STATUS=132
    run bash "$install/start-central.sh"
    [ "$status" -eq 132 ]
    [[ "$output" == *"Illegal Instruction"* ]]
    [ ! -f "$install/central-started" ]
}
