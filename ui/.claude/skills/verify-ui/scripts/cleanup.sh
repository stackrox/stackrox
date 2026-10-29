#!/usr/bin/env bash
# Tear down what verify-ui started. Evidence under runs/ is never removed here (drive.sh
# keeps the last VERIFY_UI_KEEP_ATTEMPTS attempts per spec).
#
#   cleanup.sh             stop the dev server, revoke recorded API tokens
#   cleanup.sh --backend   also delete the connected infra cluster

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

acquire_lifecycle_lock
trap release_lifecycle_lock EXIT
runs="$(active_runs)"
[[ -z "${runs}" ]] || die "active drive.sh run(s): $(tr '\n' ' ' <<< "${runs}"); wait for them to finish before cleanup"

# Check ownership before touching tokens or the backend. A reused pid must never be killed.
server_running=0
if vite_running; then
    server_running=1
fi

if [[ -f "${BACKEND_ENV}" && -f "${PASSWORD_FILE}" ]]; then
    load_backend
    revoked=0
    failed=0
    for record in "${TOKENS_DIR}"/*; do
        [[ -d "${record}" && -f "${record}/target" && -f "${record}/id" ]] || continue
        [[ "$(cat "${record}/target")" == "${UI_START_TARGET}" ]] || continue
        if revoke_recorded_token "${record}"; then
            ((revoked += 1))
        else
            ((failed += 1))
        fi
    done
    info "revoked ${revoked} recorded API token(s)"
    [[ "${failed}" -eq 0 ]] || die "could not revoke ${failed} recorded API token(s); retry cleanup after checking Central"
fi

if [[ "${server_running}" -eq 1 ]]; then
    pid="$(vite_pid)"
    kill "${pid}"
    for _ in $(seq 1 10); do vite_pid_alive || break; sleep 1; done
    if vite_pid_alive; then
        vite_running
        kill -9 "${pid}"
    fi
    info "stopped dev server pid ${pid}"
fi
rm -f "${VITE_PID_FILE}"

if [[ "${1:-}" == "--backend" ]]; then
    "$(dirname "${BASH_SOURCE[0]}")/backend.sh" delete
fi

info "evidence kept in ${RUNS_DIR}"
