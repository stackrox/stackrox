# OpenShift-API shim for real `ocp4` PLATFORM checks (`FIDELITY=platform`)

`content` fidelity gets the real `ocp4`/`rhcos4` ProfileBundles to sync, but every
`ocp4-*` PLATFORM scan comes back NOT-APPLICABLE on kind: the OVAL checks read
`config.openshift.io`/`operator.openshift.io`/`security.openshift.io` resources
(API server config, SCCs, cluster network, etc.) that a plain kind cluster simply
does not have. This directory adds those resources so the Compliance Operator's
api-resource-collector can actually fetch them, which flips `ocp4-cis` from
NOT-APPLICABLE to real PASS/FAIL/MANUAL results.

## Layout
- `crds/` — the REAL CRDs for the OpenShift API groups the `ocp4-cis` OVAL checks
  read, fetched verbatim from `github.com/openshift/api` (`zz_generated.crd-manifests/`):
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
The CRDs here are a point-in-time snapshot from `openshift/api@master`, and the
ConfigMap contents are point-in-time snapshots of operator defaults. If a future
`ocp4` content update (pinned via `CO_CONTENT_IMAGE`, see the harness README)
starts reading a field, CRD, or resource this shim does not provide, those
specific rules will fall back to NOT-APPLICABLE/MANUAL rather than erroring —
but to keep `ocp4-cis` fully exercised, re-sync this directory against
`github.com/openshift/api` (same paths, `zz_generated.crd-manifests/`) and the
three operator repos' `defaultconfig.yaml` when bumping `CO_CONTENT_IMAGE`.

## Source commands (how this was generated)
```bash
# CRDs: raw.githubusercontent.com/openshift/api/master/<group>/v1/zz_generated.crd-manifests/<file>
# e.g. config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_apiservers-Default.crd.yaml

# openshift-kube-apiserver/config apiServerArguments:
curl -fsSL https://raw.githubusercontent.com/openshift/cluster-kube-apiserver-operator/master/bindata/assets/config/defaultconfig.yaml

# openshift-kube-controller-manager/config extendedArguments:
curl -fsSL https://raw.githubusercontent.com/openshift/cluster-kube-controller-manager-operator/master/bindata/assets/config/defaultconfig.yaml

# SCCs:
curl -fsSL https://raw.githubusercontent.com/openshift/cluster-kube-apiserver-operator/master/bindata/bootkube/scc-manifests/0000_20_kube-apiserver-operator_00_scc-restricted-v2.yaml
curl -fsSL https://raw.githubusercontent.com/openshift/cluster-kube-apiserver-operator/master/bindata/bootkube/scc-manifests/0000_20_kube-apiserver-operator_00_scc-privileged.yaml
```
