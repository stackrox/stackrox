#!/usr/bin/env bats
# shellcheck disable=SC1091

load "${BATS_TEST_DIRNAME}/test_helpers.bats"

setup() {
    demo=""
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/check-changed.sh"
}

teardown() {
    if [[ -n "${demo}" && -d "${demo}" ]]; then
        rm -rf "${demo}"
    fi
}

new_git_repo() {
    demo=$(mktemp -d)
    git -C "${demo}" init -q -b master
    git -C "${demo}" config user.email "check-changed@example.com"
    git -C "${demo}" config user.name "check-changed"
}

@test "with_test_tag matches the Makefile comma form" {
    run with_test_tag ""
    assert_success
    assert_output ",test"
    run with_test_tag "release"
    assert_success
    assert_output "release,test"
}

@test "sql_integration build lines are detected" {
    local f
    f="${BATS_TEST_TMPDIR}/sql_test.go"
    printf '//go:build sql_integration\n\npackage p\n' > "${f}"
    run file_requires_sql_integration "${f}"
    assert_success

    printf '//go:build sql_integration && linux\n\npackage p\n' > "${f}"
    run file_requires_sql_integration "${f}"
    assert_success

    printf '//go:build test\n\npackage p\n' > "${f}"
    run file_requires_sql_integration "${f}"
    assert_failure

    printf '//go:build test_e2e || sql_integration\n\npackage p\n' > "${f}"
    run file_requires_sql_integration "${f}"
    assert_failure
}

@test "UNIT_TEST_IGNORE is applied and is not copied into the script" {
    UNIT_TEST_IGNORE='stackrox/rox/sensor/tests|stackrox/rox/scanner/e2etests'
    run package_is_ignored "github.com/stackrox/rox/sensor/tests/foo"
    assert_success
    run package_is_ignored "github.com/stackrox/rox/pkg/binenc"
    assert_failure
    run grep -n "sensor/tests" "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_failure
}

@test "missing BASE is a setup error" {
    run env BASE="not-a-ref-check-changed" "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_failure
    assert_equal "${status}" 3
    assert_output --partial "exit: 3"
}

@test "a docs-only diff does nothing" {
    new_git_repo
    printf 'hello\n' > "${demo}/README.md"
    git -C "${demo}" add README.md
    git -C "${demo}" commit -q -m init
    printf 'hello world\n' > "${demo}/README.md"
    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "no Go packages"
    assert_output --partial "exit: 0"
}

@test "changed packages are selected and sql and ignore are dropped from go test" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    git -C "${demo}" add go.mod
    git -C "${demo}" commit -q -m init
    mkdir -p "${demo}/pkg/a" "${demo}/pkg/sqlonly" "${demo}/pkg/skip" "${demo}/pkg/mix"
    printf 'package a\nfunc A() int { return 1 }\n' > "${demo}/pkg/a/a.go"
    printf 'package a\nimport "testing"\nfunc TestA(t *testing.T) {}\n' > "${demo}/pkg/a/a_test.go"
    printf 'package sqlonly\nfunc S() {}\n' > "${demo}/pkg/sqlonly/s.go"
    printf '//go:build sql_integration\n\npackage sqlonly\nimport "testing"\nfunc TestS(t *testing.T) {}\n' > "${demo}/pkg/sqlonly/s_test.go"
    printf 'package skip\nfunc K() {}\n' > "${demo}/pkg/skip/k.go"
    printf 'package mix\nfunc M() int { return 1 }\n' > "${demo}/pkg/mix/m.go"
    printf 'package mix\nimport "testing"\nfunc TestM(t *testing.T) {}\n' > "${demo}/pkg/mix/m_test.go"
    printf '//go:build sql_integration\n\npackage mix\nimport "testing"\nfunc TestMSQL(t *testing.T) {}\n' > "${demo}/pkg/mix/m_sql_test.go"

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 \
        UNIT_TEST_IGNORE='example.com/demo/pkg/skip' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose --print-resources-usage' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "example.com/demo/pkg/a"
    assert_output --partial "example.com/demo/pkg/mix"
    assert_output --partial "skipping sql_integration packages"
    assert_output --partial "example.com/demo/pkg/sqlonly"
    assert_output --partial "dropped UNIT_TEST_IGNORE"
    assert_output --partial "example.com/demo/pkg/skip"
    assert_output --partial "sql_integration tests, which were not run"
    assert_output --partial "DRY_RUN go test -race:"
    # The go test command lists packages to run. sqlonly is linted, not tested.
    local line saw_test=0 saw_lint=0 saw_release=0
    while IFS= read -r line; do
        if [[ "${line}" == "DRY_RUN go test -race: "* ]]; then
            saw_test=1
            [[ "${line}" == *example.com/demo/pkg/a* ]]
            [[ "${line}" == *example.com/demo/pkg/mix* ]]
            [[ "${line}" != *sqlonly* ]]
            [[ "${line}" != *pkg/skip* ]]
        fi
        if [[ "${line}" == "DRY_RUN golangci-lint: "* ]]; then
            saw_lint=1
            [[ "${line}" == *"--enable=unused"* ]]
            [[ "${line}" != *"--fix"* ]]
        fi
        if [[ "${line}" == "DRY_RUN golangci-lint (release tags): "* ]]; then
            saw_release=1
            [[ "${line}" == *"--build-tags release"* ]]
            [[ "${line}" == *"--tests=false"* ]]
            [[ "${line}" == *./pkg/sqlonly* ]]
            [[ "${line}" != *pkg/skip* ]]
        fi
    done <<< "${output}"
    [[ "${saw_test}" -eq 1 ]]
    [[ "${saw_lint}" -eq 1 ]]
    [[ "${saw_release}" -eq 1 ]]
}

@test "RACE=0 omits the race detector and says so" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    git -C "${demo}" add go.mod
    git -C "${demo}" commit -q -m init
    mkdir -p "${demo}/pkg/a"
    printf 'package a\nfunc A() int { return 1 }\n' > "${demo}/pkg/a/a.go"
    printf 'package a\nimport "testing"\nfunc TestA(t *testing.T) {}\n' > "${demo}/pkg/a/a_test.go"
    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 RACE=0 \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "RACE=0"
    assert_output --partial "DRY_RUN go test:"
    local line saw=0
    while IFS= read -r line; do
        if [[ "${line}" == "DRY_RUN go test: "* ]]; then
            saw=1
            [[ "${line}" != *"-race"* ]]
        fi
    done <<< "${output}"
    [[ "${saw}" -eq 1 ]]
}

@test "a rename contributes both paths" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    mkdir -p "${demo}/old"
    printf 'package old\nfunc O() int { return 1 }\n' > "${demo}/old/a.go"
    git -C "${demo}" add go.mod old/a.go
    git -C "${demo}" commit -q -m first
    local base
    base=$(git -C "${demo}" rev-parse HEAD)
    mkdir -p "${demo}/new"
    git -C "${demo}" mv old/a.go new/a.go
    git -C "${demo}" commit -q -m rename
    run env CHECK_CHANGED_ROOT="${demo}" BASE="${base}" DRY_RUN=1 \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "old/a.go"
    assert_output --partial "new/a.go"
    assert_output --partial "example.com/demo/new"
}

@test "a nested module is tested in its own directory and not with the root linter" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    mkdir -p "${demo}/mod/p"
    printf 'module example.com/nested\n\ngo 1.22\n' > "${demo}/mod/go.mod"
    git -C "${demo}" add go.mod mod/go.mod
    git -C "${demo}" commit -q -m init
    printf 'package p\nfunc P() int { return 1 }\n' > "${demo}/mod/p/p.go"
    printf 'package p\nimport "testing"\nfunc TestP(t *testing.T) {}\n' > "${demo}/mod/p/p_test.go"
    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS=mod \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "example.com/nested/p"
    assert_output --partial "DRY_RUN go test -race (nested):"
    assert_output --partial "golangci-lint was not run in nested module mod"
    local line
    while IFS= read -r line; do
        if [[ "${line}" == "DRY_RUN go test -race (nested): "* ]]; then
            [[ "${line}" == *"(cd mod && CGO_ENABLED=1 go test "* ]]
            [[ "${line}" == *"./p"* ]]
        fi
        [[ "${line}" != "DRY_RUN golangci-lint: "* ]]
    done <<< "${output}"
}

@test "Postgres runs sql_integration packages and PG=require fails when it is down" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    git -C "${demo}" add go.mod
    git -C "${demo}" commit -q -m init
    mkdir -p "${demo}/pkg/sqlonly" "${demo}/migrator/m"
    printf 'package sqlonly\nfunc S() {}\n' > "${demo}/pkg/sqlonly/s.go"
    printf '//go:build sql_integration\n\npackage sqlonly\nimport "testing"\nfunc TestS(t *testing.T) {}\n' > "${demo}/pkg/sqlonly/s_test.go"
    printf 'package m\nfunc M() {}\n' > "${demo}/migrator/m/m.go"
    printf '//go:build sql_integration\n\npackage m\nimport "testing"\nfunc TestM(t *testing.T) {}\n' > "${demo}/migrator/m/m_test.go"

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 PG_READY=1 \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "Postgres is reachable"
    assert_output --partial "GOTAGS=,test,sql_integration"
    assert_output --partial "example.com/demo/pkg/sqlonly"
    assert_output --partial "example.com/demo/migrator/m"
    assert_output --partial -- "-p 1"

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 PG_READY=0 PG=require \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_failure
    assert_equal "${status}" 1
    assert_output --partial "skipping sql_integration packages"
    assert_output --partial "exit: 1"
}

@test "DEPS modes add importers and OUTPUT=json records the run" {
    new_git_repo
    printf 'module example.com/demo\n\ngo 1.22\n' > "${demo}/go.mod"
    mkdir -p "${demo}/pkg/a" "${demo}/pkg/b" "${demo}/pkg/c"
    printf 'package a\nfunc A() int { return 1 }\n' > "${demo}/pkg/a/a.go"
    printf 'package b\nimport "example.com/demo/pkg/a"\nfunc B() int { return a.A() }\n' > "${demo}/pkg/b/b.go"
    printf 'package c\nimport "example.com/demo/pkg/b"\nfunc C() int { return b.B() }\n' > "${demo}/pkg/c/c.go"
    git -C "${demo}" add go.mod pkg
    git -C "${demo}" commit -q -m init
    printf 'package a\nfunc A() int { return 2 }\n' > "${demo}/pkg/a/a.go"

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 DEPS=direct OUTPUT=json \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "untested importers: 1"
    assert_output --partial "wrote test-output/check-changed.json"
    python3 - "${demo}/test-output/check-changed.json" << 'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
assert doc["exit"] == 0
assert doc["deps"] == "direct"
assert doc["untested_dependents"] == 1
PY

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 DEPS=rdeps \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "DEPS=rdeps added:"
    assert_output --partial "example.com/demo/pkg/b"
    local line
    while IFS= read -r line; do
        if [[ "${line}" == "DRY_RUN go test -race: "* ]]; then
            [[ "${line}" == *example.com/demo/pkg/a* ]]
            [[ "${line}" == *example.com/demo/pkg/b* ]]
            [[ "${line}" != *example.com/demo/pkg/c* ]]
        fi
    done <<< "${output}"

    run env CHECK_CHANGED_ROOT="${demo}" BASE=HEAD DRY_RUN=1 DEPS=rdeps-all \
        UNIT_TEST_IGNORE='a^' RELEASE_GOTAGS=release \
        GOLANGCILINT_BIN=/bin/true GOLANGCILINT_FLAGS='--verbose' \
        NESTED_GO_MODULE_DIRS= \
        "${BATS_TEST_DIRNAME}/check-changed.sh"
    assert_success
    assert_output --partial "example.com/demo/pkg/c"
}

@test "check-changed target passes Makefile variables" {
    run grep -A 12 "^check-changed:" "${BATS_TEST_DIRNAME}/../Makefile"
    assert_success
    # shellcheck disable=SC2016 # the Makefile text contains literal make variables
    assert_output --partial 'UNIT_TEST_IGNORE=$(UNIT_TEST_IGNORE)'
    # shellcheck disable=SC2016
    assert_output --partial 'RELEASE_GOTAGS="$(RELEASE_GOTAGS)"'
    assert_output --partial 'scripts/check-changed.sh'
}
