#!/bin/bash
# Runs kubectl with supplied arguments, and retries on failures indicating network errors.
#
# NOTE: Reads all of stdin. If no input should be passed to kubectl, caller should redirect stdin from /dev/null!
#
# To detect such transient timeout errors, stderr from kubectl is redirected to a file and grepped for certain patterns
# in case kubectl exits with an error code.

set -euo pipefail

# RegExes for which we attempt a retry (one regex per line).
error_regex=$(tr '\n' '|' <<EOT | sed -e 's/|$//;'
: i/o timeout$
net/http: request canceled \(Client\.Timeout exceeded while awaiting headers\)$
: the server is currently unable to handle the request
EOT
)

tmp_in="$(mktemp)"
tmp_out="$(mktemp --suffix=-stdout.txt)"
tmp_err="$(mktemp --suffix=-stderr.txt)"
grep_out="$(mktemp)"
trap 'cat ${tmp_out}; cat ${tmp_err} >&2; rm -f ${tmp_in} ${tmp_out} ${tmp_err} ${grep_out}' EXIT

cat > "${tmp_in}"

# We do not set -e on purpose, to be able to capture the exit code.
set +e
set +o pipefail

attempts=5
for attempt in $(seq 0 ${attempts})
do
    delay=$((attempt*attempt)) # Crude exponential backoff.
    sleep "${delay}"
    "${KUBECTL:-kubectl}" "$@" < "${tmp_in}" > "${tmp_out}" 2>"${tmp_err}"
    ret="$?"
    [[ $ret -eq 0 ]] && exit 0
    if [[ ${attempt} -eq ${attempts} ]] || ! grep --extended-regexp "${error_regex}" "${tmp_err}" > "${grep_out}"
    then
        break
    fi
    echo "$(date -Ins) Found the following message(s) in kubectl stderr, retrying..." >&2
    cat "${grep_out}" >&2
done
exit "${ret}"
