# Operator Architecture

This document describes the internal architecture of the StackRox operator.
For how to extend the CRDs and set defaults, see [EXTENDING_CRDS.md](EXTENDING_CRDS.md) and [DEFAULTING.md](DEFAULTING.md).

## High-level design

The operator is a hybrid Go+Helm-based operator built on a
[fork](https://github.com/stackrox/helm-operator) of
[operator-framework/helm-operator-plugins](https://github.com/operator-framework/helm-operator-plugins).
The fork includes features and bug fixes specific to StackRox that have not
been upstreamed; see [ROX-7911](https://issues.redhat.com/browse/ROX-7911).

It watches essentially two CRDs — `Central` and `SecuredCluster` — and for each, renders
the corresponding Helm chart (`stackrox-central-services` /
`stackrox-secured-cluster-services`) with values derived from the CR spec.

Apart from that, there are some more features and caveats such as:
- additional watches (see [section on reconcilers](#cr-specific-reconcilers-internalcentralreconciler-internalsecuredclusterreconciler)),
- [post-renderers](#post-renderers),
- an advanced [defaulting](#defaulting-internalcentraldefaults-internalsecuredclusterdefaults) mechanism,
- separate [status controller](#status-controller-internalcommonstatuscontrollergo).

## Entry point ([`cmd/main.go`](cmd/main.go))

`main.go` creates a controller-runtime `Manager` and registers:

1. **Central reconciler** — via [`internal/central/reconciler.RegisterNewReconciler`](internal/central/reconciler/reconciler.go).
2. **SecuredCluster reconciler** — via [`internal/securedcluster/reconciler.RegisterNewReconciler`](internal/securedcluster/reconciler/reconciler.go).
3. **Status controllers** — one per CR type, via [`internal/common/status.New`](internal/common/status/controller.go). These are independent of the Helm reconcilers (see [Status controller](#status-controller-internalcommonstatuscontrollergo) below).

Each reconciler and status controller can be independently disabled via
environment variables (`CENTRAL_RECONCILER_ENABLED`, etc.).

The manager also configures:
- label-filtered caching for Secrets and ConfigMaps (to limit memory consumption),
- leader election,
- metrics serving,
- TLS profile enforcement and watcher (see [TLS profile management](#tls-profile-management-internaltlsprofile)).

### Reconciliation pipeline

The Helm reconcilers process each CR through this pipeline:

1. CR change detected
2. Pre-extensions (defaulting, secret reconciliation, validation, ...)
3. Values translation (CR spec -> Helm values)
4. Values enrichment (proxy env, TLS profile, image pull secrets, routes)
5. Helm render templates from `image/templates/helm` into manifests using above values
6. Post-renderers apply changes to rendered manifests (overlays, labels, CA-hash annotations)
7. Apply to cluster

## Key components

### CRD types ([`api/v1alpha1/`](api/v1alpha1/))

[`central_types.go`](api/v1alpha1/central_types.go) and [`securedcluster_types.go`](api/v1alpha1/securedcluster_types.go) define the CR spec and status
structs with kubebuilder validation markers and operator-sdk CSV markers.

[`common_types.go`](api/v1alpha1/common_types.go) contains shared types (image overrides, TLS config,
customization, scanner specs, etc.).

[`defaults_merging.go`](api/v1alpha1/defaults_merging.go) implements the defaults-related logic, see section on defaulting, below.

### Generic reconciler factory ([`internal/reconciler/reconciler_factory.go`](internal/reconciler/reconciler_factory.go))

`SetupReconcilerWithManager` is the common function that both CR-specific
reconcilers call. It:

1. Loads the embedded Helm chart via `image.GetDefaultImage().LoadChart(...)`.
2. Creates a helm-operator-plugins `ActionClientGetter` with
   [post-renderers](#post-renderers) chained.
3. Assembles reconciler options (value translator, release history size,
   failure timeout, etc.) and registers the reconciler with the manager.

### CR-specific reconcilers ([`internal/central/reconciler/`](internal/central/reconciler/reconciler.go), [`internal/securedcluster/reconciler/`](internal/securedcluster/reconciler/reconciler.go))

Each `RegisterNewReconciler` function wires up the CR-specific pieces:

- **Pre-extensions** — run before Helm rendering; registered via
  `pkgReconciler.WithPreExtension(...)`. The ordering matters: the
  `FeatureDefaultingExtension` always runs first, followed by all other
  extensions. Extensions handle concerns such as feature defaulting,
  secret and PVC lifecycle management, input validation, and status
  enrichment. Central-specific extensions live in
  [`internal/central/extensions/`](internal/central/extensions/),
  SecuredCluster-specific ones in
  [`internal/securedcluster/extensions/`](internal/securedcluster/extensions/),
  and shared ones in
  [`internal/common/extensions/`](internal/common/extensions/).

- **Extra watches** — each reconciler watches the sibling CR type
  (see [Cross-CR interaction](#cross-cr-interaction)) and SecuredCluster
  also watches specific Secrets and ConfigMaps (sensor TLS, CA bundle).

- **Values translator** — the CR-specific translator wrapped in an enrichment
  chain (see [Values translation pipeline](#values-translation-pipeline) below).

- **Event predicates** — skip reconciliation for status-only updates
  (see [Status controller](#status-controller-internalcommonstatuscontrollergo)).

### Values translation pipeline

Translation converts a CR spec into Helm chart values. It is a two-stage
pipeline:

1. **Translator** ([`internal/central/values/translation/`](internal/central/values/translation/translation.go), [`internal/securedcluster/values/translation/`](internal/securedcluster/values/translation/translation.go)) —
   converts the typed CR into a `chartutil.Values` map. Each translator
   starts from a [`base-values.yaml`](internal/central/values/translation/base-values.yaml) file embedded at compile time, then
   overlays field-by-field translations using [`ValuesBuilder`](internal/values/translation/values_builder.go) (a typed
   helper that accumulates values and errors).

2. **Enrichers** ([`internal/values/translation/enricher.go`](internal/values/translation/enricher.go)) — a chain of
   `Enricher` implementations that post-process the values. The enrichment
   chain is assembled in each `RegisterNewReconciler`:
   - **Proxy env-var injector** ([`internal/proxy/`](internal/proxy/translation.go)) — injects `HTTP_PROXY` / `HTTPS_PROXY` values.
   - **TLS profile enricher** ([`internal/tlsprofile/enricher.go`](internal/tlsprofile/enricher.go)) — injects `ROX_TLS_MIN_VERSION` /
     `ROX_TLS_CIPHER_SUITES` env vars from the cluster-wide TLS profile.
   - **Image pull secret reference injector** ([`internal/legacy/secrets.go`](internal/legacy/secrets.go)) — adds references to
     pre-existing well-known pull secrets for backward compatibility.
   - **Route injector** (Central only) ([`internal/route/translation.go`](internal/route/translation.go)) — injects the Central CA cert into
     OpenShift re-encrypt route values.

Shared translation utilities (resource conversion helpers, scheduling
defaults, scanner values) live in [`internal/values/translation/translation.go`](internal/values/translation/translation.go).

### Defaulting ([`internal/central/defaults/`](internal/central/defaults/), [`internal/securedcluster/defaults/`](internal/securedcluster/defaults/))

Defaulting is implemented as a sequence of *defaulting flows*. Each flow
populates `central.Defaults` (or `securedCluster.Defaults`) which is later
merged onto `.Spec` — preserving user choices and without persisting defaults
to the cluster. This allows changing defaults in future operator versions
without breaking existing CRs.

Flow types:
- **Static defaults** ([`internal/central/defaults/static.go`](internal/central/defaults/static.go), [`internal/securedcluster/defaults/static.go`](internal/securedcluster/defaults/static.go)) — a fixed `CentralSpec` / `SecuredClusterSpec` struct
  applied first, providing baseline values for all fields.
- **Dynamic defaults** — more complex flows (e.g. Scanner V4 enabling,
  Central DB persistence) that can inspect status, annotations, and spec to
  make decisions. They may persist their choices as annotations on the CR.

See [DEFAULTING.md](DEFAULTING.md) for full details.

### Status controller ([`internal/common/status/controller.go`](internal/common/status/controller.go))

A lightweight controller (separate from the Helm reconciler) that provides
real-time status updates without invoking Helm. It watches Deployments and
DaemonSets owned by the CR and maintains two conditions:

- **Available** — true when all owned workloads report available replicas.
- **Progressing** — true when `metadata.generation > status.observedGeneration`
  (spec change pending reconciliation) or when the `Irreconcilable` condition
  is set.

A custom predicate ([`NewSkipStatusControllerUpdates`](internal/common/status/predicate.go)) prevents the Helm
reconciler from being triggered by status-only updates made by this
controller.

### Post-renderers

After Helm renders manifests, three post-renderers process the output:

1. **Overlays** ([`internal/overlays/postrenderer.go`](internal/overlays/postrenderer.go)) — applies user-defined patches from `spec.overlays` using
   [k8s-overlay-patch](https://github.com/stackrox/k8s-overlay-patch).
2. **Labels** ([`internal/common/labels/labels.go`](internal/common/labels/labels.go)) — adds standard operator labels to all rendered objects.
3. **Config-hash annotations** ([`internal/common/confighash/pod_template_annotation.go`](internal/common/confighash/pod_template_annotation.go)) — computes a hash of the CA
   certificate and annotates Deployment and DaemonSet pod templates, causing
   automatic rollout when the CA rotates.

### TLS profile management ([`internal/tlsprofile/`](internal/tlsprofile/))

On OpenShift clusters, the operator reads the cluster-wide TLS security
profile from `apiserver.config.openshift.io/cluster`. This profile influences:

- The operator's own metrics server TLS configuration.
- Operand TLS settings (injected via the TLS profile enricher).
- A [watcher](internal/tlsprofile/watch.go) that triggers a graceful shutdown of the operator process
  (by cancelling the manager's root context) when the cluster TLS profile
  changes, so that it restarts with the updated profile.

### Proxy handling ([`internal/proxy/`](internal/proxy/))

The operator captures proxy environment variables (`HTTP_PROXY`, `HTTPS_PROXY`,
etc.) at startup and:

- Creates a Secret per CR containing proxy env vars (via a [pre-extension](internal/proxy/extension.go)).
- Injects the proxy settings into Helm values (via an [enricher](internal/proxy/translation.go)).

### Cross-CR interaction

Central and SecuredCluster reconcilers watch each other's CR type. This enables
SecuredCluster to decide whether to deploy a local scanner based on whether a
Central exists in the same namespace.

## Directory layout

```text
operator/
├── api/v1alpha1/          CRD Go types, defaults merging, deep-copy
├── cmd/                   Entry point (main.go)
├── internal/
│   ├── central/           Central-specific logic
│   │   ├── carotation/      CA certificate rotation
│   │   ├── defaults/        Static and dynamic defaulting flows
│   │   ├── extensions/      Pre-extensions (secrets, PVCs, TLS, defaulting, collision check)
│   │   ├── reconciler/      Wires up the Central Helm reconciler
│   │   └── values/          CR-to-Helm-values translation
│   ├── securedcluster/    SecuredCluster-specific logic (mirrors central/ structure)
│   │   ├── defaults/        Static and dynamic defaulting flows
│   │   ├── extensions/      Pre-extensions (cluster name, passwords, defaulting)
│   │   ├── reconciler/      Wires up the SecuredCluster Helm reconciler
│   │   ├── scanner/         Local scanner auto-sense logic
│   │   └── values/          CR-to-Helm-values translation
│   ├── common/            Shared logic
│   │   ├── confighash/      CA certificate hash and pod-template annotation for rollout on CA rotation
│   │   ├── extensions/      Shared pre-extensions (forbidden namespaces, label selectors, etc.)
│   │   ├── labels/          Default labels and label post-renderer
│   │   ├── rendercache/     Caches rendered manifests between reconciliation cycles
│   │   └── status/          Status controller (Available/Progressing conditions)
│   ├── reconciler/        Generic reconciler factory (SetupReconcilerWithManager)
│   ├── values/            Shared values translation utilities and enricher chain
│   ├── overlays/          Overlay post-renderer (applies spec.overlays patches)
│   ├── proxy/             Proxy env-var forwarding (extension + enricher)
│   ├── legacy/            Backward-compat image pull secret reference injector
│   ├── route/             OpenShift Route TLS CA injection enricher
│   ├── tlsprofile/        Cluster TLS profile fetching, watching, and enricher
│   ├── images/            Image override resolution
│   ├── config/            Static configuration (mapkubeapis)
│   ├── types/             Shared type definitions
│   └── utils/             Helpers (predicates, REST config, storage class checks)
├── config/                Kustomize bases for CRDs, RBAC, manager deployment
├── bundle/                OLM bundle manifests
├── bundle_helpers/        Go tools for patching CSV and sorting descriptors
├── tests/                 Kuttl-based E2E tests
├── install/               Community operator installation samples and docs
├── hack/                  Shell scripts (chart generation, etc.)
└── tools/                 Pinned Go tool dependencies
```
