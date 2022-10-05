#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"
if [[ "${CI:-}" == "true" ]]; then
    source "$ROOT/scripts/ci/lib.sh"
fi

known_failures_file="scripts/style/shellcheck_skip.txt"

run_shellcheck() {
    info "Running shellcheck on all .sh files (excluding ${known_failures_file})"

    pushd "$ROOT" > /dev/null

    local output="shellcheck-reports"
    local flag_failure="scripts/style/shellcheck_fail_flag"

    rm -f "${output}/*" "${flag_failure}"

    for shell in $(git ls-files | grep -E '.sh$' | grep -v -x -f "${known_failures_file}"); do
        if ! shellcheck -x "$shell"; then
            if [[ "${CI:-}" == "true" ]]; then
                mkdir -p "${output}"
                local xmlout="${shell//.sh/.xml}"
                xmlout="${xmlout//\//_}"
                shellcheck -f checkstyle -x "$shell" | xmlstarlet tr scripts/style/checkstyle2junit.xslt > \
                    "${output}/${xmlout}" || true
            fi
            touch "${flag_failure}"
        fi
    done

    popd > /dev/null

    if [[ -e "${flag_failure}" ]]; then
        rm -f "${flag_failure}"
        if [[ "${CI:-}" == "true" ]]; then
            store_test_results "${output}" "${output}"
        fi
        info "errors were detected"
        exit 1
    fi
}

update_failing_list() {
    info "Will discover shell scripts that fail shellcheck and update ${known_failures_file}"

    pushd "$ROOT" > /dev/null

    for shell in $(git ls-files | grep -E '.sh$'); do
        if ! shellcheck -x "$shell" > /dev/null; then
            echo "$shell" >> "$known_failures_file"
        fi
    done

    popd > /dev/null
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        run_shellcheck "$@"
    else
        fn="$1"
        shift
        "$fn" "$@"
    fi
fi
