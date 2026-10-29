#!/usr/bin/env bash
# Provision or connect the Central that the UI dev server proxies to.
#
#   backend.sh create [name]          start an infra qa-demo running the image for local HEAD
#   backend.sh wait <name>            block until the infra cluster is READY
#   backend.sh connect <name>         fetch Central URL + admin password from infra artifacts
#   backend.sh connect-url <url> <password-file>   use any reachable Central
#   backend.sh status                 print the connected backend (never the password)
#   backend.sh delete                 delete the connected infra cluster and forget it

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

cmd="${1:-status}"
shift || true

write_backend_env() {
    local target="$1" cluster="${2:-}"
    cat > "${BACKEND_ENV}" <<EOF
UI_START_TARGET=${target}
INFRA_CLUSTER=${cluster}
EOF
    chmod 600 "${BACKEND_ENV}"
    info "connected: UI_START_TARGET=${target}${cluster:+ (infra ${cluster})}"
}

check_central() {
    local target="$1"
    local code
    code="$(admin_curl "${target}/v1/auth/status" -o /dev/null -w '%{http_code}' --max-time 15 || true)"
    [[ "${code}" == "200" ]] || die "Central at ${target} rejected admin basic auth (HTTP ${code:-none})"
}

case "${cmd}" in
create)
    command -v infractl >/dev/null || die "infractl not installed; see https://github.com/stackrox/infra"
    tag="$(make -C "${REPO_DIR}" --no-print-directory tag)"
    image="quay.io/rhacs-eng/main:${tag}"
    docker manifest inspect "${image}" >/dev/null 2>&1 \
        || die "image ${image} not on quay; push the commit and wait for CI, or check out a pushed commit"
    name="${1:-verify-ui-$(date +%m%d%H%M)}"
    # us-central1-b (flavor default) regularly runs out of GCE capacity; -a is the team's usual zone.
    # --no-slack keeps throwaway verification clusters out of #infra_notifications.
    infractl create qa-demo "${name}" --no-slack --arg main-image="${image}" \
        --arg gcp-zone="${VERIFY_UI_GCP_ZONE:-us-central1-a}" \
        --lifespan "${VERIFY_UI_LIFESPAN:-8h}" --description "verify-ui backend" >&2
    echo "${name}"
    ;;
wait)
    name="${1:?usage: backend.sh wait <name>}"
    while :; do
        status="$(infractl get "${name}" 2>&1 | awk '/Status:/{print $2}')"
        case "${status}" in
        READY) info "${name} READY"; break ;;
        FAILED|FINISHED|"") die "${name} status ${status:-unknown}; see infractl logs ${name}" ;;
        *) sleep 60 ;;
        esac
    done
    ;;
connect)
    name="${1:?usage: backend.sh connect <name>}"
    dl="${STATE_DIR}/artifacts-${name}"
    rm -rf "${dl}"
    infractl artifacts "${name}" --download-dir "${dl}" >/dev/null
    [[ -f "${dl}/url" && -f "${dl}/admin-password" ]] \
        || die "artifacts for ${name} lack url/admin-password; contents: $(ls "${dl}" | tr '\n' ' ')"
    install -m 600 "${dl}/admin-password" "${PASSWORD_FILE}"
    target="$(tr -d '[:space:]' < "${dl}/url")"
    check_central "${target}"
    write_backend_env "${target}" "${name}"
    ;;
connect-url)
    target="${1:?usage: backend.sh connect-url <url> <password-file>}"
    pw="${2:?usage: backend.sh connect-url <url> <password-file>}"
    install -m 600 "${pw}" "${PASSWORD_FILE}"
    check_central "${target}"
    write_backend_env "${target}"
    ;;
status)
    load_backend
    echo "UI_START_TARGET=${UI_START_TARGET}"
    if [[ -n "${INFRA_CLUSTER}" ]]; then
        infractl get "${INFRA_CLUSTER}" 2>&1 | grep -E 'Status|Lifespan' || true
    fi
    ;;
delete)
    load_backend
    [[ -n "${INFRA_CLUSTER}" ]] || die "connected backend is not an infra cluster; nothing to delete"
    infractl delete "${INFRA_CLUSTER}" >&2
    rm -rf "${BACKEND_ENV}" "${PASSWORD_FILE}" "${STATE_DIR}/artifacts-${INFRA_CLUSTER}"
    info "deleted ${INFRA_CLUSTER}"
    ;;
*)
    die "unknown command ${cmd}; see header of $0"
    ;;
esac
