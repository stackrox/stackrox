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
#   CO_INSTALL=bundle|olm            how CO is installed (default bundle, see NOTES)
#   CO_CATALOG_IMAGE=...@sha256:...  CO_INSTALL=olm catalog image (see NOTES)
#   FIDELITY=cel|content|platform    CO fidelity (default cel, see FIDELITY NOTES below)
#   CO_CONTENT_IMAGE=...@sha256:...  FIDELITY=platform content image digest (see NOTES)
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
# How the Compliance Operator gets onto each cluster:
#   bundle - (default) reconstructed Deployment/RBAC + co-crds/, no OLM. What the
#            script has always done; fastest, no catalog/subscription resolution.
#   olm    - install upstream OLM, then a CatalogSource/OperatorGroup/Subscription
#            against CO_CATALOG_IMAGE, exercising the real install path OpenShift
#            uses. Opt-in: slower (extra images, catalog pod, CSV resolution).
#            NOTE: implies FIDELITY=content (the upstream catalog's CSV runs
#            PLATFORM=OpenShift regardless of the Subscription env override).
CO_INSTALL="${CO_INSTALL:-bundle}"
CO_CATALOG_IMAGE="${CO_CATALOG_IMAGE:-ghcr.io/complianceascode/compliance-operator-catalog@sha256:b5ad7b09f3fa12d99821b7768d6af54eba02e9328feb71c5aee7624401d5a3e8}"
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
# Fidelity of the Compliance Operator install:
#   cel      - operator image only, PLATFORM=Generic, CEL CustomRules (real
#              PASS/FAIL results on kind). Fast, deterministic. Default.
#   content  - PLATFORM=OpenShift: the operator also builds the real ocp4/rhcos4
#              ProfileBundles (~48 profiles / ~1166 rules) which sync to Central.
#              Needs two kind shims (serving-cert secret + SCC-UID webhook). Note:
#              ocp4/rhcos4 SCANS yield NOT-APPLICABLE on kind (no OpenShift API /
#              RHCOS), so content mode validates catalog+profile sync, not results.
#   platform - SUPERSET of content (same PLATFORM=OpenShift, same two shims),
#              PLUS an OpenShift-API shim (assets/openshift-platform-shim/: real
#              config.openshift.io/operator.openshift.io/security.openshift.io
#              CRDs + singleton CRs/ConfigMaps/SCCs) that lets the api-resource-
#              collector fetch what ocp4 OVAL checks read, so ocp4-* PLATFORM
#              scans (e.g. ocp4-cis) produce real PASS/FAIL/MANUAL instead of
#              NOT-APPLICABLE. Opt-in: heavier and has a content/CRD version
#              coupling caveat (see the shim's README).
FIDELITY="${FIDELITY:-cel}"
# OLM installs use the upstream catalog, whose bundled CSV effectively runs
# PLATFORM=OpenShift and ignores the Subscription's config.env override (the
# catalog's channel-head name doesn't match the bundled CSV, so installedCSV — and
# thus the env override — never applies). So an OLM install always behaves as
# content fidelity on kind (real ProfileBundles + the serving-cert/SCC shims).
[[ "$CO_INSTALL" == "olm" ]] && FIDELITY=content
CO_PLATFORM=Generic
[[ "$FIDELITY" == "content" || "$FIDELITY" == "platform" ]] && CO_PLATFORM=OpenShift
REAL_PROFILE_NAME="${REAL_PROFILE_NAME:-ocp4-cis}"  # a real profile to gate on in content/platform mode
# platform mode pins the content (ProfileBundle) image by digest for determinism:
# a moving `:master` tag would make which ocp4-cis rules exist/PASS/FAIL drift out
# from under the openshift-platform-shim assets. Default resolved once via:
#   crane digest quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-content-dev:master
CO_CONTENT_IMAGE="${CO_CONTENT_IMAGE:-quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-content-dev@sha256:c0920cbcae5c816cc50572917d2f439a7b5a5f3f493200a380e2e0f39a8cc308}"
# Deterministic workdir so `up`, `verify`, `down` share state across invocations.
WORK="${WORK:-${TMPDIR:-/tmp}/kind-co-ci/${PREFIX}}"
CENTRAL_CLUSTER="${PREFIX}"                 # kind name of the Central cluster
CENTRAL_CTX="kind-${CENTRAL_CLUSTER}"
CENTRAL_PF_PORT="${CENTRAL_PF_PORT:-18443}" # local port-forward to Central
STATE_DIR="$WORK/state"; mkdir -p "$STATE_DIR"

# Logs go to stderr so functions whose stdout is captured via $(...) are not
# corrupted by log lines.
log()  { printf '>> %s\n' "$*" >&2; }
warn() { printf '!! %s\n' "$*" >&2; }
die()  { printf 'XX %s\n' "$*" >&2; exit 1; }

require_tools() {
  local t
  for t in kind kubectl helm docker jq openssl curl "$ROXCTL"; do
    command -v "$t" >/dev/null 2>&1 || die "missing required tool: $t"
  done
  if [[ "$MODE" == "vcluster" ]]; then
    command -v vcluster >/dev/null 2>&1 || die "MODE=vcluster needs the vcluster CLI"
  fi
  [[ -d "$ASSET_DIR/co-crds" ]] || die "ASSET_DIR missing co-crds/: $ASSET_DIR"
  [[ -f "$ASSET_DIR/operator-generic.yaml" ]] || die "missing operator-generic.yaml in $ASSET_DIR"
  [[ -f "$ASSET_DIR/cel-profile.yaml" ]] || die "missing cel-profile.yaml in $ASSET_DIR"
  case "$FIDELITY" in
    cel) ;;
    content) [[ -f "$ASSET_DIR/scc-shim-webhook.py" ]] || die "content fidelity needs scc-shim-webhook.py in $ASSET_DIR" ;;
    platform)
      [[ -f "$ASSET_DIR/scc-shim-webhook.py" ]] || die "platform fidelity needs scc-shim-webhook.py in $ASSET_DIR"
      [[ -d "$ASSET_DIR/openshift-platform-shim/crds" ]] || die "platform fidelity needs openshift-platform-shim/crds/ in $ASSET_DIR"
      [[ -f "$ASSET_DIR/openshift-platform-shim/objects.yaml" ]] || die "platform fidelity needs openshift-platform-shim/objects.yaml in $ASSET_DIR"
      [[ -f "$ASSET_DIR/openshift-platform-shim/sccs.yaml" ]] || die "platform fidelity needs openshift-platform-shim/sccs.yaml in $ASSET_DIR"
      ;;
    *) die "FIDELITY must be 'cel', 'content', or 'platform' (got '$FIDELITY')" ;;
  esac
  case "$CO_INSTALL" in
    bundle|olm) ;;
    *) die "CO_INSTALL must be 'bundle' or 'olm' (got '$CO_INSTALL')" ;;
  esac
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
# content mode shim: dummy serving-cert secret (OpenShift's service-ca normally
# injects it; without it the operator os.Exit(1)s in addMetrics on OpenShift platforms).
create_serving_cert() {
  local ctx="$1"; local K="kubectl --context ${ctx}"
  log "[$ctx] creating serving-cert secret (service-ca shim)"
  openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/co-metrics.key" -out "$WORK/co-metrics.crt" \
    -days 365 -subj "/CN=metrics.${CO_NS}.svc" >/dev/null 2>&1
  $K create secret tls compliance-operator-serving-cert -n "$CO_NS" \
    --cert="$WORK/co-metrics.crt" --key="$WORK/co-metrics.key" --dry-run=client -o yaml | $K apply -f -
}

# content mode shim: a mutating webhook that injects a non-root runAsUser into pods
# in the CO namespace, emulating OpenShift SCC. Needed because the content/openscap
# scanner pods run as root with runAsNonRoot set and no runAsUser.
install_scc_webhook() {
  local ctx="$1"; local K="kubectl --context ${ctx}"; local sns=scc-shim svc=scc-shim-webhook
  log "[$ctx] deploying SCC-UID mutating webhook"
  openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/scc-ca.key" -out "$WORK/scc-ca.crt" \
    -days 3650 -subj "/CN=scc-shim-ca" >/dev/null 2>&1
  openssl genrsa -out "$WORK/scc-tls.key" 2048 >/dev/null 2>&1
  cat >"$WORK/scc-csr.cnf" <<EOF
[req]
req_extensions=v3_req
distinguished_name=dn
[dn]
[v3_req]
subjectAltName=@alt
[alt]
DNS.1=${svc}.${sns}.svc
EOF
  openssl req -new -key "$WORK/scc-tls.key" -out "$WORK/scc.csr" -subj "/CN=${svc}.${sns}.svc" -config "$WORK/scc-csr.cnf" >/dev/null 2>&1
  openssl x509 -req -in "$WORK/scc.csr" -CA "$WORK/scc-ca.crt" -CAkey "$WORK/scc-ca.key" -CAcreateserial \
    -out "$WORK/scc-tls.crt" -days 3650 -extensions v3_req -extfile "$WORK/scc-csr.cnf" >/dev/null 2>&1
  $K create namespace "$sns" --dry-run=client -o yaml | $K apply -f -
  $K create secret tls scc-shim-tls -n "$sns" --cert="$WORK/scc-tls.crt" --key="$WORK/scc-tls.key" --dry-run=client -o yaml | $K apply -f -
  $K create configmap scc-shim-code -n "$sns" --from-file=webhook.py="$ASSET_DIR/scc-shim-webhook.py" --dry-run=client -o yaml | $K apply -f -
  $K label namespace "$CO_NS" scc-shim=enabled --overwrite
  local cab; cab="$(base64 -w0 "$WORK/scc-ca.crt")"
  $K apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata: {name: ${svc}, namespace: ${sns}}
spec:
  replicas: 1
  selector: {matchLabels: {app: ${svc}}}
  template:
    metadata: {labels: {app: ${svc}}}
    spec:
      containers:
      - name: webhook
        image: python:3.12-slim
        command: ["python3","/code/webhook.py"]
        ports: [{containerPort: 8443}]
        volumeMounts: [{name: code, mountPath: /code},{name: tls, mountPath: /tls}]
        readinessProbe: {httpGet: {path: /, port: 8443, scheme: HTTPS}, initialDelaySeconds: 3}
      volumes:
      - {name: code, configMap: {name: scc-shim-code}}
      - {name: tls, secret: {secretName: scc-shim-tls}}
---
apiVersion: v1
kind: Service
metadata: {name: ${svc}, namespace: ${sns}}
spec:
  selector: {app: ${svc}}
  ports: [{port: 443, targetPort: 8443}]
---
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata: {name: scc-shim}
webhooks:
- name: scc-shim.stackrox.io
  admissionReviewVersions: ["v1"]
  sideEffects: None
  failurePolicy: Ignore
  namespaceSelector: {matchLabels: {scc-shim: enabled}}
  rules: [{apiGroups: [""], apiVersions: ["v1"], operations: ["CREATE"], resources: ["pods"], scope: Namespaced}]
  clientConfig:
    service: {name: ${svc}, namespace: ${sns}, path: /mutate}
    caBundle: ${cab}
EOF
  $K -n "$sns" rollout status "deploy/${svc}" --timeout=120s
}

# content mode: wait for the operator-created ProfileBundles to parse the content
# image and become VALID.
wait_profilebundles_valid() {
  local ctx="$1"; local K="kubectl --context ${ctx}"; local pb deadline st
  for pb in ocp4 rhcos4; do
    log "[$ctx] waiting for ProfileBundle/$pb to become VALID"
    deadline=$(( $(date +%s) + TIMEOUT_ROLLOUT ))
    while :; do
      st="$($K -n "$CO_NS" get profilebundle "$pb" -o jsonpath='{.status.dataStreamStatus}' 2>/dev/null || true)"
      [[ "$st" == VALID ]] && { log "  ProfileBundle/$pb VALID"; break; }
      (( $(date +%s) >= deadline )) && die "[$ctx] ProfileBundle/$pb did not become VALID (last: ${st:-none})"
      sleep 10
    done
  done
}

# platform mode: apply the OpenShift-API shim (real config.openshift.io/
# operator.openshift.io/security.openshift.io CRDs + singleton CRs/ConfigMaps/
# SCCs) so the api-resource-collector can fetch what ocp4 OVAL checks read.
# Must run AFTER the ocp4 ProfileBundle is VALID (profileparser already parsed
# the content) and after the real CRDs below are in place, so the SCCs (which
# depend on the real security.openshift.io CRD, not the generic co-crds one)
# validate correctly.
apply_platform_shim() {
  local ctx="$1"; local K="kubectl --context ${ctx}"
  local shim="$ASSET_DIR/openshift-platform-shim"
  log "[$ctx] applying OpenShift-API platform shim (real CRDs + objects + SCCs)"
  $K apply --server-side -f "$shim/crds"
  $K apply --server-side -f "$shim/objects.yaml"
  # The real SCC CRD requires priority/allowedCapabilities/defaultAddCapabilities/
  # volumes present even when null; `apply` strips explicit nulls and fails CRD
  # validation. `create` sends the object as-is; on a re-run (idempotency) fall
  # back to delete+recreate rather than `replace`/`apply`, which have the same
  # null-stripping behavior.
  if ! $K create -f "$shim/sccs.yaml" 2>/dev/null; then
    $K delete --ignore-not-found -f "$shim/sccs.yaml"
    $K create -f "$shim/sccs.yaml"
  fi
}

# OLM mode: install upstream OLM (idempotent) so CO can be installed via the
# normal CatalogSource/OperatorGroup/Subscription flow instead of the
# reconstructed bundle manifests.
install_olm() {
  local ctx="$1"
  local K="kubectl --context ${ctx}"
  local olm_version=v0.46.0
  local crds_url="https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${olm_version}/crds.yaml"
  local olm_url="https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${olm_version}/olm.yaml"
  if $K -n olm get deploy catalog-operator >/dev/null 2>&1; then
    log "[$ctx] OLM already installed"
    return 0
  fi
  log "[$ctx] installing OLM $olm_version"
  # --server-side is required: the clusterserviceversions CRD is too large for
  # client-side apply (exceeds the last-applied-configuration annotation limit).
  $K apply --server-side -f "$crds_url"
  $K wait --for=condition=Established -f "$crds_url"
  $K apply -f "$olm_url"
  $K -n olm rollout status deploy/olm-operator --timeout="${TIMEOUT_ROLLOUT}s"
  $K -n olm rollout status deploy/catalog-operator --timeout="${TIMEOUT_ROLLOUT}s"
}

# OLM mode: CatalogSource + OperatorGroup + Subscription for the Compliance
# Operator, then wait for the resolved CSV to succeed.
install_co_olm() {
  local ctx="$1"
  local K="kubectl --context ${ctx}"
  install_olm "$ctx"
  # The OLM-created Deployment carries nodeSelector
  # node-role.kubernetes.io/master, which kind control-plane nodes lack by
  # default; label it so the operator pod can schedule.
  $K label node -l node-role.kubernetes.io/control-plane node-role.kubernetes.io/master="" --overwrite
  log "[$ctx] installing Compliance Operator via OLM (catalog=$CO_CATALOG_IMAGE)"
  $K apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: compliance-operator
  namespace: olm
spec:
  sourceType: grpc
  image: ${CO_CATALOG_IMAGE}
  displayName: Compliance Operator (kind-compliance)
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: compliance-operator
  namespace: ${CO_NS}
spec:
  targetNamespaces:
  - ${CO_NS}
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: compliance-operator-sub
  namespace: ${CO_NS}
spec:
  channel: alpha
  name: compliance-operator
  source: compliance-operator
  sourceNamespace: olm
  installPlanApproval: Automatic
  config:
    env:
    - name: PLATFORM
      value: ${CO_PLATFORM}
EOF
  log "[$ctx] waiting for Subscription/compliance-operator-sub to resolve a CSV"
  local deadline csv phase
  deadline=$(( $(date +%s) + TIMEOUT_ROLLOUT ))
  csv=""
  while [[ -z "$csv" ]]; do
    csv="$($K -n "$CO_NS" get subscription compliance-operator-sub -o jsonpath='{.status.currentCSV}' 2>/dev/null || true)"
    # Tolerate a catalog whose declared channel head name does not match the
    # actual CSV it ships (seen in practice: status.currentCSV pointed at a CSV
    # that was never created because the bundle's embedded CSV uses a newer
    # name than the catalog's own channel/package metadata). Fall back to
    # whatever compliance-operator CSV object OLM actually installed.
    if [[ -z "$csv" ]] || ! $K -n "$CO_NS" get csv "$csv" >/dev/null 2>&1; then
      csv="$($K -n "$CO_NS" get csv -o name 2>/dev/null | sed -n 's#.*/\(compliance-operator\..*\)#\1#p' | head -1)"
    fi
    [[ -n "$csv" ]] && break
    (( $(date +%s) >= deadline )) && die "[$ctx] Subscription/compliance-operator-sub did not resolve an installable CSV"
    sleep 5
  done
  log "[$ctx] waiting for CSV/$csv to reach phase Succeeded"
  while :; do
    phase="$($K -n "$CO_NS" get csv "$csv" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    [[ "$phase" == "Succeeded" ]] && break
    (( $(date +%s) >= deadline )) && die "[$ctx] CSV/$csv did not reach Succeeded (last: ${phase:-none})"
    sleep 10
  done
  log "[$ctx] CSV/$csv Succeeded"
  $K -n "$CO_NS" rollout status deploy/compliance-operator --timeout="${TIMEOUT_ROLLOUT}s"
}

install_co() {
  local ctx="$1"
  local K="kubectl --context ${ctx}"
  log "[$ctx] installing Compliance Operator (PLATFORM=$CO_PLATFORM, fidelity=$FIDELITY, install=$CO_INSTALL)"
  $K create namespace "$CO_NS" --dry-run=client -o yaml | $K apply -f -
  if [[ "$CO_INSTALL" == "bundle" ]]; then
    $K apply --server-side -f "$ASSET_DIR/co-crds"
  fi
  $K apply --server-side -f "$ASSET_DIR"/openshift-shim/config-crds.yaml
  $K apply -f "$ASSET_DIR"/openshift-shim/config-objects.yaml
  $K patch infrastructure cluster --subresource=status --type=merge \
    -p '{"status":{"infrastructureName":"kind","controlPlaneTopology":"HighlyAvailable","platform":"None","platformStatus":{"type":"None"}}}' \
    >/dev/null 2>&1 || true
  if [[ "$FIDELITY" == "content" || "$FIDELITY" == "platform" ]]; then
    create_serving_cert "$ctx"
    install_scc_webhook "$ctx"
  fi
  if [[ "$CO_INSTALL" == "olm" ]]; then
    install_co_olm "$ctx"
  else
    # Render operator with the pinned image and the selected PLATFORM (the asset
    # ships PLATFORM=Generic; content/platform mode rewrites it to OpenShift).
    # platform mode ALSO pins RELATED_IMAGE_PROFILE to CO_CONTENT_IMAGE (a
    # digest) instead of the moving :master tag, so the default ocp4/rhcos4
    # ProfileBundles the operator creates at startup (cmd/manager/operator.go,
    # which reads this env var) are deterministic.
    local -a sed_args=(
      -e "s#image: quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-dev:master#image: ${CO_OPERATOR_IMAGE}#g"
      -e "s/^\( *\)value: Generic\$/\1value: ${CO_PLATFORM}/"
    )
    if [[ "$FIDELITY" == "platform" ]]; then
      sed_args+=(-e "s#value: quay.io/redhat-user-workloads/ocp-isc-tenant/compliance-operator-content-dev:master#value: ${CO_CONTENT_IMAGE}#g")
    fi
    sed "${sed_args[@]}" "$ASSET_DIR/operator-generic.yaml" >"$WORK/operator.rendered.yaml"
    $K apply -f "$WORK/operator.rendered.yaml"
    $K -n "$CO_NS" rollout status deploy/compliance-operator --timeout="${TIMEOUT_ROLLOUT}s"
  fi
  if [[ "$FIDELITY" == "content" || "$FIDELITY" == "platform" ]]; then
    wait_profilebundles_valid "$ctx"
  fi
  if [[ "$FIDELITY" == "platform" ]]; then
    apply_platform_shim "$ctx"
  fi
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
  # Shrink resource REQUESTS so everything co-schedules on a single kind node
  # (kind advertises the whole host's CPU, and central-db alone requests 4 CPU by
  # default, which does not fit next to central on a small CI runner). Limits are
  # left at chart defaults; actual usage with tiny CEL scans is low.
  helm --kube-context "$ctx" upgrade --install stackrox-central-services \
    "$WORK/charts/central-services" -n "$ROX_NS" --create-namespace \
    --set central.adminPassword.value="$ROX_ADMIN_PASSWORD" \
    --set scanner.disable=true --set scannerV4.disable=true \
    --set monitoring.openshift.enabled=false \
    --set imagePullSecrets.allowNone=true \
    --set central.exposure.nodePort.enabled=true \
    --set central.resources.requests.cpu=500m --set central.resources.requests.memory=1Gi \
    --set central.db.resources.requests.cpu=500m --set central.db.resources.requests.memory=1Gi \
    --set centralWorker.resources.requests.cpu=50m --set centralWorker.resources.requests.memory=128Mi \
    --set configController.resources.requests.cpu=50m --set configController.resources.requests.memory=128Mi \
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
  # central.exposure.nodePort.enabled creates a NodePort Service named
  # 'central-loadbalancer' (plain 'svc/central' is ClusterIP with no nodePort).
  port="$(kubectl --context "$CENTRAL_CTX" -n "$ROX_NS" get svc central-loadbalancer \
    -o jsonpath='{.spec.ports[?(@.port==443)].nodePort}')"
  [[ -n "$port" ]] || die "could not resolve Central NodePort"
  echo "${ip}:${port}"
}

# Path of the generated helm-format init bundle (shared across up/verify).
INIT_BUNDLE="$STATE_DIR/init-bundle-helm.yaml"

# Start a background port-forward to Central and wait until it actually answers.
# Idempotent: reuses a live port-forward recorded in pf.pid. Verifying readiness
# (instead of a blind sleep) avoids silently talking to the wrong listener if the
# port is already bound.
start_central_pf() {
  if kill -0 "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null; then return 0; fi
  kubectl --context "$CENTRAL_CTX" -n "$ROX_NS" port-forward svc/central "${CENTRAL_PF_PORT}:443" \
    >/dev/null 2>&1 &
  echo $! >"$STATE_DIR/pf.pid"
  trap 'kill "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null || true' EXIT
  local i
  for i in $(seq 1 30); do
    curl -sk "https://localhost:${CENTRAL_PF_PORT}/v1/ping" >/dev/null 2>&1 && return 0
    sleep 2
  done
  die "Central port-forward on localhost:${CENTRAL_PF_PORT} did not become ready"
}

# Generate the helm-format init bundle at $INIT_BUNDLE. Must NOT be called inside a
# command substitution: roxctl failures would be masked by set -e in $(...).
generate_init_bundle() {
  [[ -s "$INIT_BUNDLE" ]] && return 0
  log "generating init bundle (helm format) via roxctl"
  start_central_pf
  "$ROXCTL" -e "localhost:${CENTRAL_PF_PORT}" -p "$(cat "$STATE_DIR/admin-password")" \
    --insecure-skip-tls-verify \
    central init-bundles generate "ci-$(date +%s)" --output "$INIT_BUNDLE"
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
    --set sensor.resources.requests.cpu=200m --set sensor.resources.requests.memory=512Mi \
    --set admissionControl.resources.requests.cpu=50m --set admissionControl.resources.requests.memory=128Mi \
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
        if [[ "$FIDELITY" == "content" || "$FIDELITY" == "platform" ]] && ! echo "$profiles" | grep -qx "$REAL_PROFILE_NAME"; then
          : # content/platform mode: also wait for the real ocp4/rhcos4 profiles to sync
        else
          log "  cluster '$name' ($cid): CO HEALTHY, profiles synced"
          return 0
        fi
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
  generate_init_bundle
  local bundle="$INIT_BUNDLE"

  # Record name->context for verification/teardown.
  : >"$STATE_DIR/clusters.tsv"

  # secured #1 co-located on the Central cluster
  install_co "$CENTRAL_CTX"
  deploy_secured "$CENTRAL_CTX" "$(secured_name 1)" "$endpoint" "$bundle"
  apply_cel_profile "$CENTRAL_CTX"
  printf '%s\t%s\n' "$(secured_name 1)" "$CENTRAL_CTX" >>"$STATE_DIR/clusters.tsv"

  # secured #2..N on separate kinds or vclusters. Each is fully independent
  # (its own cluster, CO install, Sensor, profile), so deploy them concurrently
  # instead of one at a time to cut wall time. Every log line from install_co /
  # deploy_secured already carries a "[$ctx]" prefix, so interleaved output from
  # parallel children stays attributable; each line is a single short printf, so
  # stderr writes do not interleave mid-line.
  local i name pids=() names=()
  for (( i=2; i<=SECURED_COUNT; i++ )); do
    name="$(secured_name "$i")"
    names+=("$name")
    # Each child writes ITS OWN clusters.tsv line to a private temp file so
    # concurrent writers can never interleave-corrupt a shared file; the
    # parent concatenates them, in order, once every child has succeeded.
    rm -f "$STATE_DIR/clusters.tsv.$i"
    (
      set -e
      local ctx vport=$(( 11000 + i - 2 ))
      if [[ "$MODE" == "vcluster" ]]; then
        ctx="$(ensure_vcluster "$name" "$vport")"
      else
        ensure_kind_cluster "$name"; ctx="kind-${name}"
      fi
      install_co "$ctx"
      deploy_secured "$ctx" "$name" "$endpoint" "$bundle"
      apply_cel_profile "$ctx"
      printf '%s\t%s\n' "$name" "$ctx" >"$STATE_DIR/clusters.tsv.$i"
    ) &
    pids+=($!)
  done

  local fail=0 j
  for j in "${!pids[@]}"; do
    wait "${pids[$j]}" || { warn "parallel deploy of secured cluster '${names[$j]}' failed (pid ${pids[$j]})"; fail=1; }
  done
  (( fail == 0 )) || die "one or more secured clusters #2..N failed to deploy; see log above"

  for (( i=2; i<=SECURED_COUNT; i++ )); do
    name="$(secured_name "$i")"
    [[ -s "$STATE_DIR/clusters.tsv.$i" ]] || die "secured cluster '$name' did not record its clusters.tsv entry"
    cat "$STATE_DIR/clusters.tsv.$i" >>"$STATE_DIR/clusters.tsv"
    rm -f "$STATE_DIR/clusters.tsv.$i"
  done

  verify
}

verify() {
  [[ -s "$STATE_DIR/admin-password" ]] || die "no admin password state; run 'up' first"
  start_central_pf
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
