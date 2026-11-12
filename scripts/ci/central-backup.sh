#!/usr/bin/env bash
#
# Generates a postgres_db_<RELEASE>.<PATCH>.sql.zip Central DB fixture for
# tests/upgrade, by deploying PREVIOUS_VERSION, restoring the last fixture
# for it, upgrading in place to RELEASE.PATCH via the operator, and backing
# up the result.
#
# Required env vars:
#   RELEASE           - target release, e.g. "4.11" (used for the operator tag and output filename)
#   PATCH             - target patch, e.g. "0"
#   PREVIOUS_VERSION  - version to migrate forward from, e.g. "4.10.0"
#   KUBECONFIG        - kubeconfig for the target cluster
#
# Run the whole thing with no arguments, or call a single function by name
# (e.g. for per-step visibility in a workflow) with its name as the first
# argument.

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$SCRIPTS_ROOT/scripts/lib.sh"
source "$SCRIPTS_ROOT/tests/upgrade/lib.sh"
source "$SCRIPTS_ROOT/tests/e2e/lib.sh"

set -Eeuo pipefail

NAMESPACE=acs-central
ENVRC_FILE="${ENVRC_FILE:-/tmp/central.envrc}"

load_envrc() {
    # shellcheck disable=SC1090
    source "$ENVRC_FILE"
}

# Central restarts itself after a DB restore and after an operator-driven
# upgrade, sometimes more than once in quick succession. kubectl port-forward
# pins to a single pod and exits for good once that pod disappears, taking
# its local listener with it, so a new forward is started whenever the ping
# cannot even connect to the local port (curl exit 7).
reconnect_port_forward() {
    kubectl -n "$NAMESPACE" rollout status deploy/central --timeout=5m

    local port="${API_ENDPOINT##*:}"
    local deadline=$((SECONDS + 300))
    local out rc
    while true; do
        rc=0
        out=$(roxcurl /v1/ping --max-time 5 2>/dev/null) || rc=$?
        if [[ "$rc" -eq 0 && "$out" == *ok* ]]; then
            break
        fi
        if (( SECONDS > deadline )); then
            cat /tmp/central-port-forward.log >&2
            die "Central did not come back within 5m"
        fi
        if (( rc == 7 )); then
            warn "port-forward died, restarting it"
            nohup kubectl -n "$NAMESPACE" port-forward svc/central "${port}:443" \
                > /tmp/central-port-forward.log 2>&1 &
        fi
        sleep 2
    done
}

deploy_previous_central() {
    require_environment "PREVIOUS_VERSION"
    github_group "Deploying ${PREVIOUS_VERSION}"

    # No test assertion checks image-level CVE content, only that images
    # exist and can be scanned without error. scanner-v4-matcher only needs
    # to reach Initialized()=true for scans to stop failing with
    # FailedPrecondition. These three bundles cover ~90% of this fixture's
    # actual images (77 debian, 42 alpine, 19 ubuntu out of 153); the rest
    # of the full bundle set (rhel-vex, suse, photon, osv, nvd, ...) adds 15+
    # minutes for data this fixture doesn't need. "do not use in production"
    # per scanner's own warning log.
    roxie deploy central \
        --tag "${PREVIOUS_VERSION}" \
        --resources ci \
        --set 'central.spec.customize.envVars[0].name=SCANNER_V4_MATCHER_VULN_BUNDLE_ALLOWLIST,central.spec.customize.envVars[0].value=alpine\,debian\,ubuntu\,stackrox-manual' \
        --set 'central.spec.customize.envVars[0].name=SCANNER_V4_MATCHER_READINESS,central.spec.customize.envVars[0].value=vulnerability' \
        --port-forwarding \
        --envrc "${ENVRC_FILE}" \
        --central-wait 5m

    github_endgroup
}

upgrade_operator() {
    require_environment "RELEASE"
    require_environment "PATCH"
    github_group "Upgrading to ${RELEASE}/${PATCH}"

    # Upgrade operator only, otherwise roxie tears down the entire ACS
    # deployment and recreates it, losing the cluster's historical data.
    roxie deploy operator --tag "${RELEASE}.${PATCH}"

    local deadline=$((SECONDS + 300))
    until kubectl -n "$NAMESPACE" get deploy/central \
            -o jsonpath='{.spec.template.spec.containers[?(@.name=="central")].image}' \
          | grep -q ":${RELEASE}.${PATCH}$"; do
        if (( SECONDS > deadline )); then
            kubectl -n "$NAMESPACE" get deploy/central \
                -o jsonpath='{.spec.template.spec.containers[*].name}{"\n"}' >&2
            die "Central was not upgraded to ${RELEASE}.${PATCH} within 5m"
        fi
        sleep 5
    done
    kubectl -n "$NAMESPACE" rollout status deploy/central --timeout=5m
    github_endgroup
}

restart_central_and_reconnect() {
    kubectl -n "$NAMESPACE" delete pod -l app=central --grace-period=0
    reconnect_port_forward
}

wait_for_image_count_stable() {
    # Poll /v1/imagescount until it stabilizes across two consecutive checks,
    # as a proxy for the reprocessor having finished its pass.
    local previous=-1
    local stabilized=0
    local current=""
    local output rc
    local deadline=$((SECONDS + 1800))
    while true; do
        rc=0
        output=$(roxcurl /v1/imagescount | jq -er '.count') || rc=$?
        if (( rc == 0 )); then
            current="$output"
            info "images count: ${current}"
            if [[ "$current" != "0" && "$current" == "$previous" ]]; then
                stabilized=1
                break
            fi
            previous="$current"
        else
            warn "image count poll failed (rc=${rc}); treating as unstable"
            previous=-1
        fi
        if (( SECONDS > deadline )); then
            warn "image count did not stabilize at a nonzero value within 30m (last seen: ${current:-unknown})"
            break
        fi
        sleep 20
    done

    if [[ "$stabilized" != "1" ]]; then
        warn "Image reprocessing did not populate images_v2; dumping reprocessor logs"
        kubectl -n "$NAMESPACE" logs deploy/central -c central --tail=500 2>&1 \
            | grep -i "reprocess" >&2 || true
        die "image count never stabilized"
    fi
}

wait_for_image_reprocessing() {
    MAX_WAIT_SECONDS="600" wait_for_scanner_V4 "acs-central"
    restart_central_and_reconnect
    wait_for_image_count_stable
}

strip_transient_cluster_auth_config() {
    # Central recreates this config for its own cluster on startup. Keeping
    # the snapshot cluster's issuer would make restores depend on that
    # cluster.
    local query='
      .configs[]?
      | select(.type == "KUBE_SERVICE_ACCOUNT")
      | select(.mappings == [{
          "key": "sub",
          "valueExpression": "system:serviceaccount:acs-central:config-controller",
          "role": "Configuration Controller"
        }])
      | .id
    '

    local id
    for id in $(roxcurl /v1/auth/m2m | jq -r "$query"); do
        [[ -n "$id" ]] || continue
        roxcurl "/v1/auth/m2m/$id" -X DELETE > /dev/null
    done
}

grab_central_backup() {
    require_environment "RELEASE"
    require_environment "PATCH"

    roxctl central backup --output "postgres_db_${RELEASE}.${PATCH}.sql.zip"
}

collect_logs() {
    info "Collecting stackrox logs to /tmp/stackrox-logs"
    "$SCRIPTS_ROOT/scripts/ci/collect-service-logs.sh" rhacs-operator-system /tmp/stackrox-logs
    "$SCRIPTS_ROOT/scripts/ci/collect-service-logs.sh" acs-central /tmp/stackrox-logs
}

run() {
    trap collect_logs ERR
    deploy_previous_central
    load_envrc
    restore_postgres_backup
    upgrade_operator
    wait_for_image_reprocessing
    wait_for_background_migrations
    strip_transient_cluster_auth_config
    grab_central_backup
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        run
    else
        fn="$1"
        shift
        "$fn" "$@"
    fi
fi
