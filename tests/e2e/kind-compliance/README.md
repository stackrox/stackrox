# Multi-cluster e2e framework on kind (PoC) — compliance is the first user

Multi-cluster e2e tests don't exist in this repo today: Prow only ever
provisions a **single** OpenShift cluster per job, so any scenario that needs
more than one real cluster — cross-cluster fan-out, N independent integrations,
per-cluster attribution — has had nowhere to run as an automated test.

This directory is a **reusable kind framework** that stands up StackRox
Central plus **N secured clusters**, each running a real upstream Compliance
Operator (CO) at full OpenShift fidelity (`PLATFORM=OpenShift`, real
`ocp4`/`rhcos4` ProfileBundles, plus an OpenShift-API shim so `ocp4-*` checks
produce real PASS/FAIL instead of NOT-APPLICABLE). **Compliance is the first
user of the framework**, not its only intended one: the harness
(`setup-multicluster-compliance-ci.sh`) only creates clusters and installs CO —
it has no compliance-specific assertions baked in — so other suites that need
several real clusters can stand up on the same scaffolding.

## What the included test proves
`tests/compliance_operator_v2_multicluster_test.go`
(`TestComplianceV2MultiClusterScanFanout`, build tag `compliance`, run via
`make -C tests compliance-v2-multicluster-tests`) exercises a scenario that is
**impossible to test today** without this framework: it creates ONE
`ComplianceScanConfiguration` targeting every secured cluster and verifies
that (1) Central tracks a distinct, HEALTHY CO integration per cluster, (2)
the scan configuration fans out its status to all of them, and (3) real
per-cluster PASS/FAIL results round-trip and are attributed to the correct
cluster. No single-cluster/kind test, and no Prow job, can cover cross-cluster
fan-out, because there is no second cluster to fan out to.

## Layout
- `setup-multicluster-compliance-ci.sh` — idempotent, non-interactive
  entrypoint (`up` | `verify` | `down`). `up` creates the Central kind
  cluster plus `SECURED_COUNT-1` additional secured kind clusters (deployed in
  parallel), installs a real CO on each, deploys Central + Sensor, and gates
  on readiness (every cluster registered in Central, CO integration HEALTHY, a
  real profile synced). The actual test assertions are the Go test itself.
  CO itself is installed via **OLM** (`install_olm`/`install_co_olm`): upstream
  OLM, then a CatalogSource/OperatorGroup/Subscription against
  `CO_CATALOG_IMAGE` — the same mechanism OpenShift uses to install operators.
  There is no hand-reconstructed operator manifest in this repo anymore (this
  replaced a committed `assets/operator-generic.yaml` Deployment+RBAC dump,
  ~1,768 lines, removed in full).
- `assets/openshift-shim/` — minimal `config.openshift.io` CRDs + singleton
  objects, applied before the operator starts. These keep the operator's
  TLS-profile watcher happy and let Sensor detect the cluster as OpenShift,
  ahead of the real, sourced CRDs the platform shim applies later.
- `assets/openshift-platform-shim/` — real `config.openshift.io`/
  `operator.openshift.io`/`security.openshift.io` CRDs (sourced, see below)
  plus ~20 singleton CRs/ConfigMaps/SCCs (`objects.yaml`, `sccs.yaml`), so the
  api-resource-collector can fetch what the `ocp4` OVAL checks actually read.
  This flips `ocp4-cis` from NOT-APPLICABLE to **real PASS/FAIL/MANUAL
  results**. See `assets/openshift-platform-shim/README.md` for what to
  re-sync when bumping the CRD or content pin.
- `assets/scc-shim-webhook.py` — a small mutating webhook that injects a
  non-root `runAsUser` into CO's scanner pods, emulating OpenShift SCC.

## Compliance Operator install: OLM, not a hand-reconstructed manifest
CO is installed the same way OpenShift installs it: a CatalogSource +
OperatorGroup + Subscription against upstream OLM
(`install_olm`/`install_co_olm` in the script), pointed at `CO_CATALOG_IMAGE`.
The operator Deployment, its RBAC, and its own CRDs all come from that
catalog's CSV bundle — none of it is authored or committed in this repo. This
replaced a committed, hand-reconstructed `assets/operator-generic.yaml`
(1 Deployment + 29 RBAC objects, ~1,768 lines) that had to be kept in sync by
hand with the compliance-operator module's `config/manager/`/`config/rbac/`.

**Content-image determinism caveat:** the upstream catalog's CSV ignores the
Subscription's `config.env` `PLATFORM`/`RELATED_IMAGE_PROFILE` overrides (a
known mismatch between the catalog's declared channel head and the CSV it
actually ships), so neither the operator PLATFORM mode nor the ProfileBundle
content image can be overridden per-run. Both come from whatever the pinned
`CO_CATALOG_IMAGE` digest bundles. That digest is the only way to change
either value intentionally — bump it, don't try to override it via the
Subscription.

## CRD sourcing (no vendored copies)
This repo's `go.mod` already pins `github.com/openshift/api` for compilation,
and it ships, verbatim, every CRD the platform shim needs. Rather than
committing a point-in-time snapshot of those CRDs (which can silently drift
from the pin), the script resolves the pinned module directory with
`go list -m` at `up` time and copies the needed files into
`$WORK/sourced/openshift-crds/` — the 18 `config.openshift.io`/
`operator.openshift.io`/`security.openshift.io` CRDs the platform shim needs,
from `zz_generated.crd-manifests/` in the `openshift/api` module (a short
authored path-mapping table in the script, not CRD content, handles the
per-type FeatureSet-variant filenames). The Compliance Operator's own CRDs are
no longer sourced this way: OLM installs them from the catalog's CSV bundle.

This requires `go` on `PATH` and the module cache to be warm (true in CI: an
earlier step already builds `roxctl`, which resolves the full module graph).

## Run it locally
```bash
# Needs: kind, kubectl, helm, docker, jq, roxctl, openssl, go (to resolve the
# pinned CRDs; the module cache must already have been populated, e.g. by a
# prior `go build`/`go mod download` in this checkout).
export MAIN_IMAGE_TAG="$(make --quiet --no-print-directory tag)"   # or a released tag
export SECURED_COUNT=2
tests/e2e/kind-compliance/setup-multicluster-compliance-ci.sh up

# Point the Go test at the new Central and run it:
kubectl --context kind-co-ci -n stackrox port-forward svc/central 8443:443 &
export API_ENDPOINT=localhost:8443 ROX_USERNAME=admin
export ROX_ADMIN_PASSWORD="$(cat "${TMPDIR:-/tmp}/kind-co-ci/co-ci/state/admin-password")"
make -C tests compliance-v2-multicluster-tests

tests/e2e/kind-compliance/setup-multicluster-compliance-ci.sh down
```

## CI
`.github/workflows/e2e-multicluster-compliance-kind.yaml` runs the flow on
GitHub Actions: it builds `roxctl`, runs `up` with `SECURED_COUNT=2` (against
the PR's `main`/`central-db` images, already waited for by the shared
dispatcher), then the Go test, and uploads diagnostics. It is a reusable
workflow dispatched by `.github/workflows/e2e-dispatch.yaml` — it only runs on
a PR when the PR carries the `e2e-multicluster-compliance-kind` label (opt-in,
like the other heavier e2e jobs), or always on push/tag builds. It can also be
run directly via `workflow_dispatch`, supplying a `tag`.

## Known limitations / hardening TODO
- Node (`rhcos4-*`) scans and MachineConfig remediation are out of scope; kind
  nodes are not RHCOS. `ocp4-*` platform scans produce real results thanks to
  the OpenShift-API shim, but the shim's ConfigMaps are point-in-time
  snapshots of operator defaults (see the shim's README).
- The CO catalog image is the upstream community catalog pinned by digest in
  the workflow; mirror it into a StackRox-controlled registry before relying
  on this as a hard gate.

## Path to a wider migration
This PoC is a single, narrow demo (one test, one suite) to validate the
framework shape. If it proves out, the natural next steps are: (1) let other
suites that currently have no way to test multi-cluster behavior adopt this
harness, and (2) evaluate migrating parts of the existing Prow e2e suite that
don't strictly need real OpenShift infrastructure onto kind (single- or
multi-cluster, as each suite needs). Both are follow-up work, not part of
this PR.
