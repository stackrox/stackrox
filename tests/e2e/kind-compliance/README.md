# Multi-cluster compliance on kind (PoC)

Compliance e2e currently runs on OpenShift only (Prow), because the Compliance
Operator (CO) is an OpenShift operator. This directory is a proof of concept that
the same StackRox compliance integration can be exercised **multi-cluster, on
kind, in GitHub Actions** — cheaply enough for a PR gate.

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
- `assets/co-crds/` — the Compliance Operator CRDs (from the upstream bundle).
- `assets/operator-generic.yaml` — the CO operator Deployment + RBAC, run with
  `PLATFORM=Generic` (reconstructed from the bundle CSV; no OLM).
- `assets/openshift-shim/` — minimal `config.openshift.io` CRDs + singleton
  objects. These keep the operator's TLS-profile watcher happy and let Sensor
  detect the cluster as OpenShift.
- `assets/cel-profile.yaml` — a `ConfigMap` + two CEL `CustomRule`s (one passes,
  one fails) + a from-scratch `TailoredProfile` (`kind-cel-tp`). Profile only: the
  scan is driven from Central by the Go test, not by a local ScanSettingBinding.

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

In CI, `content` is available via the workflow's manual `workflow_dispatch` input;
PR runs use `cel`.

## The tests
`tests/compliance_operator_v2_multicluster_test.go` (build tag `compliance`) talk
only to Central's v2 API:
- `TestComplianceV2MultiClusterIntegrations` — every secured cluster has its own
  HEALTHY CO integration, and they are distinct.
- `TestComplianceV2MultiClusterScanConfigResults` — one scan configuration fans
  out to all clusters exposing the CEL profile, and real PASS + FAIL results come
  back attributed to each cluster.
- `TestComplianceV2MultiClusterRealProfiles` — (content mode) the real
  `ocp4`/`rhcos4` profile+rule catalog synced to Central for every cluster. Auto-
  skips when only the CEL profile is present.

All skip (not fail) when fewer than two suitable clusters are present (or, for the
real-profiles test, when real ProfileBundles are absent), so they are safe in
single-cluster and `cel`-mode runs.

## Run it locally
```bash
# Needs: kind, kubectl, helm, docker, jq, roxctl, openssl.
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
`.github/workflows/multicluster-compliance-poc.yaml` runs the flow on GitHub
Actions: it waits for the PR's `main` and `central-db` images, builds `roxctl`,
runs `up`, then the Go tests, and uploads diagnostics.

## Known limitations / hardening TODO
- Node (`rhcos4-*`) scans and MachineConfig remediation are out of scope; kind
  nodes are not RHCOS. `ocp4-*` platform scans run but yield NOT-APPLICABLE on
  kind — CEL `CustomRule`s are how we get real results here.
- The CO operator image is a dev image pinned by digest in the workflow; mirror it
  into a StackRox-controlled registry before relying on this as a hard gate.
- Compute (not disk) is the limit when packing many clusters: Sensor/CO run per
  cluster. `collector.disable=true` is used to cut CPU/RAM for the compliance path.
