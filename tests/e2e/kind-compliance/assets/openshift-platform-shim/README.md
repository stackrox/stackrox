# OpenShift-API shim for real `ocp4` PLATFORM checks

`PLATFORM=OpenShift` gets the real `ocp4`/`rhcos4` ProfileBundles to sync, but every
`ocp4-*` PLATFORM scan comes back NOT-APPLICABLE on kind: the OVAL checks read
`config.openshift.io`/`operator.openshift.io`/`security.openshift.io` resources
(API server config, SCCs, cluster network, etc.) that a plain kind cluster simply
does not have. This directory adds those resources so the Compliance Operator's
api-resource-collector can actually fetch them, which flips `ocp4-cis` from
NOT-APPLICABLE to real PASS/FAIL/MANUAL results.

## Layout
- CRDs (NOT committed here — sourced at setup time, see below) — the REAL CRDs
  for the OpenShift API groups the `ocp4-cis` OVAL checks read:
  `config.openshift.io/v1` (apiservers, infrastructures, networks, oauths, images,
  proxies*, schedulers*, clusteroperators, consoles*, ingresses*, authentications*,
  featuregates*, clusterversions), `operator.openshift.io/v1` (kubeapiservers,
  openshiftapiservers, networks, ingresscontrollers), `security.openshift.io/v1`
  (securitycontextconstraints). (*: CRD installed for API completeness/discovery;
  no singleton object is created for it in `objects.yaml`.) These REPLACE/SUPERSEDE
  the minimal `x-kubernetes-preserve-unknown-fields: true` placeholder
  `config.openshift.io` CRDs that `assets/openshift-shim/config-crds.yaml` installs
  for `cel`/`content` fidelity — same groups/kinds, real schemas.
- `objects.yaml` — namespaces plus ~20 singleton CRs/ConfigMaps/misc objects the
  checks actually read: `infrastructures/cluster`, `apiservers/cluster`,
  `networks/cluster` (both `config` and `operator` groups), `oauths/cluster`,
  `images/cluster`, a `ClusterOperator`, `kubeapiservers/cluster` and
  `openshiftapiservers/cluster` (operator CRs), `ingresscontrollers/default` (ns
  `openshift-ingress-operator`), an empty `ClusterRole/cluster-debugger`, a
  `DaemonSet/sdn` (scoped with a nodeSelector that matches no kind node, so it
  costs zero CPU/RAM/image pulls while still being a real queryable API object),
  and the four ConfigMaps below. Applied with `kubectl apply` (idempotent).
- `sccs.yaml` — the real `restricted-v2` and `privileged` SecurityContextConstraints,
  verbatim from `cluster-kube-apiserver-operator`'s bootkube bindata. Applied with
  **`kubectl create`**, not `apply`: the real SCC CRD requires `priority`,
  `allowedCapabilities`, `defaultAddCapabilities`, and `volumes` to be *present*
  even when `null`, but `kubectl apply`'s client-side JSON-merge machinery strips
  explicit `key: null` fields before sending the request, which then fails CRD
  validation ("missing required field"). A plain `create` (or `apply
  --server-side`) sends the object as submitted, nulls included.

### The four ConfigMaps (the actual PASS/FAIL drivers)
`ocp4-cis`'s OVAL checks are mostly `apiServerArguments`/`extendedArguments`
lookups against these control-plane ConfigMaps — a field's value maps directly to
a rule's PASS/FAIL, which is what makes this shim deterministic:
- `openshift-kube-apiserver/config` — the single highest-leverage object (drives
  ~27 of the 73 rules). `apiServerArguments` is the **real, unmodified** upstream
  default from `cluster-kube-apiserver-operator`'s
  `bindata/assets/config/defaultconfig.yaml` (fetched at asset-authoring time, not
  hand-written), wrapped in a realistic `KubeAPIServerConfig` envelope
  (`servingInfo.cipherSuites`/`minTLSVersion` matching the default `Intermediate`
  TLS profile).
- `openshift-apiserver/config` — captured from a **real cluster**, borrowed from
  the ComplianceAsCode/compliance-operator unit-test fixture
  `tests/data/configmap_json.json` (`cmd/manager/scap_test.go` et al. use it).
- `openshift-kube-controller-manager/config` — the real upstream default from
  `cluster-kube-controller-manager-operator`'s
  `bindata/assets/config/defaultconfig.yaml`.
- `openshift-etcd/etcd-pod` — a trimmed-but-structurally-real rendering of
  `cluster-etcd-operator`'s `bindata/etcd/pod.gotpl.yaml` static pod template,
  with template placeholders substituted for concrete values (TLS cert/key
  paths, `--client-cert-auth=true`, `--peer-client-cert-auth=true`, no
  `--auto-tls`, etc.).

Because these are **real upstream defaults** (not a hand-tuned "pass everything"
or "fail everything" config), scanning them naturally produces a realistic mix of
PASS and FAIL, the same way a freshly installed, not-yet-hardened OpenShift
cluster would.

## Maintenance caveat: content/CRD version coupling
The CRDs are sourced at `up` time from the `github.com/openshift/api` version
this repo's `go.mod` pins (see the harness README's "CRD and manifest
sourcing"), so they can't drift from that pin the way a committed snapshot
could — bumping the go.mod pin is all that's needed to pick up new CRD schema.
`sccs.yaml`, and the `apiServerArguments`/`extendedArguments` fields in
`objects.yaml`, are kept in sync automatically — see "Keeping this up to
date" below — against SHA-pinned upstream commits, independent of that pin.
The other two ConfigMaps (`openshift-apiserver/config`,
`openshift-etcd/etcd-pod`) are still hand-captured, point-in-time snapshots
with no single upstream file to pin (see above): re-capture those by hand if
they ever need to change. The content (ProfileBundle) image is pinned
transitively via `CO_CATALOG_IMAGE` (see the harness README's
content-determinism caveat): if a future `ocp4` content update starts reading
a field, CRD, or resource this shim does not provide, those specific rules
will fall back to NOT-APPLICABLE/MANUAL rather than erroring.

## Keeping this up to date
`sccs.yaml` (entirely) and two fields in `objects.yaml` (the
`openshift-kube-apiserver/config` ConfigMap's `apiServerArguments`, and the
`openshift-kube-controller-manager/config` ConfigMap's `extendedArguments`)
are copies of real upstream OpenShift operator files, not hand-written. They
are regenerated by [`regenerate.sh`](./regenerate.sh) from SHA-pinned upstream
commits (never a floating branch) — see the script for the exact source paths
and the two `# BEGIN/END UPSTREAM` marker comments in `objects.yaml` that
bound where it splices in the two fields, leaving the rest of that
first-party file untouched.

`.github/workflows/update_openshift_platform_shim_periodic.yaml` runs
`regenerate.sh` weekly (and on demand via `workflow_dispatch`) against each
upstream repo's latest commit, and opens a PR with the result only if
something actually changed (handled by `peter-evans/create-pull-request`'s
default no-op-on-no-diff behavior — the same pattern already used by
`update_collector_periodic.yaml`/`update_fact_periodic.yaml`). Review that PR
like any other dependency bump: a changed default flag or a new SCC field can
flip which `ocp4-cis` checks PASS/FAIL.
