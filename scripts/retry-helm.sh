#!/usr/bin/env bash

# Runs the helm CLU with supplied arguments and retries on failures indicating network errors.
#
# To detect such transient errors, stderr from helm is redirected to a file and grepped for certain patterns
# in case helm exits with an error code.
#
# To make helm invocations a little bit more idempotent in the presence of failures we are injecting
# the --atomic flag in case the helm command is `install` or `upgrade`.

set -euo pipefail

# RegExes for which we attempt a retry (one regex per line).
error_regex=$(tr '\n' '|' <<EOT | sed -e 's/|$//;'
: the server is currently unable to handle the request
EOT
)

tmp_in="$(mktemp)"
tmp_out="$(mktemp --suffix=-stdout.txt)"
tmp_err="$(mktemp --suffix=-stderr.txt)"
grep_out="$(mktemp)"
trap 'cat ${tmp_out}; cat ${tmp_err} >&2; rm -f ${tmp_in} ${tmp_out} ${tmp_err} ${grep_out}' EXIT

inject_atomic_flag() {
    local -n modified_args_ref=$1; shift
    local orig_args=("$@")
    modified_args_ref=()
    local i=0
    for arg in "${orig_args[@]}"; do
        modified_args_ref+=("$arg")
        i=$((i + 1)) # Shall always point to the beginnig of the yet-unprocessed sub-array of the orig_args.
        if [[ "$arg" == "-"* ]]; then
            # Some flag before the actual Helm command.
            :
        else
            # Some command.
            if [[ $arg == "install" || $arg == "upgrade" ]]; then
                # Inject atomic flag.
                modified_args_ref+=("--atomic")
            fi
            break
        fi
    done

    # Copy remaining.
    modified_args_ref+=("${orig_args[@]:$i}")
}

declare -a modified_args=()
inject_atomic_flag modified_args "$@"

# We do not set -e on purpose, to be able to capture the exit code.
set +e
set +o pipefail

attempts=5
for attempt in $(seq 0 ${attempts}); do
    delay=$((attempt*attempt)) # Crude exponential backoff.
    sleep "${delay}"
    "${HELM_CMD:-helm}" "${modified_args[@]}" > "${tmp_out}" 2>"${tmp_err}"
    ret="$?"
    [[ $ret -eq 0 ]] && exit 0
    if [[ ${attempt} -eq ${attempts} ]] || ! grep --extended-regexp "${error_regex}" "${tmp_err}" > "${grep_out}"; then
        break
    fi
    echo "$(date -Ins) Found the following message(s) in helm stderr, retrying..." >&2
    cat "${grep_out}" >&2
done
exit "${ret}"
