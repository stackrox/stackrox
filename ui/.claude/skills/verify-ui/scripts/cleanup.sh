#!/usr/bin/env bash
# Tear down what verify-ui started. Evidence under runs/ is never removed.
#
#   cleanup.sh             stop the dev server, revoke verify-ui-* API tokens
#   cleanup.sh --backend   also delete the connected infra cluster

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

if [[ -f "${BACKEND_ENV}" && -f "${PASSWORD_FILE}" ]]; then
    load_backend
    ids="$(admin_curl "${UI_START_TARGET}/v1/apitokens?revoked=false" --max-time 15 \
        | jq -r '.tokens[]? | select(.name | startswith("verify-ui-")) | .id' || true)"
    for id in ${ids}; do
        admin_curl "${UI_START_TARGET}/v1/apitokens/revoke/${id}" -X PATCH -o /dev/null
    done
    info "revoked $(echo "${ids}" | grep -c . || true) verify-ui API token(s)"
fi

if vite_running; then
    pid="$(vite_pid)"
    kill "${pid}"
    for _ in $(seq 1 10); do kill -0 "${pid}" 2>/dev/null || break; sleep 1; done
    kill -0 "${pid}" 2>/dev/null && kill -9 "${pid}"
    info "stopped dev server pid ${pid}"
fi
rm -f "${VITE_PID_FILE}"

if [[ "${1:-}" == "--backend" ]]; then
    "$(dirname "${BASH_SOURCE[0]}")/backend.sh" delete
fi

info "evidence kept in ${RUNS_DIR}"
