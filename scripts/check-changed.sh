#!/usr/bin/env bash
# Test and lint the Go packages touched by the current diff.
#
# Exit codes:
#   0  every step that ran passed
#   1  a test or a lint failed
#   2  codegen is needed
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
deps_mode_value=direct
untested_dependents=0
warn_count=0
pg_failed=0
skip_deps_note=0
declare -A codegen_make=()
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

file_scope() {
    local n=$1
    if [[ "${n}" -eq 1 ]]; then
        printf '1 file'
    else
        printf '%s files' "${n}"
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

warn() {
    printf 'WARN: %s\n' "$*"
    warn_count=$((warn_count + 1))
}

# postgres_ready is true when something accepts TCP connections on Postgres's
# port. PG_READY=1 or PG_READY=0 overrides the probe for tests.
postgres_ready() {
    if [[ -n "${PG_READY+x}" ]]; then
        [[ "${PG_READY}" == "1" ]]
        return
    fi
    local host=${PGHOST:-127.0.0.1}
    local port=${PGPORT:-5432}
    if command -v pg_isready >/dev/null 2>&1; then
        pg_isready -h "${host}" -p "${port}" -t 1 >/dev/null 2>&1
        return
    fi
    timeout 1 bash -c "echo >/dev/tcp/${host}/${port}" >/dev/null 2>&1
}

# build_import_index writes one go list record per package:
# import|imports|testimports|xtestimports|deps
# A clean HEAD reuses .check-changed-cache/<sha>/.
build_import_index() {
    local module_dir=$1
    local out_file=$2
    local head dirty key cache_dir cache_file work
    head=$(git -C "${root}" rev-parse HEAD)
    if [[ -n "$(git -C "${root}" status --porcelain)" ]]; then
        dirty=1
    else
        dirty=0
    fi
    key=${module_dir//\//_}
    if [[ "${key}" == "." ]]; then
        key=root
    fi
    cache_dir="${root}/.check-changed-cache/${head}"
    cache_file="${cache_dir}/${key}.list"
    if [[ "${dirty}" -eq 0 && -s "${cache_file}" ]]; then
        cp "${cache_file}" "${out_file}"
        printf 'import index: cache %s\n' "${cache_file#"${root}/"}"
        return 0
    fi
    if [[ "${module_dir}" == "." ]]; then
        work=${root}
    else
        work="${root}/${module_dir}"
    fi
    if ! (cd "${work}" && go list -test -f '{{.ImportPath}}|{{join .Imports ","}}|{{join .TestImports ","}}|{{join .XTestImports ","}}|{{join .Deps ","}}' ./...) > "${out_file}"; then
        return 1
    fi
    if [[ "${dirty}" -eq 0 ]]; then
        mkdir -p "${cache_dir}"
        cp "${out_file}" "${cache_file}"
    fi
}

# list_importers prints module packages that import any of the named packages.
# kind=direct uses Imports, TestImports, and XTestImports.
# kind=all also uses Deps, which is the transitive closure.
list_importers() {
    local index=$1
    local kind=$2
    local mod=$3
    shift 3
    local targets
    targets=$(printf '%s,' "$@")
    targets=${targets%,}
    awk -F'|' -v kind="${kind}" -v targets="${targets}" -v mod="${mod}" '
        function has(list, target,    n, i, a) {
            if (list == "") return 0
            n = split(list, a, ",")
            for (i = 1; i <= n; i++) if (a[i] == target) return 1
            return 0
        }
        function in_mod(path, mod) {
            if (mod == "") return 1
            if (path == mod) return 1
            return index(path, mod "/") == 1
        }
        BEGIN { n = split(targets, t, ",") }
        {
            path = $1
            # go list -test emits "pkg.test" and "pkg [pkg.test]".
            # Those are the test binary, not a package go test can run.
            if (path ~ /\.test$/ || path ~ /[[:space:]]/ || index(path, "[")) next
            if (!in_mod(path, mod)) next
            for (i = 1; i <= n; i++) {
                if (path == t[i]) continue
                hit = 0
                if (has($2, t[i]) || has($3, t[i]) || has($4, t[i])) hit = 1
                if (kind == "all" && has($5, t[i])) hit = 1
                if (hit) print path
            }
        }
    ' "${index}" | sort -u
}

# expand_deps adds importers to the kept and sql arrays when DEPS asks for them.
# In direct mode it only counts importers that will not be tested.
# The namerefs are the kept, sql-only, and sql-mixed arrays.
expand_deps() {
    local module_dir=$1
    local mod=$2
    # shellcheck disable=SC2178 # namerefs of the caller's arrays
    local -n _kept=$3
    # shellcheck disable=SC2178
    local -n _sql_only=$4
    # shellcheck disable=SC2178
    local -n _sql_mixed=$5
    shift 5
    local -a seeds=("$@")
    if [[ ${#seeds[@]} -eq 0 ]]; then
        return 0
    fi
    local index kind pkg
    local -a found=() extras=()
    index=$(mktemp)
    if ! build_import_index "${module_dir}" "${index}"; then
        rm -f "${index}"
        return 1
    fi
    if [[ "${deps_mode_value}" == "rdeps-all" ]]; then
        kind=all
    else
        kind=direct
    fi
    mapfile -t found < <(list_importers "${index}" "${kind}" "${mod}" "${seeds[@]}")
    rm -f "${index}"
    for pkg in ${found[@]+"${found[@]}"}; do
        [[ -z "${pkg}" ]] && continue
        if package_is_ignored "${pkg}"; then
            continue
        fi
        local already=0 one
        for one in ${seeds[@]+"${seeds[@]}"} ${_kept[@]+"${_kept[@]}"} ${_sql_only[@]+"${_sql_only[@]}"} ${_sql_mixed[@]+"${_sql_mixed[@]}"}; do
            if [[ "${one}" == "${pkg}" ]]; then
                already=1
                break
            fi
        done
        if [[ "${already}" -eq 1 ]]; then
            continue
        fi
        if [[ "${deps_mode_value}" == "direct" ]]; then
            untested_dependents=$((untested_dependents + 1))
            continue
        fi
        extras+=("${pkg}")
    done
    if [[ ${#extras[@]} -eq 0 ]]; then
        return 0
    fi
    printf 'DEPS=%s added:\n' "${deps_mode_value}"
    printf '  %s\n' "${extras[@]}"
    local -a extra_kept=() extra_sql=() extra_mixed=()
    partition_tests "${module_dir}" "${mod}" extra_kept extra_sql extra_mixed "${extras[@]}"
    _kept+=(${extra_kept[@]+"${extra_kept[@]}"})
    _sql_only+=(${extra_sql[@]+"${extra_sql[@]}"})
    _sql_mixed+=(${extra_mixed[@]+"${extra_mixed[@]}"})
}

run_root_go_test() {
    local tags=$1
    local suffix=$2
    local serial=$3
    shift 3
    local -a pkgs=("$@")
    local step="go test${suffix}"
    local -a args=(-timeout 25m -v)
    if race_enabled; then
        step="go test -race${suffix}"
        args=(-timeout 25m -race -v)
    fi
    if [[ "${serial}" == "1" ]]; then
        args+=(-p 1)
        step="${step} -p 1"
    fi
    local rerun
    rerun="CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=${tags} scripts/go-test.sh ${args[*]} ${pkgs[*]}"
    run_recorded "${step}" "$(package_scope "${#pkgs[@]}")" "${rerun}" \
        env CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS="${MUTEX_WATCHDOG_TIMEOUT_SECS:-30}" GOTAGS="${tags}" \
        "${SCRIPT_DIR}/go-test.sh" "${args[@]}" "${pkgs[@]}"
}

run_sql_packages() {
    local -a pkgs=("$@")
    [[ ${#pkgs[@]} -eq 0 ]] && return 0
    local tags pkg
    local -a rest=() mig=()
    tags="$(with_test_tag "${GOTAGS-}"),sql_integration"
    for pkg in "${pkgs[@]}"; do
        case "${pkg}" in
            */migrator | */migrator/*) mig+=("${pkg}") ;;
            *) rest+=("${pkg}") ;;
        esac
    done
    if [[ ${#rest[@]} -gt 0 ]]; then
        run_root_go_test "${tags}" " (sql_integration)" 0 "${rest[@]}"
    fi
    if [[ ${#mig[@]} -gt 0 ]]; then
        run_root_go_test "${tags}" " (sql_integration)" 1 "${mig[@]}"
    fi
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
    local tags
    tags=$(with_test_tag "${GOTAGS-}")
    run_root_go_test "${tags}" "" 0 "$@"
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
    local tags=${2-}
    shift 2
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
    if [[ -n "${tags}" ]]; then
        args+=(-tags "${tags}")
        step="${step} ${tags}"
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
    if [[ "${skip_deps_note}" -eq 1 ]]; then
        printf 'NOTE: only packages that contain changed files were tested. Importers were not.\n'
    elif [[ "${deps_mode_value}" == "rdeps" ]]; then
        printf 'NOTE: DEPS=rdeps added one level of importers.\n'
    elif [[ "${deps_mode_value}" == "rdeps-all" ]]; then
        printf 'NOTE: DEPS=rdeps-all added the transitive importers.\n'
    else
        printf 'NOTE: untested importers: %s. Run with DEPS=rdeps.\n' "${untested_dependents}"
    fi
    if [[ "${RELEASE:-0}" == "1" ]]; then
        printf 'NOTE: not run: integration, test_e2e, test_e2e_vm, scanner_db_integration, destructive.\n'
    else
        printf 'NOTE: not run: integration, test_e2e, test_e2e_vm, scanner_db_integration, destructive, and release-tagged tests.\n'
    fi
    if ! race_enabled; then
        printf 'NOTE: RACE=0, so this run does not match CI (CI passes -race).\n'
    fi
}

write_output_json() {
    [[ "${OUTPUT:-}" == "json" ]] || return 0
    mkdir -p "${root}/test-output"
    local dest="${root}/test-output/check-changed.json"
    local tmp i
    tmp=$(mktemp)
    for i in "${!summary_steps[@]}"; do
        printf '%s\t%s\t%s\t%s\t%s\n' \
            "${summary_steps[$i]}" "${summary_scopes[$i]}" "${summary_results[$i]}" \
            "${summary_times[$i]}" "${summary_reruns[$i]}"
    done > "${tmp}"
    python3 - "${tmp}" "${dest}" "${final_exit}" "${deps_mode_value}" "${untested_dependents}" "${merge_base_sha}" << 'PY'
import json
import sys

src, dest, exit_code, deps, untested, base = sys.argv[1:7]
steps = []
with open(src, encoding="utf-8") as handle:
    for line in handle:
        line = line.rstrip("\n")
        if not line:
            continue
        parts = line.split("\t")
        while len(parts) < 5:
            parts.append("")
        steps.append({
            "step": parts[0],
            "scope": parts[1],
            "result": parts[2],
            "time": parts[3],
            "rerun": parts[4],
        })
doc = {
    "exit": int(exit_code),
    "base": base,
    "deps": deps,
    "untested_dependents": int(untested),
    "steps": steps,
}
with open(dest, "w", encoding="utf-8") as handle:
    json.dump(doc, handle, indent=2)
    handle.write("\n")
PY
    rm -f "${tmp}"
    printf 'wrote %s\n' "${dest#"${root}/"}"
}

note_codegen() {
    local target=$1
    local why=$2
    if [[ -z "${codegen_make[${target}]+x}" ]]; then
        codegen_make[${target}]=""
    fi
    codegen_make[${target}]+="${why}"$'\n'
}

# scan_codegen records make targets for changed generator inputs.
# It does not run the generators.
scan_codegen() {
    local f dir
    codegen_make=()
    for f in "$@"; do
        [[ -z "${f}" ]] && continue
        case "${f}" in
            proto/storage/*)
                note_codegen proto-generated-srcs "${f}"
                note_codegen storage-protos-compatible "${f}"
                ;;
            proto/*)
                note_codegen proto-generated-srcs "${f}"
                ;;
            config-controller/api/*)
                note_codegen config-controller-gen "${f}"
                ;;
            central/graphql/schema/* | central/graphql/generator/* | central/graphql/resolvers/gen/*)
                note_codegen go-generated-srcs "${f}"
                ;;
        esac
        if [[ -f "${root}/${f}" ]] && grep -q '//go:generate' "${root}/${f}"; then
            note_codegen go-generated-srcs "${f}"
        fi
        if [[ "${f}" == *.go && -f "${root}/${f}" ]]; then
            dir=$(dirname "${root}/${f}")
            if [[ -d "${dir}/mocks" ]] && grep -Eq '^type[[:space:]]+[[:alnum:]_]+[[:space:]]+interface([[:space:]]|\{|$)' "${root}/${f}"; then
                note_codegen go-generated-srcs "${f}"
            fi
        fi
    done
}

ui_modules_dir() {
    if [[ -d "${root}/ui/apps/platform/node_modules" ]]; then
        printf '%s\n' "${root}/ui/apps/platform/node_modules"
        return 0
    fi
    if [[ -d "${root}/ui/node_modules" ]]; then
        printf '%s\n' "${root}/ui/node_modules"
        return 0
    fi
    return 1
}

ui_tool_cmd() {
    local name=$1
    local modules bin
    if modules=$(ui_modules_dir); then
        bin="${modules}/.bin/${name}"
        if [[ -x "${bin}" ]]; then
            printf '%s\n' "${bin}"
            return 0
        fi
    fi
    printf 'npx\n--no-install\n%s\n' "${name}"
}

run_ui_checks() {
    local -a files=() code=()
    local f rel modules
    for f in "$@"; do
        case "${f}" in
            ui/apps/platform/src/*) files+=("${f}") ;;
        esac
    done
    if [[ ${#files[@]} -eq 0 ]]; then
        return 0
    fi
    for f in "${files[@]}"; do
        case "${f}" in
            *.js | *.jsx | *.ts | *.tsx | *.mjs | *.cjs)
                code+=("${f#ui/apps/platform/}")
                ;;
        esac
    done
    if ! modules=$(ui_modules_dir); then
        warn "ui node_modules is missing. Run npm ci in ui/."
        summary_add "ui" "$(file_scope "${#files[@]}")" "skip" "-" "npm ci --prefix ui"
        return 0
    fi
    local tool_file start elapsed rerun status=0
    local -a eslint_cmd=() vitest_cmd=() tsc_cmd=()
    tool_file=$(mktemp)
    if [[ ${#code[@]} -gt 0 ]]; then
        ui_tool_cmd eslint > "${tool_file}"
        mapfile -t eslint_cmd < "${tool_file}"
        rerun="(cd ui/apps/platform && ${eslint_cmd[*]} --cache --cache-strategy content ${code[*]})"
        if [[ "${DRY_RUN:-0}" == "1" ]]; then
            printf 'DRY_RUN eslint: %s\n' "${rerun}"
            summary_add "eslint" "$(file_scope "${#code[@]}")" "dry-run" "-" "${rerun}"
        else
            start=$(epoch_ns)
            set +e
            (cd "${root}/ui/apps/platform" && "${eslint_cmd[@]}" --cache --cache-strategy content "${code[@]}")
            status=$?
            set -e
            elapsed=$(format_elapsed "${start}")
            if [[ "${status}" -eq 0 ]]; then
                summary_add "eslint" "$(file_scope "${#code[@]}")" "pass" "${elapsed}" "${rerun}"
            else
                summary_add "eslint" "$(file_scope "${#code[@]}")" "fail" "${elapsed}" "${rerun}"
                final_exit=1
            fi
        fi
        ui_tool_cmd vitest > "${tool_file}"
        mapfile -t vitest_cmd < "${tool_file}"
        rerun="(cd ui/apps/platform && TZ=UTC ${vitest_cmd[*]} related --run ${code[*]})"
        if [[ "${DRY_RUN:-0}" == "1" ]]; then
            printf 'DRY_RUN vitest: %s\n' "${rerun}"
            summary_add "vitest related" "$(file_scope "${#code[@]}")" "dry-run" "-" "${rerun}"
        else
            start=$(epoch_ns)
            set +e
            (cd "${root}/ui/apps/platform" && TZ=UTC "${vitest_cmd[@]}" related --run "${code[@]}")
            status=$?
            set -e
            elapsed=$(format_elapsed "${start}")
            if [[ "${status}" -eq 0 ]]; then
                summary_add "vitest related" "$(file_scope "${#code[@]}")" "pass" "${elapsed}" "${rerun}"
            else
                summary_add "vitest related" "$(file_scope "${#code[@]}")" "fail" "${elapsed}" "${rerun}"
                final_exit=1
            fi
        fi
    fi
    if [[ "${UI_TSC:-1}" == "0" ]]; then
        printf 'UI_TSC=0, skipping tsc.\n'
        summary_add "tsc" "skipped" "skip" "-" ""
    else
        ui_tool_cmd tsc > "${tool_file}"
        mapfile -t tsc_cmd < "${tool_file}"
        rerun="(cd ui/apps/platform && ${tsc_cmd[*]} --noEmit)"
        if [[ "${DRY_RUN:-0}" == "1" ]]; then
            printf 'DRY_RUN tsc: %s\n' "${rerun}"
            summary_add "tsc" "ui/apps/platform" "dry-run" "-" "${rerun}"
        else
            start=$(epoch_ns)
            set +e
            (cd "${root}/ui/apps/platform" && "${tsc_cmd[@]}" --noEmit)
            status=$?
            set -e
            elapsed=$(format_elapsed "${start}")
            if [[ "${status}" -eq 0 ]]; then
                summary_add "tsc" "ui/apps/platform" "pass" "${elapsed}" "${rerun}"
            else
                summary_add "tsc" "ui/apps/platform" "fail" "${elapsed}" "${rerun}"
                final_exit=1
            fi
        fi
    fi
    rm -f "${tool_file}"
}

report_codegen() {
    local target reasons
    if [[ ${#codegen_make[@]} -eq 0 ]]; then
        summary_add "codegen" "none" "pass" "-" ""
        return 0
    fi
    printf 'codegen:\n'
    for target in "${!codegen_make[@]}"; do
        printf '  make %s\n' "${target}"
        reasons=${codegen_make[${target}]}
        while IFS= read -r reasons; do
            [[ -z "${reasons}" ]] && continue
            printf '    %s\n' "${reasons}"
        done <<< "${codegen_make[${target}]}"
    done
    local scope
    if [[ ${#codegen_make[@]} -eq 1 ]]; then
        scope="1 target"
    else
        scope="${#codegen_make[@]} targets"
    fi
    if [[ "${CODEGEN:-}" == "skip" ]]; then
        printf 'CODEGEN=skip, so this did not fail the run.\n'
        summary_add "codegen" "${scope}" "skip" "-" "CODEGEN=skip"
        return 0
    fi
    summary_add "codegen" "${scope}" "fail" "-" "run the make targets above"
    if [[ "${final_exit}" -eq 0 ]]; then
        final_exit=${EXIT_CODEGEN}
    fi
}

finish_run() {
    if [[ "${STRICT:-0}" == "1" && "${warn_count}" -gt 0 && "${final_exit}" -eq 0 ]]; then
        printf 'STRICT=1 and a warning was printed.\n'
        final_exit=1
    fi
    if [[ "${pg_failed}" -eq 1 && "${final_exit}" -eq 0 ]]; then
        final_exit=1
    fi
    print_summary
    write_output_json
    exit "${final_exit}"
}

run_flake() {
    local name=${FLAKE}
    local pkg=${PKG:-}
    local count=${COUNT:-20}
    local extra_tags=${TAGS:-}
    local i status pass=0 fail=0
    local log safe first="" tags step
    local -a args
    if [[ -z "${pkg}" ]]; then
        setup_error "PKG is required when FLAKE is set. Example: FLAKE=TestFoo PKG=./pkg/foo COUNT=20"
    fi
    if [[ ! "${count}" =~ ^[1-9][0-9]*$ ]]; then
        setup_error "COUNT must be a positive integer"
    fi
    if [[ ! "${name}" =~ ^[A-Za-z0-9_./]+$ ]]; then
        setup_error "FLAKE must be a Go test name, such as TestFoo or Suite/Method"
    fi
    tags=$(with_test_tag "${GOTAGS-}")
    if [[ -n "${extra_tags}" ]]; then
        tags="${tags},${extra_tags}"
    fi
    args=(-count=1 -timeout 25m -v -run "^${name}$" "${pkg}")
    step="flake"
    if race_enabled; then
        args=(-count=1 -timeout 25m -race -v -run "^${name}$" "${pkg}")
        step="flake -race"
    fi
    if [[ -n "${PARALLEL:-}" ]]; then
        args+=(-parallel "${PARALLEL}")
    fi
    safe=${name//\//_}
    mkdir -p "${root}/test-output/flake"
    local stress_pid=""
    if [[ "${STRESS:-0}" == "1" ]]; then
        yes >/dev/null &
        stress_pid=$!
    fi
    local start
    start=$(epoch_ns)
    for ((i = 1; i <= count; i++)); do
        log="${root}/test-output/flake/${safe}.${i}.log"
        set +e
        env CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS="${MUTEX_WATCHDOG_TIMEOUT_SECS:-30}" GOTAGS="${tags}" \
            "${SCRIPT_DIR}/go-test.sh" "${args[@]}" > "${log}" 2>&1
        status=$?
        set -e
        if [[ "${status}" -eq 0 ]]; then
            pass=$((pass + 1))
            printf 'flake %s/%s pass\n' "${i}" "${count}"
        else
            fail=$((fail + 1))
            printf 'flake %s/%s fail\n' "${i}" "${count}"
            if [[ -z "${first}" ]]; then
                first="${root}/test-output/flake/${safe}.first-failure.log"
                cp "${log}" "${first}"
            fi
        fi
    done
    if [[ -n "${stress_pid}" ]]; then
        kill "${stress_pid}" >/dev/null 2>&1 || true
        wait "${stress_pid}" 2>/dev/null || true
    fi
    local elapsed result rerun
    elapsed=$(format_elapsed "${start}")
    printf 'flake result: pass %s / fail %s\n' "${pass}" "${fail}"
    rerun="CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=${tags} scripts/go-test.sh ${args[*]}"
    if [[ "${fail}" -eq 0 ]]; then
        result=pass
    else
        result=fail
        final_exit=1
        printf 'first failure: %s\n' "${first#"${root}/"}"
    fi
    summary_add "${step}" "${name} x ${count}" "${result}" "${elapsed}" "${rerun}"
    print_summary
    write_output_json
    exit "${final_exit}"
}

main() {
    case "${DEPS:-direct}" in
        direct | rdeps | rdeps-all) deps_mode_value=${DEPS:-direct} ;;
        *) setup_error "DEPS=${DEPS} is not direct, rdeps, or rdeps-all" ;;
    esac
    root=$(repo_root)
    cd "${root}" || setup_error "cannot enter ${root}"
    if [[ -n "${FLAKE:-}" ]]; then
        run_flake
    fi
    resolve_base

    local -a changed_files=()
    local -a go_files=()
    local f
    mapfile -t changed_files < <(collect_changed_files "${merge_base_sha}")
    scan_codegen ${changed_files[@]+"${changed_files[@]}"}
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
        skip_deps_note=1
        summary_add "go test" "no Go packages" "pass" "-" ""
        summary_add "golangci-lint" "no Go packages" "pass" "-" ""
        printf 'no Go packages in this diff. Nothing to test or lint.\n'
        run_ui_checks ${changed_files[@]+"${changed_files[@]}"}
        report_codegen
        finish_ok_notes
        finish_run
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
        if ! expand_deps "." "${mod_path}" root_kept sql_only_packages sql_mixed_packages \
            ${root_kept[@]+"${root_kept[@]}"} ${sql_only_packages[@]+"${sql_only_packages[@]}"} ${sql_mixed_packages[@]+"${sql_mixed_packages[@]}"}; then
            setup_error "go list failed while building the importer index"
        fi
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
    local pg_up=0
    if postgres_ready; then
        pg_up=1
    fi
    if [[ ${#sql_only_packages[@]} -gt 0 || ${#sql_mixed_packages[@]} -gt 0 ]]; then
        if [[ "${pg_up}" -eq 1 ]]; then
            printf 'Postgres is reachable. Running sql_integration tests.\n'
            run_sql_packages ${sql_only_packages[@]+"${sql_only_packages[@]}"} ${sql_mixed_packages[@]+"${sql_mixed_packages[@]}"}
        else
            if [[ ${#sql_only_packages[@]} -gt 0 ]]; then
                warn "skipping sql_integration packages. Postgres is not reachable on ${PGHOST:-127.0.0.1}:${PGPORT:-5432}. Run make go-postgres-unit-tests."
                printf '  %s\n' "${sql_only_packages[@]}"
                summary_add "sql_integration" "$(package_scope "${#sql_only_packages[@]}")" "skip" "-" "make go-postgres-unit-tests"
            fi
            if [[ ${#sql_mixed_packages[@]} -gt 0 ]]; then
                warn "these packages also contain sql_integration tests, which were not run:"
                printf '  %s\n' "${sql_mixed_packages[@]}"
                summary_add "sql_integration (mixed)" "$(package_scope "${#sql_mixed_packages[@]}")" "skip" "-" "make go-postgres-unit-tests"
            fi
            if [[ "${PG:-}" == "require" ]]; then
                pg_failed=1
            fi
        fi
    fi

    if [[ ${#root_kept[@]} -gt 0 ]]; then
        go_test_root "${root_kept[@]}"
        if [[ "${RELEASE:-0}" == "1" ]]; then
            run_root_go_test "$(with_test_tag "${RELEASE_GOTAGS}")" " (release)" 0 "${root_kept[@]}"
        fi
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
        if ! expand_deps "${d}" "${mod_path}" nested_kept nested_sql nested_mixed \
            ${nested_kept[@]+"${nested_kept[@]}"} ${nested_sql[@]+"${nested_sql[@]}"} ${nested_mixed[@]+"${nested_mixed[@]}"}; then
            setup_error "go list failed while building the importer index in ${d}"
        fi
        if [[ ${#nested_sql[@]} -gt 0 || ${#nested_mixed[@]} -gt 0 ]]; then
            if [[ "${pg_up}" -eq 1 ]]; then
                printf 'Postgres is reachable. Running sql_integration tests in %s.\n' "${d}"
                go_test_nested "${d}" "sql_integration" ${nested_sql[@]+"${nested_sql[@]}"} ${nested_mixed[@]+"${nested_mixed[@]}"}
            else
                if [[ ${#nested_sql[@]} -gt 0 ]]; then
                    warn "skipping sql_integration packages in ${d}."
                    printf '  %s\n' "${nested_sql[@]}"
                    summary_add "sql_integration" "${d}" "skip" "-" "make go-postgres-unit-tests"
                fi
                if [[ ${#nested_mixed[@]} -gt 0 ]]; then
                    warn "nested module ${d} also has sql_integration tests, which were not run:"
                    printf '  %s\n' "${nested_mixed[@]}"
                    summary_add "sql_integration (mixed)" "${d}" "skip" "-" "make go-postgres-unit-tests"
                fi
                if [[ "${PG:-}" == "require" ]]; then
                    pg_failed=1
                fi
            fi
        fi
        if [[ ${#nested_kept[@]} -gt 0 ]]; then
            printf 'Go packages:\n'
            printf '  %s\n' "${nested_kept[@]}"
            go_test_nested "${d}" "" "${nested_kept[@]}"
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

    run_ui_checks ${changed_files[@]+"${changed_files[@]}"}
    report_codegen
    finish_ok_notes
    finish_run
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
