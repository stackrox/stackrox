#!/usr/bin/env bash
# @language: bash
# @created: 2026-10-30
#
# FRAMEWORK: stand up StackRox Central + N secured clusters on kind, so
# suites that need MORE THAN ONE cluster have somewhere to run. Prow only
# ever gives us a single OpenShift cluster, so multi-cluster scenarios
# (cross-cluster fan-out, per-cluster attribution, N independent CO
# integrations, ...) have never had anywhere to run as an automated test.
# This script is generic infrastructure: it only creates kind clusters,
# installs a real Compliance Operator (CO) on each, and wires them all into
# one Central. It does not know about any specific test suite.
#
# Compliance is the FIRST user (see tests/compliance_operator_v2_multicluster_test.go),
# chosen because a verified manual recipe already existed for it (see NOTES
# below). Other suites that need N real clusters can reuse this same harness.
#
# Topology:
#   - 1 Central kind cluster that ALSO runs a secured cluster + CO (co-located
#     to save disk/CPU). This is secured cluster #1.
#   - SECURED_COUNT-1 ADDITIONAL secured clusters, each its own kind cluster
#     with its own CO, deployed in parallel. Each registers independently in
#     the one Central.
#   - Every secured cluster's CO always runs full OpenShift fidelity:
#     PLATFORM=OpenShift (so the operator builds the real ocp4/rhcos4
#     ProfileBundles), a serving-cert shim (OpenShift's service-ca normally
#     injects it), an SCC-UID mutating webhook (emulating OpenShift SCC for
#     the openscap scanner pods), and an OpenShift-API shim
#     (assets/openshift-platform-shim/: real config.openshift.io/
#     operator.openshift.io/security.openshift.io CRDs plus ~20 singleton
#     CRs/ConfigMaps/SCCs) so the api-resource-collector can fetch what the
#     ocp4 OVAL checks read -- without it, ocp4-* PLATFORM scans come back
#     NOT-APPLICABLE instead of real PASS/FAIL. CO is installed via OLM
#     (CatalogSource + OperatorGroup + Subscription, in the same namespace
#     layout OpenShift itself uses), not a hand-reconstructed manifest: see
#     install_olm/install_co_olm below.
#
# This script only builds the environment and gates on readiness (every
# secured cluster registered in Central, CO integration HEALTHY, the real
# ocp4-cis profile synced). The actual scan + per-cluster result round-trip
# is asserted by the Go e2e test, which creates ONE scan configuration across
# all clusters (see tests/compliance_operator_v2_multicluster_test.go, run via
# `make -C tests compliance-v2-multicluster-tests`; see
# .github/workflows/e2e-multicluster-compliance-kind.yaml).
#
# Exit codes: 0 = all clusters verified; non-zero = a step or assertion failed.
# Idempotent: re-running reconciles (kind/helm/kubectl apply are create-or-update).
#
# USAGE:
#   setup-multicluster-compliance-ci.sh up        # create everything + verify
#   setup-multicluster-compliance-ci.sh verify    # re-run verification only
#   setup-multicluster-compliance-ci.sh down      # tear down clusters we created
#
# KEY ENV (all have defaults; override in CI):
#   SECURED_COUNT=2                  total secured clusters (>=1)
#   PREFIX=co-ci                     cluster/context name prefix (keep unique!)
#   KIND_NODE_IMAGE=kindest/node:v1.31.2@sha256:...   PIN this in CI
#   CO_CATALOG_IMAGE=...@sha256:...  PIN the CO catalog image in CI (see NOTES).
#                                    The operator (CSV) and content (ProfileBundle)
#                                    images both come from this catalog, so pinning
#                                    it by digest is what makes the whole install
#                                    deterministic -- there is no separate operator-
#                                    or content-image pin anymore.
#   MAIN_IMAGE_REGISTRY=quay.io/stackrox-io   StackRox image registry
#   MAIN_IMAGE_TAG=<make tag>        StackRox image tag to test (REQUIRED in CI)
#   ROX_ADMIN_PASSWORD=<pw>          Central admin password (generated if unset)
#   ASSET_DIR=<dir>                  dir with openshift-shim/,
#                                    openshift-platform-shim/, scc-shim-webhook.py
#                                    (the openshift/api platform CRDs are NOT
#                                    here: they are sourced at setup time from
#                                    the pinned go.mod dep, see
#                                    stage_openshift_crds below; the Compliance
#                                    Operator's own CRDs come from OLM's catalog
#                                    bundle, not from this repo at all)
#   ROXCTL=roxctl                    roxctl binary
#   DISABLE_COLLECTOR=true           the suite needs no Collector; saves CPU/RAM
#   KEEP=false                       if true, `up` does not auto-teardown on failure
#   TIMEOUT_ROLLOUT=600  TIMEOUT_RESULTS=600  seconds
#
# REQUIRES: kind, kubectl, helm, docker, jq, roxctl, openssl, go (to resolve the
#           openshift/api platform CRDs from pinned go.mod deps, see "CRD
#           sourcing" below).
#
# Compliance Operator install (OLM, the same way OpenShift installs it): each
# cluster gets upstream OLM (CatalogSource/OperatorGroup/Subscription CRDs +
# the olm-operator/catalog-operator controllers), then a CatalogSource in OLM's
# global catalog namespace (olm; OLM does not resolve Subscriptions against
# CatalogSources outside that namespace) pointing at CO_CATALOG_IMAGE, plus an
# OperatorGroup + Subscription in openshift-compliance. The operator
# Deployment, its RBAC, and its own CRDs all come from the catalog's CSV --
# none of it is hand-authored here anymore (this replaced a committed,
# hand-reconstructed assets/operator-generic.yaml, ~1,768 lines of Deployment +
# RBAC YAML). Content-image determinism is a side effect of pinning
# CO_CATALOG_IMAGE by digest: the catalog's CSV ignores the Subscription's
# config.env PLATFORM/RELATED_IMAGE_PROFILE overrides (a known channel-head/CSV
# name mismatch in the upstream catalog), so whatever operator+content image
# pair that CSV bundles is what runs; moving the catalog digest is the only way
# to intentionally change either.
#
# CRD sourcing (no vendored copies): the openshift/api platform CRDs are NOT
# committed under assets/. They are already shipped, byte-for-byte, inside the
# github.com/openshift/api module this repo's go.mod already pins for
# compilation. At `up` time this script resolves that module directory with
# `go list -m` and copies the needed files into $WORK/sourced/openshift-crds/,
# which is what gets applied. This makes the pinned go.mod version the single
# source of truth (no separate snapshot to fall out of sync), at the cost of
# requiring `go` + the module cache to be present (true in CI: an earlier step
# already builds roxctl, which resolves the full module graph).
#
# See tests/e2e/kind-compliance/README.md for the framework framing and the
# intended path for other suites to adopt this harness.
set -euo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Root of the stackrox/stackrox checkout (tests/e2e/kind-compliance -> repo root),
# resolved once, up front: `go list -m` (CRD sourcing, see below) must run with
# this repo's go.mod/go.sum in scope, not wherever a later `cd` might land.
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SECURED_COUNT="${SECURED_COUNT:-2}"
PREFIX="${PREFIX:-co-ci}"
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.31.2}"
# Pinned by digest: the CSV this catalog ships bundles both the operator image
# and the RELATED_IMAGE_PROFILE (content/ProfileBundle) image, so this is the
# single knob that determines both (see the OLM install note above).
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
# A real profile to gate readiness on, once it has synced to Central.
REAL_PROFILE_NAME="${REAL_PROFILE_NAME:-ocp4-cis}"
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
  # `go` is required unconditionally: the openshift/api platform-shim CRDs are
  # sourced from its pinned go.mod module (see stage_openshift_crds), not from
  # a committed copy. The Compliance Operator's own CRDs come from OLM's
  # catalog bundle at install time, so they need no local tool support.
  for t in kind kubectl helm docker jq openssl curl go "$ROXCTL"; do
    command -v "$t" >/dev/null 2>&1 || die "missing required tool: $t"
  done
  [[ -f "$ASSET_DIR/scc-shim-webhook.py" ]] || die "missing scc-shim-webhook.py in $ASSET_DIR"
  [[ -f "$ASSET_DIR/openshift-platform-shim/objects.yaml" ]] || die "missing openshift-platform-shim/objects.yaml in $ASSET_DIR"
  [[ -f "$ASSET_DIR/openshift-platform-shim/sccs.yaml" ]] || die "missing openshift-platform-shim/sccs.yaml in $ASSET_DIR"
}

# ---------------------------------------------------------------------------
# CRD sourcing: resolve the pinned go.mod dep that already ships, verbatim,
# every openshift/api CRD the platform shim needs (see the "CRD sourcing"
# header note above). This is the only place the script reads from the module
# cache; everything else below reads from $WORK/sourced/ like it used to read
# from a committed asset. The Compliance Operator's own CRDs are no longer
# sourced here: OLM installs them from the catalog's CSV bundle.
# ---------------------------------------------------------------------------
OPENSHIFT_API_MODULE="github.com/openshift/api"
OPENSHIFT_API_DIR=""

# Short-name -> path of the CRD manifest inside the openshift/api module, for
# the 18 OpenShift CRDs the platform shim needs. Several config/v1 and
# operator/v1 types ship one manifest per FeatureSet (e.g. "-Default.crd.yaml",
# "-CustomNoUpgrade.crd.yaml", "-OKD.crd.yaml"); we always want "-Default" (the
# standard self-managed featureset). Types without a featureset ship a single
# file. This table is the only thing authored here: it is a lookup of file
# paths, not a copy of CRD content, and only needs to change if openshift/api
# restructures its zz_generated.crd-manifests/ layout (rare).
OPENSHIFT_CRD_MAP=(
  "apiservers.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_apiservers-Default.crd.yaml"
  "authentications.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_authentications-Default.crd.yaml"
  "clusteroperators.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_00_cluster-version-operator_01_clusteroperators.crd.yaml"
  "clusterversions.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_00_cluster-version-operator_01_clusterversions-Default.crd.yaml"
  "consoles.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_consoles.crd.yaml"
  "featuregates.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_featuregates.crd.yaml"
  "images.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_images.crd.yaml"
  "infrastructures.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_infrastructures-Default.crd.yaml"
  "ingresses.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_ingresses.crd.yaml"
  "networks.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_networks.crd.yaml"
  "oauths.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_oauths.crd.yaml"
  "proxies.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_03_config-operator_01_proxies.crd.yaml"
  "schedulers.config.openshift.io=config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_schedulers-SelfManagedHA-Default.crd.yaml"
  "ingresscontrollers.operator.openshift.io=operator/v1/zz_generated.crd-manifests/0000_50_ingress_00_ingresscontrollers.crd.yaml"
  "kubeapiservers.operator.openshift.io=operator/v1/zz_generated.crd-manifests/0000_20_kube-apiserver_01_kubeapiservers-Default.crd.yaml"
  "networks.operator.openshift.io=operator/v1/zz_generated.crd-manifests/0000_70_network_01_networks.crd.yaml"
  "openshiftapiservers.operator.openshift.io=operator/v1/zz_generated.crd-manifests/0000_30_openshift-apiserver_01_openshiftapiservers.crd.yaml"
  "securitycontextconstraints.security.openshift.io=security/v1/zz_generated.crd-manifests/0000_03_config-operator_01_securitycontextconstraints.crd.yaml"
)

# Resolve $OPENSHIFT_API_DIR once, from $REPO_ROOT (not the process's current
# directory, which nothing here changes, but which a future edit should not be
# allowed to invalidate this). `go -C` avoids a `cd` entirely. Safe/cheap to
# call repeatedly: `go mod download` and `go list -m` are no-ops once the
# module cache is warm (already true in CI: an earlier step builds roxctl,
# which resolves the full module graph).
resolve_go_module_dirs() {
  [[ -n "$OPENSHIFT_API_DIR" ]] && return 0
  log "resolving pinned go.mod module dir for CRD sourcing ($OPENSHIFT_API_MODULE)"
  go -C "$REPO_ROOT" mod download "$OPENSHIFT_API_MODULE" \
    || die "go mod download failed for $OPENSHIFT_API_MODULE"
  OPENSHIFT_API_DIR="$(go -C "$REPO_ROOT" list -m -f '{{.Dir}}' "$OPENSHIFT_API_MODULE")"
  [[ -d "$OPENSHIFT_API_DIR" ]] || die "could not resolve module dir for $OPENSHIFT_API_MODULE"
}

# Stage the 18 openshift/api platform CRDs into $WORK/sourced/openshift-crds/,
# named exactly as the (now-deleted) committed openshift-platform-shim/crds/
# directory named them.
stage_openshift_crds() {
  resolve_go_module_dirs
  local dest="$WORK/sourced/openshift-crds" entry name relpath src
  rm -rf "$dest"; mkdir -p "$dest"
  for entry in "${OPENSHIFT_CRD_MAP[@]}"; do
    name="${entry%%=*}"
    relpath="${entry#*=}"
    src="$OPENSHIFT_API_DIR/$relpath"
    [[ -f "$src" ]] || die "openshift/api module ($OPENSHIFT_API_DIR) missing expected CRD manifest: $relpath (module layout changed? update OPENSHIFT_CRD_MAP)"
    cp "$src" "$dest/$name.crd.yaml"
  done
  log "staged ${#OPENSHIFT_CRD_MAP[@]} OpenShift CRDs from $OPENSHIFT_API_MODULE into $dest"
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
# Compliance Operator (always full OpenShift fidelity, installed via OLM)
# ---------------------------------------------------------------------------
# dummy serving-cert secret (OpenShift's service-ca normally injects it;
# without it the operator os.Exit(1)s in addMetrics on OpenShift platforms).
create_serving_cert() {
  local ctx="$1"; local K="kubectl --context ${ctx}"
  log "[$ctx] creating serving-cert secret (service-ca shim)"
  openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/co-metrics.key" -out "$WORK/co-metrics.crt" \
    -days 365 -subj "/CN=metrics.${CO_NS}.svc" >/dev/null 2>&1
  $K create secret tls compliance-operator-serving-cert -n "$CO_NS" \
    --cert="$WORK/co-metrics.crt" --key="$WORK/co-metrics.key" --dry-run=client -o yaml | $K apply -f -
}

# a mutating webhook that injects a non-root runAsUser into pods in the CO
# namespace, emulating OpenShift SCC. Needed because the content/openscap
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

# wait for the operator-created ProfileBundles to parse the content image and
# become VALID.
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

# apply the OpenShift-API shim (real config.openshift.io/operator.openshift.io/
# security.openshift.io CRDs + singleton CRs/ConfigMaps/SCCs) so the
# api-resource-collector can fetch what ocp4 OVAL checks read. Must run AFTER
# the ocp4 ProfileBundle is VALID (profileparser already parsed the content)
# and after the real CRDs below are in place, so the SCCs (which depend on the
# real security.openshift.io CRD, not the generic placeholder one) validate
# correctly.
apply_platform_shim() {
  local ctx="$1"; local K="kubectl --context ${ctx}"
  local shim="$ASSET_DIR/openshift-platform-shim"
  log "[$ctx] applying OpenShift-API platform shim (real CRDs + objects + SCCs)"
  $K apply --server-side -f "$WORK/sourced/openshift-crds"
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

# Install upstream OLM (idempotent per cluster) so CO can be installed via the
# normal CatalogSource/OperatorGroup/Subscription flow -- the same way
# OpenShift itself installs operators -- instead of a hand-reconstructed
# bundle manifest.
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

# Install the Compliance Operator via OLM: a CatalogSource in OLM's global
# catalog namespace (olm -- NOT openshift-marketplace; OLM only resolves
# Subscriptions against CatalogSources in its own namespace, so putting it
# anywhere else leaves the Subscription stuck unresolved), an OperatorGroup +
# Subscription in $CO_NS, then wait for the resolved CSV to succeed and the
# operator Deployment to roll out. The operator Deployment, its RBAC, and its
# own CRDs all come from this catalog's CSV bundle.
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
      value: OpenShift
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
  log "[$ctx] installing Compliance Operator (PLATFORM=OpenShift, via OLM)"
  $K create namespace "$CO_NS" --dry-run=client -o yaml | $K apply -f -
  $K apply --server-side -f "$ASSET_DIR"/openshift-shim/config-crds.yaml
  $K apply -f "$ASSET_DIR"/openshift-shim/config-objects.yaml
  $K patch infrastructure cluster --subresource=status --type=merge \
    -p '{"status":{"infrastructureName":"kind","controlPlaneTopology":"HighlyAvailable","platform":"None","platformStatus":{"type":"None"}}}' \
    >/dev/null 2>&1 || true
  create_serving_cert "$ctx"
  install_scc_webhook "$ctx"
  install_co_olm "$ctx"
  wait_profilebundles_valid "$ctx"
  apply_platform_shim "$ctx"
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
  # left at chart defaults; actual usage with tiny scans is low.
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
# shared "kind" docker network + Central NodePort.
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
# Verification: the secured clusters are registered + compliance-ready in Central
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
# integration is HEALTHY, and the real ocp4-cis profile has synced to Central.
# The actual scan + per-cluster result round-trip is asserted by the Go e2e
# test (tests/compliance_operator_v2_multicluster_test.go), which creates one
# scan configuration across all clusters. We deliberately do NOT create a scan
# here: Central rejects a second scan configuration reusing a profile already
# bound on a cluster, so a local scan would collide with the Go test's.
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
      if [[ "$status" == "HEALTHY" ]] && echo "$profiles" | grep -qx "$REAL_PROFILE_NAME"; then
        log "  cluster '$name' ($cid): CO HEALTHY, profiles synced"
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

  # Stage the openshift/api CRDs sourced from the pinned go.mod dep ONCE,
  # before any secured cluster (including parallel ones below) calls
  # apply_platform_shim.
  stage_openshift_crds

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
  printf '%s\t%s\n' "$(secured_name 1)" "$CENTRAL_CTX" >>"$STATE_DIR/clusters.tsv"

  # secured #2..N on separate kind clusters. Each is fully independent (its
  # own cluster, CO install, Sensor), so deploy them concurrently instead of
  # one at a time to cut wall time. Every log line from install_co /
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
      local ctx
      ensure_kind_cluster "$name"; ctx="kind-${name}"
      install_co "$ctx"
      deploy_secured "$ctx" "$name" "$endpoint" "$bundle"
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
  echo '{"clusters":[]}' >"$STATE_DIR/proof.json"
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
  log "tearing down clusters created with prefix '$PREFIX'"
  kill "$(cat "$STATE_DIR/pf.pid" 2>/dev/null)" 2>/dev/null || true
  local i
  for (( i=2; i<=SECURED_COUNT; i++ )); do
    kind delete cluster --name "$(secured_name "$i")" 2>/dev/null || true
  done
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
