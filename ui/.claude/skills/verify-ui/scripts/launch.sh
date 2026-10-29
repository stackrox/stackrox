#!/usr/bin/env bash
# Start the Vite dev server from this checkout, proxied to the connected Central,
# and block until it serves the app. Refuses to reuse a port it did not claim.
#
#   launch.sh            start on VERIFY_UI_PORT (default 3000)

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

load_backend

if vite_running; then
    info "dev server already running (pid $(vite_pid)) at ${UI_URL}"
    exit 0
fi

if port_in_use "${VERIFY_UI_PORT}"; then
    die "port ${VERIFY_UI_PORT} is held by a process this skill did not start; stop it or set VERIFY_UI_PORT"
fi

[[ -d "${PLATFORM_DIR}/node_modules" ]] || die "run 'npm ci' in ${PLATFORM_DIR} first (Node 22.13 or newer)"

# Run vite directly, not through npm, so the recorded pid is the server itself.
cd "${PLATFORM_DIR}"
UI_START_TARGET="${UI_START_TARGET}" nohup node node_modules/vite/bin/vite.js serve \
    --port "${VERIFY_UI_PORT}" --strictPort > "${VITE_LOG}" 2>&1 &
echo $! > "${VITE_PID_FILE}"

for _ in $(seq 1 90); do
    vite_running || die "dev server exited; see ${VITE_LOG}"
    if curl -sk -o /dev/null -w '%{http_code}' "${UI_URL}/" 2>/dev/null | grep -q 200; then
        info "dev server ready at ${UI_URL} (pid $(vite_pid), proxy -> ${UI_START_TARGET})"
        exit 0
    fi
    sleep 2
done
die "dev server not ready after 180s; see ${VITE_LOG}"
