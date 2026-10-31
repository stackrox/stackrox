#!/usr/bin/env bash
# @language: bash
# @created: 2026-10-30
#
# CI SPIKE: stand up a multi-cluster "StackRox + Compliance Operator (CO)" env on
# kind and verify that real compliance results round-trip to one Central.
#
# This assembles the ALREADY-VERIFIED manual recipe into one parameterized,
# idempotent, non-interactive entrypoint a CI job can call. See the recipe:
#   ~/ai/library/docs/stackrox/kind-compliance-operator-stackrox.md
#   ~/ai/library/helpers/stackrox/kind-compliance-operator-setup.sh
#
# Topology (lean path = PLATFORM=Generic + CEL CustomRules, operator image only):
#   - 1 Central kind cluster that ALSO runs a secured cluster + CO (co-located to
#     save disk/CPU, exactly like the live recipe). This is secured cluster #1.
#   - SECURED_COUNT-1 ADDITIONAL secured clusters, each with its own CO, as either
#     separate kind clusters (MODE=separate-kind) or vclusters on the Central host
#     (MODE=vcluster). Each registers independently in the one Central.
#   - Each secured cluster gets a CEL CustomRule + TailoredProfile (profile only,
#     no local ScanSettingBinding) that yields real PASS/FAIL results when scanned.
#
# This script only builds the environment and gates on readiness (every secured
# cluster registered in Central, CO integration HEALTHY, CEL profile synced). The
# actual scan + per-cluster result round-trip is asserted by the Go e2e test
# tests/compliance_operator_v2_multicluster_test.go, which creates one scan
# configuration across all clusters. (Central rejects a second scan config reusing
# a profile already bound on a cluster, so the scan is owned solely by the test.)
#
# Exit codes: 0 = all clusters verified; non-zero = a step or assertion failed.
# Idempotent: re-running reconciles (kind/helm/kubectl apply are create-or-update).
#
# USAGE:
#   setup-multicluster-compliance-ci.sh up        # create everything + verify
#   setup-multicluster-compliance-ci.sh verify    # re-run verification only
#   setup-multicluster-compliance-ci.sh down       # tear down clusters we created
#
# KEY ENV (all have defaults; override in CI):
#   MODE=separate-kind|vcluster      (default separate-kind)
#   SECURED_COUNT=2                  total secured clusters (>=1)
#   PREFIX=co-ci                     cluster/context name prefix (keep unique!)
#   KIND_NODE_IMAGE=kindest/node:v1.31.2@sha256:...   PIN this in CI
#   CO_OPERATOR_IMAGE=...@sha256:... PIN the CO operator image in CI (see NOTES)
#   MAIN_IMAGE_REGISTRY=quay.io/stackrox-io   StackRox image registry
#   MAIN_IMAGE_TAG=<make tag>        StackRox image tag to test (REQUIRED in CI)
#   ROX_ADMIN_PASSWORD=<pw>          Central admin password (generated if unset)
#   ASSET_DIR=<dir>                  dir with co-crds/, openshift-shim/,
#                                    operator-generic.yaml, cel-profile.yaml
#   ROXCTL=roxctl                    roxctl binary
#   DISABLE_COLLECTOR=true           CEL path needs no collector; saves CPU/RAM
#   KEEP=false                       if true, `up` does not auto-teardown on failure
#   TIMEOUT_ROLLOUT=600  TIMEOUT_RESULTS=600  seconds
#
# REQUIRES: kind, kubectl, helm, docker, jq, roxctl, openssl; vcluster for MODE=vcluster.
set -euo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${MODE:-separate-kind}"
SECURED_COUNT="${SECURED_COUNT:-2}"
PREFIX="${PREFIX:-co-ci}"
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.31.2}"
CO_OPERATOR_IMAGE="${CO_OPERATOR_IMAGE:-quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-dev:master}"
MAIN_IMAGE_REGISTRY="${MAIN_IMAGE_REGISTRY:-quay.io/stackrox-io}"
MAIN_IMAGE_TAG="${MAIN_IMAGE_TAG:-}"
IMAGE_DEFAULTS="${IMAGE_DEFAULTS:-opensource}"
ROX_ADMIN_PASSWORD="${ROX_ADMIN_PASSWORD:-}"
ASSET_DIR="${ASSET_DIR:-$SCRIPT_DIR/assets}"
ROXCTL="${ROXCTL:-roxctl}"
DISABLE_COLLECTOR="${DISABLE_COLLECTOR:-true}"
KEEP="${KEEP:-false}"
TIMEOUT_ROLLOUT="${TIMEOUT_ROLLOUT:-600}"
TIMEOUT_RESULTS="${TIMEOUT_RESULTS:-600}"
CO_NS="openshift-compliance"
ROX_NS="stackrox"
CEL_PROFILE_NAME="${CEL_PROFILE_NAME:-kind-cel-tp}"  # TailoredProfile in cel-profile.yaml
# Deterministic workdir so `up`, `verify`, `down` share state across invocations.
WORK="${WORK:-${TMPDIR:-/tmp}/kind-co-ci/${PREFIX}}"
CENTRAL_CLUSTER="${PREFIX}"                 # kind name of the Central cluster
CENTRAL_CTX="kind-${CENTRAL_CLUSTER}"
CENTRAL_PF_PORT="${CENTRAL_PF_PORT:-18443}" # local port-forward to Central
STATE_DIR="$WORK/state"; mkdir -p "$STATE_DIR"

log()  { printf '>> %s\n' "$*"; }
warn() { printf '!! %s\n' "$*" >&2; }
die()  { printf 'XX %s\n' "$*" >&2; exit 1; }

require_tools() {
  local t
  for t in kind kubectl helm docker jq openssl "$ROXCTL"; do
    command -v "$t" >/dev/null 2>&1 || die "missing required tool: $t"
  done
  if [[ "$MODE" == "vcluster" ]]; then
    command -v vcluster >/dev/null 2>&1 || die "MODE=vcluster needs the vcluster CLI"
  fi
  [[ -d "$ASSET_DIR/co-crds" ]] || die "ASSET_DIR missing co-crds/: $ASSET_DIR"
  [[ -f "$ASSET_DIR/operator-generic.yaml" ]] || die "missing operator-generic.yaml in $ASSET_DIR"
  [[ -f "$ASSET_DIR/cel-profile.yaml" ]] || die "missing cel-profile.yaml in $ASSET_DIR"
}

# Names of the secured clusters. #1 is co-located on the Central cluster.
secured_name() { echo "${PREFIX}-sec-$1"; }

# ---------------------------------------------------------------------------
# kind cluster lifecycle
# ---------------------------------------------------------------------------
# CO + a 2nd kind cluster need a higher inotify instance limit (recipe note).
raise_inotify() {
  local want=512 cur
  cur="$(cat /proc/sys/fs/inotify/max_user_instances 2>/dev/null || echo 0)"
  if (( cur < want )); then
    sudo sysctl -w fs.inotify.max_user_instances=$want >/dev/null 2>&1 \
      || warn "could not raise fs.inotify.max_user_instances (need >= $want for multi-kind)"
  fi
}

ensure_kind_cluster() {
  local name="$1"
  if kind get clusters 2>/dev/null | grep -qx "$name"; then
    log "kind cluster '$name' already exists"
    return 0
  fi
  log "creating kind cluster '$name' (node image pinned: $KIND_NODE_IMAGE)"
  cat >"$WORK/kind-$name.yaml" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${name}
nodes:
  - role: control-plane
    image: ${KIND_NODE_IMAGE}
EOF
  kind create cluster --config "$WORK/kind-$name.yaml" --wait 120s
}

# ---------------------------------------------------------------------------
# Compliance Operator (lean: Generic + CEL, operator image only)
# ---------------------------------------------------------------------------
install_co() {
  local ctx="$1"
  local K="kubectl --context ${ctx}"
  log "[$ctx] installing Compliance Operator (PLATFORM=Generic)"
  $K create namespace "$CO_NS" --dry-run=client -o yaml | $K apply -f -
  $K apply --server-side -f "$ASSET_DIR"/co-crds/compliance.openshift.io_*.yaml
  $K apply --server-side -f "$ASSET_DIR"/openshift-shim/config-crds.yaml
  $K apply -f "$ASSET_DIR"/openshift-shim/config-objects.yaml
  $K patch infrastructure cluster --subresource=status --type=merge \
    -p '{"status":{"infrastructureName":"kind","controlPlaneTopology":"HighlyAvailable","platform":"None","platformStatus":{"type":"None"}}}' \
    >/dev/null 2>&1 || true
  # Render operator with the (pinned, overridable) operator image.
  sed "s#image: quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-dev:master#image: ${CO_OPERATOR_IMAGE}#g" \
    "$ASSET_DIR/operator-generic.yaml" >"$WORK/operator-generic.rendered.yaml"
  $K apply -f "$WORK/operator-generic.rendered.yaml"
  $K -n "$CO_NS" rollout status deploy/compliance-operator --timeout="${TIMEOUT_ROLLOUT}s"
}

apply_cel_profile() {
  local ctx="$1"
  log "[$ctx] applying CEL profile (ConfigMap + CustomRules + TailoredProfile)"
  kubectl --context "$ctx" apply -f "$ASSET_DIR/cel-profile.yaml"
}

# ---------------------------------------------------------------------------
# StackRox: Central (central cluster) + Secured (each cluster)
# ---------------------------------------------------------------------------
render_charts() {
  log "rendering StackRox helm charts via roxctl (image-defaults=$IMAGE_DEFAULTS)"
  rm -rf "$WORK/charts"; mkdir -p "$WORK/charts"
  "$ROXCTL" helm output central-services --image-defaults="$IMAGE_DEFAULTS" \
    --output-dir "$WORK/charts/central-services" --remove
  "$ROXCTL" helm output secured-cluster-services --image-defaults="$IMAGE_DEFAULTS" \
    --output-dir "$WORK/charts/secured-cluster-services" --remove
}

deploy_central() {
  local ctx="$CENTRAL_CTX"
  [[ -n "$ROX_ADMIN_PASSWORD" ]] || ROX_ADMIN_PASSWORD="$(openssl rand -base64 20)"
  echo "$ROX_ADMIN_PASSWORD" >"$STATE_DIR/admin-password"
  local -a imgargs=()
  [[ -n "$MAIN_IMAGE_TAG" ]] && imgargs+=(--set "central.image.tag=$MAIN_IMAGE_TAG" --set "central.image.registry=$MAIN_IMAGE_REGISTRY")
  log "[$ctx] deploying Central (scanners off, monitoring off, NodePort exposure)"
  helm --kube-context "$ctx" upgrade --install stackrox-central-services \
    "$WORK/charts/central-services" -n "$ROX_NS" --create-namespace \
    --set central.adminPassword.value="$ROX_ADMIN_PASSWORD" \
    --set scanner.disable=true --set scannerV4.disable=true \
    --set monitoring.openshift.enabled=false \
    --set imagePullSecrets.allowNone=true \
    --set central.exposure.nodePort.enabled=true \
    "${imgargs[@]}" \
    --wait --timeout "${TIMEOUT_ROLLOUT}s"
  kubectl --context "$ctx" -n "$ROX_NS" rollout status deploy/central --timeout="${TIMEOUT_ROLLOUT}s"
}

# Central endpoint reachable from secured clusters = Central kind node IP on the
# shared "kind" docker network + Central NodePort. Works for separate-kind AND
# vcluster (vcluster tenant pods run on the Central host nodes).
central_internal_endpoint() {
  local node_container ip port
  node_container="${CENTRAL_CLUSTER}-control-plane"
  ip="$(docker inspect -f '{{(index .NetworkSettings.Networks "kind").IPAddress}}' "$node_container")"
  [[ -n "$ip" ]] || die "could not resolve Central node IP on the kind network"
  port="$(kubectl --context "$CENTRAL_CTX" -n "$ROX_NS" get svc central \
    -o jsonpath='{.spec.ports[?(@.port==443)].nodePort}')"
  [[ -n "$port" ]] || die "could not resolve Central NodePort"
  echo "${ip}:${port}"
}

generate_init_bundle() {
  local bundle="$STATE_DIR/init-bundle-helm.yaml"
  [[ -s "$bundle" ]] && { echo "$bundle"; return 0; }
  log "generating init bundle (helm format) via roxctl"
  # roxctl talks to Central over a local port-forward.
  kubectl --context "$CENTRAL_CTX" -n "$ROX_NS" port-forward svc/central "${CENTRAL_PF_PORT}:443" \
    >/dev/null 2>&1 &
  local pf=$!; echo "$pf" >"$STATE_DIR/pf.pid"
  trap 'kill "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null || true' EXIT
  sleep 4
  "$ROXCTL" -e "localhost:${CENTRAL_PF_PORT}" -p "$(cat "$STATE_DIR/admin-password")" \
    --insecure-skip-tls-verify \
    central init-bundles generate "ci-$(date +%s)" --output "$bundle"
  echo "$bundle"
}

deploy_secured() {
  local ctx="$1" cluster_name="$2" endpoint="$3" bundle="$4"
  local -a extra=()
  [[ "$DISABLE_COLLECTOR" == "true" ]] && extra+=(--set collector.disable=true)
  local -a imgargs=()
  [[ -n "$MAIN_IMAGE_TAG" ]] && imgargs+=(--set "image.main.tag=$MAIN_IMAGE_TAG" --set "image.registry=$MAIN_IMAGE_REGISTRY")
  log "[$ctx] deploying secured-cluster-services (name=$cluster_name, central=$endpoint)"
  helm --kube-context "$ctx" upgrade --install stackrox-secured-cluster-services \
    "$WORK/charts/secured-cluster-services" -n "$ROX_NS" --create-namespace \
    -f "$bundle" \
    --set clusterName="$cluster_name" \
    --set centralEndpoint="$endpoint" \
    --set env.openshift=false \
    --set monitoring.openshift.enabled=false \
    --set scannerV4.disable=true \
    --set imagePullSecrets.allowNone=true \
    "${extra[@]}" "${imgargs[@]}" \
    --wait --timeout "${TIMEOUT_ROLLOUT}s"
  kubectl --context "$ctx" -n "$ROX_NS" rollout status deploy/sensor --timeout="${TIMEOUT_ROLLOUT}s"
}

# ---------------------------------------------------------------------------
# vcluster helpers (MODE=vcluster): each secured cluster is a vcluster on Central host
# ---------------------------------------------------------------------------
ensure_vcluster() {
  local name="$1" port="$2"
  if vcluster list --context "$CENTRAL_CTX" 2>/dev/null | grep -q "\b${name}\b"; then
    log "vcluster '$name' already exists"
  else
    log "creating vcluster '$name' on $CENTRAL_CTX"
    vcluster create "$name" --namespace "$name" --context "$CENTRAL_CTX" \
      --driver helm --connect=false
  fi
  vcluster connect "$name" --namespace "$name" --context "$CENTRAL_CTX" \
    --local-port "$port" --background-proxy >/dev/null 2>&1 || \
  vcluster connect "$name" --namespace "$name" --context "$CENTRAL_CTX" \
    --local-port "$port" >/dev/null 2>&1 &
  sleep 6
  echo "vcluster_${name}_${name}_${CENTRAL_CTX}"
}

# ---------------------------------------------------------------------------
# Verification: results round-trip to Central v2 API
# ---------------------------------------------------------------------------
central_api() {
  # $1 = METHOD, $2 = path, $3 = optional json body
  local method="$1" path="$2" body="${3:-}"
  local -a curlargs=(-sk -u "admin:$(cat "$STATE_DIR/admin-password")"
    -X "$method" "https://localhost:${CENTRAL_PF_PORT}${path}")
  [[ -n "$body" ]] && curlargs+=(-H 'Content-Type: application/json' -d "$body")
  curl "${curlargs[@]}"
}

cluster_id_for() {
  local name="$1"
  central_api GET /v1/clusters | jq -r \
    --arg n "$name" '.clusters[] | select(.name==$n) | .id' | head -1
}

# Readiness gate for one secured cluster: it is registered in Central, its CO
# integration is HEALTHY, and the CEL profile has synced to Central. The actual
# scan + per-cluster result round-trip is asserted by the Go e2e test
# (tests/compliance_operator_v2_multicluster_test.go), which creates one scan
# configuration across all clusters. We deliberately do NOT create a scan here:
# Central rejects a second scan configuration reusing a profile already bound on a
# cluster, so a local scan would collide with the Go test's.
cluster_ready() {
  local name="$1" deadline cid status profiles
  log "waiting for cluster '$name' to be compliance-ready in Central"
  deadline=$(( $(date +%s) + TIMEOUT_RESULTS ))
  while :; do
    cid="$(cluster_id_for "$name" 2>/dev/null || true)"
    if [[ -n "$cid" ]]; then
      status="$(central_api GET /v2/compliance/integrations 2>/dev/null \
        | jq -r --arg c "$cid" '.integrations[]? | select(.clusterId==$c) | .status' | head -1)"
      profiles="$(central_api GET "/v2/compliance/profiles/cluster?clusterId=$cid" 2>/dev/null \
        | jq -r '.profiles[]?.name' 2>/dev/null)"
      if [[ "$status" == "HEALTHY" ]] && echo "$profiles" | grep -qx "$CEL_PROFILE_NAME"; then
        log "  cluster '$name' ($cid): CO HEALTHY, profile '$CEL_PROFILE_NAME' synced"
        return 0
      fi
    fi
    (( $(date +%s) >= deadline )) && { warn "cluster '$name' not ready (cid=${cid:-none}, CO=${status:-none})"; return 1; }
    sleep 10
  done
}

# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------
up() {
  require_tools
  [[ "$SECURED_COUNT" -ge 1 ]] || die "SECURED_COUNT must be >= 1"
  raise_inotify

  # 1. Central cluster (also hosts secured #1 + CO).
  ensure_kind_cluster "$CENTRAL_CLUSTER"
  render_charts
  deploy_central
  local endpoint; endpoint="$(central_internal_endpoint)"
  log "Central internal endpoint for secured clusters: $endpoint"
  local bundle; bundle="$(generate_init_bundle)"

  # Record name->context for verification/teardown.
  : >"$STATE_DIR/clusters.tsv"

  # secured #1 co-located on the Central cluster
  install_co "$CENTRAL_CTX"
  deploy_secured "$CENTRAL_CTX" "$(secured_name 1)" "$endpoint" "$bundle"
  apply_cel_profile "$CENTRAL_CTX"
  printf '%s\t%s\n' "$(secured_name 1)" "$CENTRAL_CTX" >>"$STATE_DIR/clusters.tsv"

  # secured #2..N on separate kinds or vclusters
  local i ctx vport=11000
  for (( i=2; i<=SECURED_COUNT; i++ )); do
    local name; name="$(secured_name "$i")"
    if [[ "$MODE" == "vcluster" ]]; then
      ctx="$(ensure_vcluster "$name" "$vport")"; vport=$((vport+1))
    else
      ensure_kind_cluster "$name"; ctx="kind-${name}"
    fi
    install_co "$ctx"
    deploy_secured "$ctx" "$name" "$endpoint" "$bundle"
    apply_cel_profile "$ctx"
    printf '%s\t%s\n' "$name" "$ctx" >>"$STATE_DIR/clusters.tsv"
  done

  verify
}

verify() {
  [[ -s "$STATE_DIR/admin-password" ]] || die "no admin password state; run 'up' first"
  # ensure a port-forward exists for the API calls
  if ! kill -0 "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null; then
    kubectl --context "$CENTRAL_CTX" -n "$ROX_NS" port-forward svc/central "${CENTRAL_PF_PORT}:443" >/dev/null 2>&1 &
    echo $! >"$STATE_DIR/pf.pid"; sleep 4
  fi
  local rc=0 name ctx
  echo '{"mode":"'"$MODE"'","clusters":[]}' >"$STATE_DIR/proof.json"
  # Prefer the recorded name->context map; fall back to derived names/contexts.
  if [[ ! -s "$STATE_DIR/clusters.tsv" ]]; then
    : >"$STATE_DIR/clusters.tsv"
    local i
    for (( i=1; i<=SECURED_COUNT; i++ )); do
      name="$(secured_name "$i")"
      if (( i == 1 )); then ctx="$CENTRAL_CTX"; else ctx="kind-${name}"; fi
      printf '%s\t%s\n' "$name" "$ctx" >>"$STATE_DIR/clusters.tsv"
    done
  fi
  while IFS=$'\t' read -r name ctx; do
    [[ -z "$name" ]] && continue
    if cluster_ready "$name"; then
      jq --arg n "$name" '.clusters += [{"name":$n,"ready":true}]' "$STATE_DIR/proof.json" >"$STATE_DIR/proof.json.tmp"
    else
      rc=1
      jq --arg n "$name" '.clusters += [{"name":$n,"ready":false}]' "$STATE_DIR/proof.json" >"$STATE_DIR/proof.json.tmp"
    fi
    mv "$STATE_DIR/proof.json.tmp" "$STATE_DIR/proof.json"
  done <"$STATE_DIR/clusters.tsv"
  log "proof written: $STATE_DIR/proof.json"; cat "$STATE_DIR/proof.json"
  if (( rc != 0 )); then
    warn "readiness FAILED for one or more clusters"
    [[ "$KEEP" == "true" ]] || warn "set KEEP=true to keep clusters for debugging"
    return 1
  fi
  log "ALL secured clusters compliance-ready in Central (run the Go e2e test next)"
}

down() {
  log "tearing down clusters created with prefix '$PREFIX' (mode=$MODE)"
  kill "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null || true
  local i
  if [[ "$MODE" == "vcluster" ]]; then
    for (( i=2; i<=SECURED_COUNT; i++ )); do
      local name; name="$(secured_name "$i")"
      vcluster delete "$name" --namespace "$name" --context "$CENTRAL_CTX" 2>/dev/null || true
    done
  else
    for (( i=2; i<=SECURED_COUNT; i++ )); do
      kind delete cluster --name "$(secured_name "$i")" 2>/dev/null || true
    done
  fi
  kind delete cluster --name "$CENTRAL_CLUSTER" 2>/dev/null || true
  log "teardown complete"
}

main() {
  local cmd="${1:-up}"
  case "$cmd" in
    up)     up ;;
    verify) verify ;;
    down)   down ;;
    *) die "unknown command '$cmd' (use: up | verify | down)" ;;
  esac
}
main "$@"
