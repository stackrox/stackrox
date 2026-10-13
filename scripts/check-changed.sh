#!/usr/bin/env bash
# Test and lint the Go packages touched by the current diff.
#
# Exit codes:
#   0  every step that ran passed
#   1  a test or a lint failed
#   2  codegen is needed (reserved; this MVP does not detect codegen)
#   3  setup error (missing ref, go list failure, linter missing)
#
# A test or lint failure stays exit 1 even if a later phase also wants exit 2.
# Setup errors return immediately.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck disable=SC2034 # reserved for the codegen phase
EXIT_OK=0
# shellcheck disable=SC2034
EXIT_FAIL=1
# shellcheck disable=SC2034
EXIT_CODEGEN=2
# shellcheck disable=SC2034
EXIT_SETUP=3

final_exit=0
merge_base_sha=
root=
nested_dirs=()
lint_flags=()
summary_steps=()
summary_scopes=()
summary_results=()
summary_times=()
summary_reruns=()

repo_root() {
    if [[ -n "${CHECK_CHANGED_ROOT:-}" ]]; then
        printf '%s\n' "${CHECK_CHANGED_ROOT}"
    else
        printf '%s\n' "${ROOT}"
    fi
}

setup_error() {
    printf 'ERROR: %s\n' "$*" >&2
    printf 'exit: 3\n' >&2
    exit 3
}

epoch_ns() {
    date +%s%N
}

format_elapsed() {
    local start=$1
    local end
    end=$(epoch_ns)
    awk -v s="${start}" -v e="${end}" 'BEGIN { printf "%.1fs", (e - s) / 1000000000 }'
}

summary_add() {
    summary_steps+=("$1")
    summary_scopes+=("$2")
    summary_results+=("$3")
    summary_times+=("$4")
    summary_reruns+=("$5")
}

print_summary() {
    local i
    printf '\n=== summary ===\n'
    printf '%-34s %-28s %-8s %8s\n' "step" "scope" "result" "time"
    for i in "${!summary_steps[@]}"; do
        printf '%-34s %-28s %-8s %8s\n' \
            "${summary_steps[$i]}" "${summary_scopes[$i]}" \
            "${summary_results[$i]}" "${summary_times[$i]}"
    done
    printf '\n'
    for i in "${!summary_steps[@]}"; do
        if [[ -n "${summary_reruns[$i]}" ]]; then
            printf 'rerun (%s):\n  %s\n' "${summary_steps[$i]}" "${summary_reruns[$i]}"
        fi
    done
    printf '\nexit: %s\n' "${final_exit}"
    if [[ "${final_exit}" -ne 0 ]]; then
        printf 'GNU make exits 2 when a recipe fails. The exit line above is the script status.\n'
    fi
}

# with_test_tag matches the Makefile: GOTAGS=$(GOTAGS),test
# An empty GOTAGS becomes ",test". scripts/go-test.sh turns commas into spaces.
with_test_tag() {
    printf '%s,test' "${1-}"
}

package_scope() {
    local n=$1
    if [[ "${n}" -eq 1 ]]; then
        printf '1 package'
    else
        printf '%s packages' "${n}"
    fi
}

normalize_ignore() {
    UNIT_TEST_IGNORE=${UNIT_TEST_IGNORE-}
    UNIT_TEST_IGNORE=${UNIT_TEST_IGNORE#\"}
    UNIT_TEST_IGNORE=${UNIT_TEST_IGNORE%\"}
}

load_make_config() {
    if [[ -n "${UNIT_TEST_IGNORE+x}" && -n "${GOLANGCILINT_BIN:-}" && -n "${RELEASE_GOTAGS+x}" && -n "${NESTED_GO_MODULE_DIRS+x}" && -n "${GOLANGCILINT_FLAGS+x}" ]]; then
        normalize_ignore
        return 0
    fi
    local line key val
    while IFS= read -r line; do
        key=${line%%=*}
        val=${line#*=}
        case "${key}" in
            UNIT_TEST_IGNORE)
                [[ -n "${UNIT_TEST_IGNORE+x}" ]] || UNIT_TEST_IGNORE=${val}
                ;;
            GOTAGS)
                [[ -n "${GOTAGS+x}" ]] || GOTAGS=${val}
                ;;
            RELEASE_GOTAGS)
                [[ -n "${RELEASE_GOTAGS+x}" ]] || RELEASE_GOTAGS=${val}
                ;;
            NESTED_GO_MODULE_DIRS)
                [[ -n "${NESTED_GO_MODULE_DIRS+x}" ]] || NESTED_GO_MODULE_DIRS=${val}
                ;;
            GOLANGCILINT_FLAGS)
                [[ -n "${GOLANGCILINT_FLAGS+x}" ]] || GOLANGCILINT_FLAGS=${val}
                ;;
            *) ;;
        esac
    done < <(make --quiet --no-print-directory -C "${ROOT}" print-check-changed-config)
    normalize_ignore
    if [[ -z "${RELEASE_GOTAGS:-}" ]]; then
        setup_error "RELEASE_GOTAGS is empty. It is defined in make/env.mk."
    fi
}

load_nested_dirs() {
    nested_dirs=()
    if [[ -n "${NESTED_GO_MODULE_DIRS:-}" ]]; then
        read -r -a nested_dirs <<< "${NESTED_GO_MODULE_DIRS}"
    fi
}

load_lint_flags() {
    lint_flags=()
    if [[ -n "${GOLANGCILINT_FLAGS:-}" ]]; then
        read -r -a lint_flags <<< "${GOLANGCILINT_FLAGS}"
    fi
}

ensure_linter() {
    if [[ -x "${GOLANGCILINT_BIN:-}" ]]; then
        return 0
    fi
    local built
    if ! built=$(make --quiet --no-print-directory -C "${ROOT}" which-golangci-lint); then
        setup_error "failed to build golangci-lint"
    fi
    GOLANGCILINT_BIN=${built//$'\n'/}
    if [[ ! -x "${GOLANGCILINT_BIN}" ]]; then
        setup_error "golangci-lint is not executable at ${GOLANGCILINT_BIN}"
    fi
}

# file_requires_sql_integration is true when the file's build line requires
# the sql_integration tag (the form make go-postgres-unit-tests searches for).
# An OR constraint is not sql-only: another tag could select the file.
file_requires_sql_integration() {
    local line re
    line=$(grep -m1 -E '^//go:build ' "$1" || true)
    re='^//go:build[[:space:]]+sql_integration([[:space:]]+&&[[:space:]]+.*)?[[:space:]]*$'
    [[ "${line}" =~ ${re} ]]
}

package_is_ignored() {
    local pkg=$1
    [[ -n "${UNIT_TEST_IGNORE:-}" ]] || return 1
    if grep -Eq -- "${UNIT_TEST_IGNORE}" <<< "${pkg}"; then
        return 0
    fi
    return 1
}

file_nested_module() {
    local f=$1
    local d best=""
    if [[ ${#nested_dirs[@]} -eq 0 ]]; then
        return 1
    fi
    for d in "${nested_dirs[@]}"; do
        case "${f}" in
            "${d}" | "${d}"/*)
                if [[ ${#d} -gt ${#best} ]]; then
                    best=${d}
                fi
                ;;
        esac
    done
    [[ -n "${best}" ]] || return 1
    printf '%s\n' "${best}"
}

resolve_base() {
    local ref=${BASE:-origin/master}
    # Reject option-like refs. git rev-parse --end-of-options echoes that flag,
    # so a verified commit SHA is used for every later git call.
    if [[ -z "${ref}" || "${ref}" == -* ]]; then
        setup_error "BASE ref '${ref}' is not a valid git revision."
    fi
    local tip_sha
    if ! tip_sha=$(git -C "${root}" rev-parse --verify --quiet "${ref}^{commit}"); then
        setup_error "BASE ref '${ref}' was not found. Fetch it, or pass BASE=<ref>."
    fi
    local tip_ct now age_h tip_date mb_date
    tip_ct=$(git -C "${root}" log -1 --format=%ct "${tip_sha}")
    now=$(date +%s)
    age_h=$(((now - tip_ct) / 3600))
    merge_base_sha=$(git -C "${root}" merge-base "${tip_sha}" HEAD)
    tip_date=$(git -C "${root}" log -1 --format=%ci "${tip_sha}")
    mb_date=$(git -C "${root}" log -1 --format=%ci "${merge_base_sha}")
    printf 'check-changed\n'
    printf 'base ref: %s %s (%s, %s hours old, not fetched)\n' \
        "${ref}" "${tip_sha}" "${tip_date}" "${age_h}"
    printf 'merge-base: %s (%s)\n' "${merge_base_sha}" "${mb_date}"
    if [[ "${ref}" == "origin/master" && "${age_h}" -gt 72 ]]; then
        printf 'WARN: origin/master is %s hours old in this checkout and was not fetched. The diff may be wrong.\n' "${age_h}"
    fi
}

parse_name_status() {
    local status p2 p3
    while IFS=$'\t' read -r status p2 p3; do
        [[ -z "${status}" ]] && continue
        case "${status}" in
            R* | C*)
                printf '%s\n' "${p2}"
                printf '%s\n' "${p3}"
                ;;
            *)
                printf '%s\n' "${p2}"
                ;;
        esac
    done
}

collect_changed_files() {
    local base_sha=$1
    {
        git -C "${root}" diff --name-status --find-renames "${base_sha}" HEAD | parse_name_status
        git -C "${root}" diff --name-status --find-renames --cached | parse_name_status
        git -C "${root}" diff --name-status --find-renames | parse_name_status
        git -C "${root}" ls-files --others --exclude-standard
    } | sed '/^$/d' | sort -u
}

dir_has_go() {
    local found
    found=$(find "$1" -maxdepth 1 -name '*.go' -print -quit 2>/dev/null || true)
    [[ -n "${found}" ]]
}

map_go_files() {
    local module_dir=$1
    shift
    local -a rel_files=()
    local f rel parent
    for f in "$@"; do
        [[ "${f}" == *.go ]] || continue
        case "${f}" in
            vendor/* | */vendor/*) continue ;;
        esac
        if [[ "${module_dir}" == "." ]]; then
            rel=${f}
            parent="${root}/$(dirname "${f}")"
        else
            rel=${f#"${module_dir}/"}
            parent="${root}/${module_dir}/$(dirname "${rel}")"
        fi
        # A delete or rename leaves the old path in the diff. go list fails
        # (and retries) when that directory no longer contains Go files.
        [[ -d "${parent}" ]] || continue
        dir_has_go "${parent}" || continue
        rel_files+=("${rel}")
    done
    if [[ ${#rel_files[@]} -eq 0 ]]; then
        return 0
    fi
    local out
    if ! out=$(printf '%s\n' "${rel_files[@]}" | (cd "${root}/${module_dir}" && "${SCRIPT_DIR}/go-files-to-packages.sh")); then
        return 1
    fi
    if [[ -n "${out}" ]]; then
        printf '%s\n' "${out}"
    fi
}

import_to_dir() {
    local module_dir=$1
    local mod=$2
    local import_path=$3
    local rel
    if [[ "${import_path}" == "${mod}" ]]; then
        if [[ "${module_dir}" == "." ]]; then
            printf '%s\n' "${root}"
        else
            printf '%s\n' "${root}/${module_dir}"
        fi
        return 0
    fi
    rel=${import_path#"${mod}/"}
    if [[ "${module_dir}" == "." ]]; then
        printf '%s\n' "${root}/${rel}"
    else
        printf '%s\n' "${root}/${module_dir}/${rel}"
    fi
}

import_to_rel() {
    local mod=$1
    local import_path=$2
    if [[ "${import_path}" == "${mod}" ]]; then
        printf '.\n'
    else
        printf './%s\n' "${import_path#"${mod}/"}"
    fi
}

# partition_tests sets the nameref arrays: kept packages, sql-only skips, mixed warnings.
partition_tests() {
    local module_dir=$1
    local mod=$2
    local -n _kept=$3
    local -n _sql_only=$4
    local -n _sql_mixed=$5
    shift 5
    local pkg dir f all_sql has_sql
    local -a tfiles
    for pkg in "$@"; do
        if package_is_ignored "${pkg}"; then
            ignored_packages+=("${pkg}")
            continue
        fi
        dir=$(import_to_dir "${module_dir}" "${mod}" "${pkg}")
        tfiles=()
        while IFS= read -r -d '' f; do
            tfiles+=("${f}")
        done < <(find "${dir}" -maxdepth 1 -name '*_test.go' -print0 2>/dev/null || true)
        if [[ ${#tfiles[@]} -eq 0 ]]; then
            _kept+=("${pkg}")
            continue
        fi
        all_sql=1
        has_sql=0
        for f in "${tfiles[@]}"; do
            if file_requires_sql_integration "${f}"; then
                has_sql=1
            else
                all_sql=0
            fi
        done
        if [[ "${has_sql}" -eq 1 && "${all_sql}" -eq 1 ]]; then
            _sql_only+=("${pkg}")
        else
            _kept+=("${pkg}")
            if [[ "${has_sql}" -eq 1 ]]; then
                _sql_mixed+=("${pkg}")
            fi
        fi
    done
}

race_enabled() {
    [[ "${RACE:-1}" != "0" ]]
}

run_recorded() {
    local name=$1
    local scope=$2
    local rerun=$3
    shift 3
    if [[ "${DRY_RUN:-0}" == "1" ]]; then
        printf 'DRY_RUN %s: %s\n' "${name}" "${rerun}"
        summary_add "${name}" "${scope}" "dry-run" "-" "${rerun}"
        return 0
    fi
    local start status elapsed
    start=$(epoch_ns)
    set +e
    "$@"
    status=$?
    set -e
    elapsed=$(format_elapsed "${start}")
    if [[ "${status}" -eq 0 ]]; then
        summary_add "${name}" "${scope}" "pass" "${elapsed}" "${rerun}"
    else
        summary_add "${name}" "${scope}" "fail" "${elapsed}" "${rerun}"
        final_exit=1
    fi
}

go_test_root() {
    local -a pkgs=("$@")
    local tags rerun step
    local -a args=(-timeout 25m -v)
    tags=$(with_test_tag "${GOTAGS-}")
    step="go test"
    if race_enabled; then
        args=(-timeout 25m -race -v)
        step="go test -race"
    fi
    rerun="CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=${tags} scripts/go-test.sh ${args[*]} ${pkgs[*]}"
    run_recorded "${step}" "$(package_scope "${#pkgs[@]}")" "${rerun}" \
        env CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS="${MUTEX_WATCHDOG_TIMEOUT_SECS:-30}" GOTAGS="${tags}" \
        "${SCRIPT_DIR}/go-test.sh" "${args[@]}" "${pkgs[@]}"
}

run_go_test_in() {
    local dir=$1
    shift
    (
        cd "${dir}" || exit 3
        CGO_ENABLED=1 go test "$@"
    )
}

go_test_nested() {
    local module_dir=$1
    shift
    local -a pkgs=("$@")
    local rels=() pkg mod rel rerun step
    local -a args=(-buildvcs=false -timeout 25m -v)
    mod=$(cd "${root}/${module_dir}" && go list -m)
    for pkg in "${pkgs[@]}"; do
        rel=$(import_to_rel "${mod}" "${pkg}")
        rels+=("${rel}")
    done
    step="go test (nested)"
    if race_enabled; then
        args=(-buildvcs=false -timeout 25m -race -v)
        step="go test -race (nested)"
    fi
    rerun="(cd ${module_dir} && CGO_ENABLED=1 go test ${args[*]} ${rels[*]})"
    run_recorded "${step}" "${module_dir} (${#pkgs[@]})" "${rerun}" \
        run_go_test_in "${root}/${module_dir}" "${args[@]}" "${rels[@]}"
}

lint_packages() {
    local mode=$1
    shift
    local -a rels=("$@")
    local -a cmd=("${GOLANGCILINT_BIN}" run)
    local name rerun
    if [[ ${#lint_flags[@]} -gt 0 ]]; then
        cmd+=("${lint_flags[@]}")
    fi
    if [[ "${FIX:-0}" == "1" ]]; then
        cmd+=(--fix)
    fi
    if [[ "${mode}" == "release" ]]; then
        name="golangci-lint (release tags)"
        # v2 appends this to run.build-tags from .golangci.yml. Same flag as the Makefile.
        cmd+=(--build-tags "${RELEASE_GOTAGS//,/ }" --tests=false)
    else
        name="golangci-lint"
        cmd+=(--enable=unused)
    fi
    cmd+=("${rels[@]}")
    rerun="${cmd[*]}"
    run_recorded "${name}" "$(package_scope "${#rels[@]}")" "${rerun}" "${cmd[@]}"
}

finish_ok_notes() {
    printf 'NOTE: only packages that contain changed files were tested. Importers were not.\n'
    printf 'NOTE: not run: integration, test_e2e, test_e2e_vm, scanner_db_integration, destructive, and release-tagged tests.\n'
    printf 'NOTE: exit 2 is reserved for codegen detection and is not produced yet.\n'
    if ! race_enabled; then
        printf 'NOTE: RACE=0, so this run does not match CI (CI passes -race).\n'
    fi
}

main() {
    root=$(repo_root)
    cd "${root}" || setup_error "cannot enter ${root}"
    resolve_base

    local -a changed_files=()
    local -a go_files=()
    local f
    mapfile -t changed_files < <(collect_changed_files "${merge_base_sha}")
    for f in ${changed_files[@]+"${changed_files[@]}"}; do
        [[ -z "${f}" ]] && continue
        if [[ "${f}" == *.go ]]; then
            go_files+=("${f}")
        fi
    done
    printf 'files: %s (%s go)\n' "${#changed_files[@]}" "${#go_files[@]}"
    if [[ ${#changed_files[@]} -gt 0 && ${#changed_files[@]} -le 50 ]]; then
        printf '  %s\n' "${changed_files[@]}"
    fi

    if [[ ${#go_files[@]} -eq 0 ]]; then
        summary_add "go test" "no Go packages" "pass" "-" ""
        summary_add "golangci-lint" "no Go packages" "pass" "-" ""
        summary_add "codegen" "not checked" "skip" "-" ""
        printf 'no Go packages in this diff. Nothing to test or lint.\n'
        finish_ok_notes
        print_summary
        exit 0
    fi

    load_make_config
    load_nested_dirs
    load_lint_flags

    local -a root_files=()
    local owner
    for f in "${go_files[@]}"; do
        if owner=$(file_nested_module "${f}"); then
            :
        else
            root_files+=("${f}")
        fi
    done

    ignored_packages=()
    sql_only_packages=()
    sql_mixed_packages=()
    local -a root_mapped=()
    local -a root_kept=()
    local mod_path

    if [[ ${#root_files[@]} -gt 0 ]]; then
        local mapped
        if ! mapped=$(map_go_files "." "${root_files[@]}"); then
            setup_error "go list failed while mapping changed files to packages"
        fi
        if [[ -n "${mapped}" ]]; then
            mapfile -t root_mapped <<< "${mapped}"
        fi
        mod_path=$(go list -m)
        partition_tests "." "${mod_path}" root_kept sql_only_packages sql_mixed_packages ${root_mapped[@]+"${root_mapped[@]}"}
    fi

    local -a lint_rels=()
    local pkg rel
    if [[ ${#root_kept[@]} -gt 0 || ${#sql_only_packages[@]} -gt 0 ]]; then
        mod_path=$(go list -m)
        for pkg in ${root_kept[@]+"${root_kept[@]}"} ${sql_only_packages[@]+"${sql_only_packages[@]}"}; do
            rel=$(import_to_rel "${mod_path}" "${pkg}")
            lint_rels+=("${rel}")
        done
    fi

    if [[ ${#root_kept[@]} -gt 0 || ${#sql_only_packages[@]} -gt 0 || ${#nested_dirs[@]} -gt 0 ]]; then
        printf 'Go packages:\n'
        for pkg in ${root_kept[@]+"${root_kept[@]}"} ${sql_only_packages[@]+"${sql_only_packages[@]}"}; do
            printf '  %s\n' "${pkg}"
        done
    fi

    if [[ ${#ignored_packages[@]} -gt 0 ]]; then
        printf 'WARN: dropped UNIT_TEST_IGNORE packages:\n'
        printf '  %s\n' "${ignored_packages[@]}"
        summary_add "unit-test-ignore" "$(package_scope "${#ignored_packages[@]}")" "skip" "-" ""
    fi
    if [[ ${#sql_only_packages[@]} -gt 0 ]]; then
        printf 'WARN: skipping sql_integration packages. Postgres is not started. Run make go-postgres-unit-tests.\n'
        printf '  %s\n' "${sql_only_packages[@]}"
        summary_add "sql_integration" "$(package_scope "${#sql_only_packages[@]}")" "skip" "-" "make go-postgres-unit-tests"
    fi
    if [[ ${#sql_mixed_packages[@]} -gt 0 ]]; then
        printf 'WARN: these packages also contain sql_integration tests, which were not run:\n'
        printf '  %s\n' "${sql_mixed_packages[@]}"
        summary_add "sql_integration (mixed)" "$(package_scope "${#sql_mixed_packages[@]}")" "skip" "-" "make go-postgres-unit-tests"
    fi

    if [[ ${#root_kept[@]} -gt 0 ]]; then
        go_test_root "${root_kept[@]}"
    elif [[ ${#root_files[@]} -gt 0 ]]; then
        summary_add "go test" "0 packages" "skip" "-" ""
    fi

    local d nested_files_text
    local -a nested_files=() nested_mapped=() nested_kept=() nested_sql=() nested_mixed=()
    for d in ${nested_dirs[@]+"${nested_dirs[@]}"}; do
        nested_files=()
        for f in "${go_files[@]}"; do
            if owner=$(file_nested_module "${f}"); then
                if [[ "${owner}" == "${d}" ]]; then
                    nested_files+=("${f}")
                fi
            fi
        done
        [[ ${#nested_files[@]} -eq 0 ]] && continue
        nested_mapped=()
        nested_kept=()
        nested_sql=()
        nested_mixed=()
        if ! nested_files_text=$(map_go_files "${d}" "${nested_files[@]}"); then
            setup_error "go list failed in nested module ${d}"
        fi
        if [[ -n "${nested_files_text}" ]]; then
            mapfile -t nested_mapped <<< "${nested_files_text}"
        fi
        mod_path=$(cd "${root}/${d}" && go list -m)
        partition_tests "${d}" "${mod_path}" nested_kept nested_sql nested_mixed ${nested_mapped[@]+"${nested_mapped[@]}"}
        if [[ ${#nested_sql[@]} -gt 0 ]]; then
            printf 'WARN: skipping sql_integration packages in %s.\n' "${d}"
            printf '  %s\n' "${nested_sql[@]}"
            summary_add "sql_integration" "${d}" "skip" "-" "make go-postgres-unit-tests"
        fi
        if [[ ${#nested_mixed[@]} -gt 0 ]]; then
            printf 'WARN: nested module %s also has sql_integration tests, which were not run:\n' "${d}"
            printf '  %s\n' "${nested_mixed[@]}"
            summary_add "sql_integration (mixed)" "${d}" "skip" "-" "make go-postgres-unit-tests"
        fi
        if [[ ${#nested_kept[@]} -gt 0 ]]; then
            printf 'Go packages:\n'
            printf '  %s\n' "${nested_kept[@]}"
            go_test_nested "${d}" "${nested_kept[@]}"
        fi
        if [[ ${#nested_mapped[@]} -gt 0 ]]; then
            printf 'WARN: golangci-lint was not run in nested module %s. The root .golangci.yml targets the root module.\n' "${d}"
            summary_add "golangci-lint (nested)" "${d}" "skip" "-" ""
        fi
    done

    if [[ ${#lint_rels[@]} -gt 0 ]]; then
        ensure_linter
        if [[ "${DRY_RUN:-0}" != "1" ]]; then
            local start elapsed
            start=$(epoch_ns)
            if ! go mod download; then
                setup_error "go mod download failed. golangci-lint uses modules-download-mode: readonly."
            fi
            elapsed=$(format_elapsed "${start}")
            summary_add "go mod download" "root module" "pass" "${elapsed}" "go mod download"
        fi
        lint_packages "default" "${lint_rels[@]}"
        lint_packages "release" "${lint_rels[@]}"
    fi

    summary_add "codegen" "not checked" "skip" "-" ""
    finish_ok_notes
    print_summary
    exit "${final_exit}"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
