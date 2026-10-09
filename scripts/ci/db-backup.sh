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
#   DOCKERHUB_USERNAME, DOCKERHUB_PASSWORD - used for an authenticated Docker Hub integration
#
# Run the whole thing with no arguments, or call a single function by name
# (e.g. for per-step visibility in a workflow) with its name as the first
# argument.
#
# State that must survive between steps (API_ENDPOINT, ROX_ADMIN_PASSWORD,
# ROXIE_PORT_FORWARD_PID) is written by roxie to ENVRC_FILE and re-sourced by
# every function that needs it.

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

NAMESPACE=acs-central
ENVRC_FILE="${ENVRC_FILE:-/tmp/central.envrc}"
GCS_FIXTURES_PATH="gs://stackrox-ci-upgrade-test-fixtures/upgrade-test-dbs"

load_envrc() {
    # shellcheck disable=SC1090
    source "$ENVRC_FILE"
}

wait_for_central_ping() {
    local timeout_s="$1"
    local deadline=$((SECONDS + timeout_s))
    until roxcurl /v1/ping | grep -q ok; do
        if (( SECONDS > deadline )); then
            die "Central did not come back within ${timeout_s}s"
        fi
        sleep 2
    done
}

deploy_previous_central() {
    require_environment "PREVIOUS_VERSION"

    roxie deploy central \
        --tag "${PREVIOUS_VERSION}" \
        --resources auto \
        --port-forwarding \
        --envrc "${ENVRC_FILE}" \
        --central-wait 5m
}

restore_previous_backup() {
    require_environment "PREVIOUS_VERSION"
    load_envrc

    gsutil cp "${GCS_FIXTURES_PATH}/postgres_db_${PREVIOUS_VERSION}.sql.zip" .

    roxctl -e "$API_ENDPOINT" --ca "" --insecure-skip-tls-verify \
        central db restore --timeout 5m "postgres_db_${PREVIOUS_VERSION}.sql.zip"

    # The restore bounces Central; wait for it to come back before continuing.
    wait_for_central_ping 300
}

add_dockerhub_integration() {
    require_environment "DOCKERHUB_USERNAME"
    require_environment "DOCKERHUB_PASSWORD"
    load_envrc

    # Central's default "Public DockerHub" integration is anonymous and gets
    # rate-limited (429) almost immediately when the image reprocessor tries
    # to re-scan hundreds of images in a burst. An authenticated integration
    # has a much higher rate limit.
    local payload
    payload=$(jq -n \
        --arg username "$DOCKERHUB_USERNAME" \
        --arg password "$DOCKERHUB_PASSWORD" \
        '{
          name: "Authenticated Docker Hub",
          type: "docker",
          categories: ["REGISTRY"],
          docker: {
            endpoint: "registry-1.docker.io",
            username: $username,
            password: $password
          }
        }')

    roxcurl /v1/imageintegrations -X POST -d "$payload"
}

upgrade_operator() {
    require_environment "RELEASE"
    require_environment "PATCH"
    load_envrc

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
}

wait_for_deploy() {
    local deploy="$1"
    local label="$2"
    if kubectl -n "$NAMESPACE" wait "deploy/${deploy}" --for=condition=available --timeout=10m; then
        return 0
    fi
    echo "${deploy} did not become available within 10m" >&2
    kubectl -n "$NAMESPACE" get pods -l "${label}" -o wide >&2
    kubectl -n "$NAMESPACE" describe "deploy/${deploy}" >&2
    kubectl -n "$NAMESPACE" describe pods -l "${label}" >&2
    local pod
    for pod in $(kubectl -n "$NAMESPACE" get pods -l "${label}" -o jsonpath='{.items[*].metadata.name}'); do
        echo "=== logs: $pod (current) ===" >&2
        kubectl -n "$NAMESPACE" logs "$pod" --all-containers --tail=200 >&2 || true
        echo "=== logs: $pod (previous) ===" >&2
        kubectl -n "$NAMESPACE" logs "$pod" --all-containers --previous --tail=200 >&2 || true
    done
    return 1
}

wait_for_matcher_import() {
    echo "Waiting for scanner-v4-matcher's initial vulnerability bundle import to complete..."
    local deadline=$((SECONDS + 1500))
    until kubectl -n "$NAMESPACE" logs deploy/scanner-v4-matcher -c matcher --tail=-1 2>/dev/null \
            | grep -q '"message":"completed update"'; do
        if (( SECONDS > deadline )); then
            kubectl -n "$NAMESPACE" logs deploy/scanner-v4-matcher -c matcher --tail=100 >&2
            die "scanner-v4-matcher did not finish its initial vulnerability import within 25m"
        fi
        sleep 15
    done
}

restart_central_and_reconnect() {
    kubectl -n "$NAMESPACE" delete pod -l app=central --grace-period=0
    kubectl -n "$NAMESPACE" rollout status deploy/central --timeout=5m

    if [[ -n "${ROXIE_PORT_FORWARD_PID:-}" ]]; then
        kill -9 "$ROXIE_PORT_FORWARD_PID" 2>/dev/null || true
    fi
    local local_port="${API_ENDPOINT##*:}"
    nohup kubectl -n "$NAMESPACE" port-forward svc/central "${local_port}:443" \
        > /tmp/central-port-forward.log 2>&1 &

    local deadline=$((SECONDS + 300))
    until roxcurl /v1/ping | grep -q ok; do
        if (( SECONDS > deadline )); then
            cat /tmp/central-port-forward.log >&2
            die "Central did not come back within 5m after operator upgrade"
        fi
        sleep 2
    done
}

wait_for_image_count_stable() {
    # Poll /v1/imagescount until it stabilizes across two consecutive checks,
    # as a proxy for the reprocessor having finished its pass.
    local previous=-1
    local stabilized=0
    local current
    local deadline=$((SECONDS + 600))
    while true; do
        current=$(roxcurl /v1/imagescount | jq -r '.count')
        echo "images count: ${current}"
        if [[ "$current" != "0" && "$current" == "$previous" ]]; then
            stabilized=1
            break
        fi
        previous="$current"
        if (( SECONDS > deadline )); then
            echo "image count did not stabilize at a nonzero value within 10m (last seen: ${current})" >&2
            break
        fi
        sleep 20
    done

    if [[ "$stabilized" != "1" ]]; then
        echo "Image reprocessing did not populate images_v2; dumping reprocessor logs" >&2
        kubectl -n "$NAMESPACE" logs deploy/central -c central --tail=500 2>&1 \
            | grep -i "reprocess" >&2 || true
        die "image count never stabilized"
    fi
}

wait_for_image_reprocessing() {
    load_envrc

    # Central's image reprocessor runs immediately on every startup, racing
    # Scanner V4's own readiness after the same operator-driven restart. A
    # failed pass doesn't retry for another 4 hours (ReprocessInterval), so
    # force one more Central restart once Scanner V4 is confirmed ready,
    # giving the reprocessor a clean run before taking the final backup.
    wait_for_deploy scanner-v4-db app=scanner-v4-db || exit 1
    wait_for_deploy scanner-v4-indexer app=scanner-v4-indexer || exit 1
    wait_for_deploy scanner-v4-matcher app=scanner-v4-matcher || exit 1

    wait_for_matcher_import
    restart_central_and_reconnect
    wait_for_image_count_stable
}

wait_for_background_migrations() {
    require_environment "RELEASE"
    require_environment "PATCH"
    load_envrc

    local target
    target=$(git show "${RELEASE}.${PATCH}:central/backgroundmigrations/seq_num.go" 2>/dev/null \
        | sed -n 's/^const CurrentBgMigrationSeqNum = \([0-9]*\)$/\1/p')

    if [[ -z "$target" ]]; then
        echo "No background migration seqnum found for ${RELEASE}.${PATCH} (feature may not exist at this version); skipping wait"
        return 0
    fi

    local db_password seqnum
    db_password=$(kubectl -n "$NAMESPACE" get secret central-db-password -o jsonpath='{.data.password}' | base64 -d)
    for _ in $(seq 1 20); do
        seqnum=$(kubectl -n "$NAMESPACE" exec deploy/central-db -c central-db -- \
            bash -c "PGPASSWORD='${db_password}' psql -U postgres -d central_active -tAc 'SELECT seqnum FROM background_migration_versions LIMIT 1'" | tr -d '[:space:]')
        [[ "$seqnum" -ge "$target" ]] 2>/dev/null && break
        sleep 15
    done
}

strip_transient_cluster_auth_config() {
    load_envrc

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
    load_envrc

    roxctl central backup --output "postgres_db_${RELEASE}.${PATCH}.sql.zip"
}

run_all() {
    deploy_previous_central
    restore_previous_backup
    add_dockerhub_integration
    upgrade_operator
    wait_for_image_reprocessing
    wait_for_background_migrations
    strip_transient_cluster_auth_config
    grab_central_backup
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        run_all
    else
        fn="$1"
        shift
        "$fn" "$@"
    fi
fi
