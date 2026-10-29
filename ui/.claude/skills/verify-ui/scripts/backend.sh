#!/usr/bin/env bash
# Provision or connect the Central that the UI dev server proxies to.
#
#   backend.sh create [name]          start an infra qa-demo running the image for local HEAD
#   backend.sh wait <name>            block until the infra cluster is READY
#   backend.sh connect <name>         fetch Central URL + admin password from infra artifacts
#   backend.sh connect-roxie <envrc>  use the Central from a roxie deploy --envrc file
#   backend.sh connect-url <url> <password-file>   use any reachable Central
#   backend.sh status                 print the connected backend (never the password)
#   backend.sh delete                 delete the connected infra cluster and forget it

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

cmd="${1:-status}"
shift || true

write_backend_env() {
    local target="$1" cluster="${2:-}" user="${3:-admin}"
    printf 'UI_START_TARGET=%q\nINFRA_CLUSTER=%q\nCENTRAL_USER=%q\n' "${target}" "${cluster}" "${user}" > "${BACKEND_ENV}"
    chmod 600 "${BACKEND_ENV}"
    info "connected: UI_START_TARGET=${target}${cluster:+ (infra ${cluster})}"
}

# Verify that Central accepts the credentials, then save them. Nothing is saved on failure,
# so a bad attempt never replaces a working connection.
connect_central() {
    local target="$1" password_file="$2" cluster="${3:-}" user="${4:-admin}"
    local code
    target="${target%/}"
    [[ -s "${password_file}" ]] || die "password file ${password_file} is missing or empty"
    code="$(basic_curl "${user}" "${password_file}" "${target}/v1/auth/status" -o /dev/null -w '%{http_code}' --max-time 15 || true)"
    [[ "${code}" == "200" ]] || die "Central at ${target} $(describe_http_failure "${code}"); nothing saved"
    install -m 600 "${password_file}" "${PASSWORD_FILE}"
    write_backend_env "${target}" "${cluster}" "${user}"
}

case "${cmd}" in
create)
    command -v infractl >/dev/null || die "infractl not installed; see https://github.com/stackrox/infra"
    tag="$(local_tag)"
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
    connect_central "$(tr -d '[:space:]' < "${dl}/url")" "${dl}/admin-password" "${name}"
    ;;
connect-roxie)
    envrc="${1:?usage: backend.sh connect-roxie <envrc>}"
    [[ -f "${envrc}" ]] || die "no envrc at ${envrc}; create it with roxie deploy --envrc <path> (deploy/AGENTS.md)"
    pw="$(mktemp "${STATE_DIR}/roxie-password.XXXXXX")"
    trap 'rm -f "${pw}"' EXIT
    # roxie writes export NAME="value" lines. Read them in a subshell so nothing leaks into
    # this shell, and pass the password through a 0600 file, never the command line.
    roxie_env="$(
        unset ROX_BASE_URL ROX_ENDPOINT ROX_ADMIN_PASSWORD ROX_USERNAME
        # shellcheck disable=SC1090
        source "${envrc}" >/dev/null
        printf '%s' "${ROX_ADMIN_PASSWORD:-}" > "${pw}"
        echo "${ROX_BASE_URL:-${ROX_ENDPOINT:+https://${ROX_ENDPOINT}}}"
        echo "${ROX_USERNAME:-admin}"
    )"
    target="$(sed -n 1p <<< "${roxie_env}")"
    user="$(sed -n 2p <<< "${roxie_env}")"
    [[ -n "${target}" ]] || die "${envrc} sets neither ROX_BASE_URL nor ROX_ENDPOINT"
    [[ -s "${pw}" ]] || die "${envrc} does not set ROX_ADMIN_PASSWORD"
    connect_central "${target}" "${pw}" "" "${user}"
    ;;
connect-url)
    target="${1:?usage: backend.sh connect-url <url> <password-file>}"
    pw="${2:?usage: backend.sh connect-url <url> <password-file>}"
    connect_central "${target}" "${pw}"
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
