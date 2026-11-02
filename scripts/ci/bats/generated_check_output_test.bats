#!/usr/bin/env bats

setup() {
    export ROOT="$BATS_TEST_TMPDIR/repo"
    export CHECK_STATE="$BATS_TEST_TMPDIR/state"
    export GENERATED_HINT='Run make go-generated-srcs and commit the result.'
    export GITHUB_ACTION=test
    mkdir -p "$ROOT" "$CHECK_STATE"
    source "$BATS_TEST_DIRNAME/../../lib.sh"
    source "$BATS_TEST_DIRNAME/../generated-check-output.sh"
    cd "$ROOT" || return
}

@test "early diagnostic survives trailing output and group closes before annotation" {
    run generated_command bash -c 'echo "pkg/example.go:3: running go: exit status 1"; for ((i=0;i<2000;i++)); do echo noise; done; exit 7'
    [ "$status" -eq 7 ]
    [[ "$output" == *'::error::pkg/example.go:3: running go: exit status 1'* ]]
    [[ "$output" == *'::endgroup::'*'::error::Command failed (exit 7)'* ]]
}

@test "successful command emits no errors" {
    run generated_command printf 'hello\n'
    [ "$status" -eq 0 ]
    [[ "$output" != *'::error'* ]]
}

@test "outside paths are not linked and duplicate diagnostics are removed" {
    run generated_command bash -c 'printf "/outside.go:2: error\n/outside.go:2: error\n"; exit 1'
    [ "$status" -eq 1 ]
    [[ "$output" != *'file=/outside.go'* ]]
    [ "$(grep -c '^::error::/outside.go' <<< "$output")" -eq 1 ]
}

@test "annotation escapes workflow command data" {
    run generated_annotation 'a,b.go' 2 $'bad%\nmessage'
    [ "$status" -eq 0 ]
    [[ "$output" == *'file=a%2Cb.go,line=2::bad%25%0Amessage'* ]]
}

@test "tracked drift links checked out line" {
    git init -q
    git -c core.hooksPath=/dev/null -c commit.gpgsign=false -c user.name=test -c user.email=test@example.org commit --allow-empty -qm initial
    printf 'old\n' > file.go
    git add file.go
    git -c core.hooksPath=/dev/null -c commit.gpgsign=false -c user.name=test -c user.email=test@example.org commit -qm file
    printf 'new\n' > file.go
    run generated_diff
    [ "$status" -eq 1 ]
    [[ "$output" == *'file=file.go,line=1::Generated output differs'* ]]
}

@test "child check stops on failure and subsequent checks can run" {
    function failing_check() {
        generated_command bash -c 'exit 9'
        touch "$CHECK_STATE/should-not-exist"
    }
    export -f failing_check
    export SHELLOPTS
    run bash -e -c 'generated_command bash -c failing_check || status=$?; generated_command true; exit "$status"'
    [ "$status" -eq 9 ]
    [ ! -e "$CHECK_STATE/should-not-exist" ]
}

@test "plain text mode does not emit workflow commands" {
    unset GITHUB_ACTION
    run generated_command bash -c 'exit 1'
    [ "$status" -eq 1 ]
    [[ "$output" == *'ERROR:'* ]]
    [[ "$output" != *'::error'* ]]
}

@test "nested generator relative paths are not linked even when a root file exists" {
    mkdir -p gen central/example/gen
    touch gen/main.go central/example/gen/main.go
    run generated_command bash -c 'cd central/example; echo "gen/main.go:2:15: undefined: missing"; exit 1'
    [ "$status" -eq 1 ]
    [[ "$output" == *'::error::gen/main.go:2:15: undefined: missing'* ]]
    [[ "$output" != *'::error file='* ]]
}

@test "absolute checkout diagnostics retain file links" {
    # shellcheck disable=SC2016 # Expanded by the child shell.
    run generated_command bash -c 'echo "$ROOT/central/example/gen/main.go:2:15: undefined: missing"; exit 1'
    [ "$status" -eq 1 ]
    [[ "$output" == *'::error file=central/example/gen/main.go,line=2::'* ]]
}
