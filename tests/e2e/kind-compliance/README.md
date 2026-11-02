# Multi-cluster compliance on kind

Compliance e2e currently runs on OpenShift only (Prow), because the Compliance
Operator (CO) is an OpenShift operator. This directory proves that the same
StackRox compliance integration can be exercised **multi-cluster, on kind, in
GitHub Actions** — cheaply enough for a PR gate.

It stands up StackRox Central plus two secured clusters, each running the **real**
upstream Compliance Operator in its leanest mode (`PLATFORM=Generic` + CEL
`CustomRule`s). In that mode CO pulls only its operator image and evaluates CEL
rules in-process against plain Kubernetes objects, so it produces real PASS/FAIL
`ComplianceCheckResult`s on any cluster — no OpenShift, OLM, RHCOS, content image,
openscap image, service-ca, or SCC webhook required.

## Layout
- `setup-multicluster-compliance-ci.sh` — idempotent, non-interactive entrypoint
  (`up` | `verify` | `down`). `up` creates the kind cluster(s), deploys Central +
  `SECURED_COUNT` secured clusters each with a real CO, applies the CEL profile to
  each, and gates on readiness (every cluster registered in Central, CO
  integration HEALTHY, CEL profile synced). Secured cluster #1 is co-located on
  the Central cluster; the rest are separate kind clusters (`MODE=separate-kind`)
  or vclusters on the Central host (`MODE=vcluster`).
- `assets/operator-generic.yaml` — the CO operator Deployment + RBAC, run with
  `PLATFORM=Generic` (reconstructed from the bundle CSV; no OLM). Kept committed:
  see "CRD and manifest sourcing" below for why this one isn't also sourced live.
- `assets/openshift-shim/` — minimal `config.openshift.io` CRDs + singleton
  objects. These keep the operator's TLS-profile watcher happy and let Sensor
  detect the cluster as OpenShift.
- `assets/cel-profile.yaml` — a `ConfigMap` + two CEL `CustomRule`s (one passes,
  one fails) + a from-scratch `TailoredProfile` (`kind-cel-tp`). Profile only: the
  scan is driven from Central by the Go test, not by a local ScanSettingBinding.

## CRD and manifest sourcing (no vendored copies)
This repo's `go.mod` already pins `github.com/ComplianceAsCode/compliance-operator`
and `github.com/openshift/api` for compilation, and both modules ship, verbatim,
every CRD this harness needs. Rather than committing point-in-time snapshots of
those CRDs (which can silently drift from the pin — see git history for an
example), the script resolves the pinned module directories with `go list -m`
at `up` time and copies the needed files into `$WORK/sourced/`:
- `$WORK/sourced/co-crds/` — the Compliance Operator's own CRDs
  (`compliance.openshift.io_*.yaml`), from `config/crd/bases/` in the
  `compliance-operator` module. Used by `install_co()` when `CO_INSTALL=bundle`
  (the default).
- `$WORK/sourced/openshift-crds/` — the 18 `config.openshift.io`/
  `operator.openshift.io`/`security.openshift.io` CRDs `FIDELITY=platform`
  needs, from `zz_generated.crd-manifests/` in the `openshift/api` module (a
  short authored path-mapping table in the script, not CRD content, handles
  the per-type FeatureSet-variant filenames).

This requires `go` on `PATH` and the module cache to be warm (true in CI: an
earlier step already builds `roxctl`, which resolves the full module graph).

**`assets/operator-generic.yaml` stays committed.** The compliance-operator
module's own `config/manager/` + `config/rbac/` can be rendered with `kubectl
kustomize config/default`, and that output is structurally valid — but it has
no `PLATFORM` env var at all (OpenShift sets it via the OLM Subscription's
`config.env`, which this harness's `bundle` install path deliberately doesn't
use), so cleanly reproducing the `PLATFORM=Generic`/`OpenShift` and
`RELATED_IMAGE_PROFILE` override points `install_co()` depends on would need a
hand-maintained kustomize/JSON6902 patch anyway — no less fragile than the
current hand-reconstructed, hand-commented manifest, for a fraction of the
diff-bloat payoff the CRDs give. Not worth it; revisit if this ever needs to
track upstream RBAC changes more closely.

## Fidelity modes (`FIDELITY` env, default `cel`)
- **`cel`** (default; what the PR gate runs): operator image only, `PLATFORM=Generic`,
  a CEL `CustomRule` profile. Fast and deterministic, and the only way to get real
  PASS/FAIL results on kind.
- **`content`**: `PLATFORM=OpenShift`, so the operator builds the real `ocp4`/`rhcos4`
  ProfileBundles (~48 profiles / ~1166 rules) which sync to Central. Needs two kind
  shims the script adds automatically — a dummy `compliance-operator-serving-cert`
  secret (OpenShift's service-ca) and a small SCC-UID mutating webhook (OpenShift's
  SCC). **Caveat:** `ocp4`/`rhcos4` *scans* evaluate NOT-APPLICABLE on kind (no
  OpenShift API resources, no RHCOS), so content mode validates the catalog +
  profile/rule sync, not scan results. CEL remains the source of real results.
- **`platform`**: a SUPERSET of `content` (same `PLATFORM=OpenShift`, same two
  shims) that ALSO applies `assets/openshift-platform-shim/` — real
  `config.openshift.io`/`operator.openshift.io`/`security.openshift.io` CRDs plus
  ~20 singleton CRs/ConfigMaps/SCCs — so the api-resource-collector can fetch what
  the `ocp4` OVAL checks actually read. This flips `ocp4-cis` from
  NOT-APPLICABLE to **real PASS/FAIL/MANUAL results** (deterministic: a ConfigMap
  field maps directly to a rule's outcome), and gates
  `TestComplianceV2MultiClusterPlatformResults`
  (`COMPLIANCE_MC_PLATFORM=true`), which asserts real PASS *and* FAIL round-trip
  to Central for `ocp4-cis`. The content (ProfileBundle) image is pinned by digest
  via `CO_CONTENT_IMAGE` (default: a resolved digest of
  `compliance-operator-content-dev:master`) instead of a moving `:master` tag, so
  which rules exist and how they evaluate does not drift out from under the shim.
  **Maintenance caveat:** the shim is coupled to both the CRD schemas (from
  `openshift/api`) and the content version (`CO_CONTENT_IMAGE`) — see
  `assets/openshift-platform-shim/README.md` for what to re-sync when bumping
  either. `rhcos4` node scans are still out of scope (no RHCOS on kind).

In CI, `content` and `platform` are available via the workflow's manual
`workflow_dispatch` input; PR runs use `cel`.

## Install method (`CO_INSTALL` env, default `bundle`)
- **`bundle`** (default): the operator's CRDs + Deployment/RBAC are applied directly
  (reconstructed from the bundle image). No OLM — fastest and deterministic.
- **`olm`**: install upstream OLM, then a `CatalogSource` + `OperatorGroup` +
  `Subscription` — the install path OpenShift actually uses, recognizable to readers.
  Gotcha baked into the script: OLM only resolves a Subscription against catalogs in
  the Subscription's namespace or OLM's *global catalog namespace* (`olm` for the
  upstream install), so the CatalogSource goes in `olm`, not `openshift-marketplace`.
  **`olm` implies `content` fidelity**: the upstream catalog's CSV runs
  `PLATFORM=OpenShift` regardless of the Subscription env override, so the real
  ProfileBundles + serving-cert/SCC shims come along for the ride.
  Available via the workflow's `co_install` dispatch input.

## The tests
`tests/compliance_operator_v2_multicluster_test.go` (build tag `compliance`) talk
only to Central's v2 API:
- `TestComplianceV2MultiClusterIntegrations` — every secured cluster has its own
  HEALTHY CO integration, and they are distinct.
- `TestComplianceV2MultiClusterScanConfigResults` — one scan configuration fans
  out to all clusters exposing the CEL profile, and real PASS + FAIL results come
  back attributed to each cluster.
- `TestComplianceV2MultiClusterRealProfiles` — (content/platform mode) the real
  `ocp4`/`rhcos4` profile+rule catalog synced to Central for every cluster. Auto-
  skips when only the CEL profile is present.
- `TestComplianceV2MultiClusterPlatformResults` — (platform mode only, gated on
  `COMPLIANCE_MC_PLATFORM=true`) one scan configuration for profile `ocp4-cis`
  fans out to every cluster exposing it, and real PASS *and* FAIL results (not
  NOT-APPLICABLE) come back attributed to each cluster. Explicitly `t.Skip`s
  unless the env var is set, since `ocp4-cis` is all NOT-APPLICABLE without the
  OpenShift-API shim and would just time out.

All skip (not fail) when fewer than two suitable clusters are present (or, for the
real-profiles/platform-results tests, when real ProfileBundles/the shim are
absent), so they are safe in single-cluster and `cel`-mode runs.

## Run it locally
```bash
# Needs: kind, kubectl, helm, docker, jq, roxctl, openssl, go (to resolve the
# pinned CRDs; the module cache must already have been populated, e.g. by a
# prior `go build`/`go mod download` in this checkout).
export MAIN_IMAGE_TAG="$(make --quiet --no-print-directory tag)"   # or a released tag
tests/e2e/kind-compliance/setup-multicluster-compliance-ci.sh up

# Point the Go tests at the new Central and run them:
kubectl --context kind-co-ci -n stackrox port-forward svc/central 8443:443 &
export API_ENDPOINT=localhost:8443 ROX_USERNAME=admin
export ROX_ADMIN_PASSWORD="$(cat "${TMPDIR:-/tmp}/kind-co-ci/co-ci/state/admin-password")"
make -C tests compliance-v2-multicluster-tests

tests/e2e/kind-compliance/setup-multicluster-compliance-ci.sh down
```

## CI
`.github/workflows/e2e-multicluster-compliance-kind.yaml` runs the flow on GitHub
Actions: it builds `roxctl`, runs `up` (against the PR's `main`/`central-db`
images, already waited for by the shared dispatcher), then the Go tests, and
uploads diagnostics. It is a reusable workflow dispatched by
`.github/workflows/e2e-dispatch.yaml` — it only runs on a PR when the PR carries
the `e2e-multicluster-compliance-kind` label (opt-in, like the other heavier e2e
jobs), or always on push/tag builds. It can also be run directly via
`workflow_dispatch`, supplying a `tag` (and optionally `fidelity`/`co_install`).

## Known limitations / hardening TODO
- Node (`rhcos4-*`) scans and MachineConfig remediation are out of scope; kind
  nodes are not RHCOS. `ocp4-*` platform scans run but yield NOT-APPLICABLE on
  kind — CEL `CustomRule`s are how we get real results here.
- The CO operator image is a dev image pinned by digest in the workflow; mirror it
  into a StackRox-controlled registry before relying on this as a hard gate.
- Compute (not disk) is the limit when packing many clusters: Sensor/CO run per
  cluster. `collector.disable=true` is used to cut CPU/RAM for the compliance path.
