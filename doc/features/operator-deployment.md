# Operator Deployment

Kubernetes operator managing complete lifecycle of Central Services and Secured Cluster Services with automatic upgrades and smart defaulting.

**Primary Package**: `operator/`
**Framework**: Kubebuilder + Helm Operator Plugins
**API Version**: `platform.stackrox.io/v1alpha1`

## What It Does

The StackRox Operator manages Central (API server, UI, PostgreSQL, Scanner) and SecuredCluster (Sensor, Collector, Admission Controller, local Scanner) through declarative Kubernetes APIs. It handles sensitive data securely, performs automatic upgrades with existing installation detection, orchestrates CA rotation, and manages PVC lifecycle.

## Architecture

### Custom Resource Definitions

**Central CRD** (`operator/apis/platform/v1alpha1/central_types.go`): Defines central services configuration including admin password settings, exposure methods (LoadBalancer/NodePort/Route), persistence (PVC claims and sizes), resources (CPU/memory requests and limits), database enablement, and scanner configurations (legacy and V4).

**SecuredCluster CRD** (`securedcluster_types.go`): Specifies cluster name and Central endpoint, admission control settings (listen/enforce flags, dynamic config, bypass policy), sensor resources, collector configuration (collection method, resources), and local scanner options.

**Common Types** (`common_types.go`): Shared enums for ExposeAsType, ScannerComponentPolicy, CollectionMethod, AutoScalingPolicy, TLSSecretSource, CustomizePolicy. Common structures for LocalSecretReference, Resources, Scaling, Exposure, and Persistence.

### Reconciliation Engine

The base reconciler in `operator/pkg/reconciler/reconciler.go` defines interfaces for Reconcile() and ReconcileContext providing client access, scheme, and event recorder.

**Central Reconciliation** (`pkg/central/reconciliation/reconciliation.go`):

1. Validate CR spec
2. Apply defaults from `pkg/defaults/central_defaults.go`
3. Translate CRD to Helm values via `pkg/central/values/translation/`
4. Apply extensions (pre-Helm modifications)
5. Render Helm chart from embedded templates
6. Apply Kubernetes resources
7. Run post-render extensions
8. Update status field

**SecuredCluster Reconciliation**: Similar flow with additional sensor bundle generation, certificate management from init bundle or Central API, and cluster registration.

### Defaulting Mechanism

The `operator/pkg/defaults/` implements three-level defaulting: static hardcoded values, platform-specific (OpenShift vs Kubernetes), and detected from existing deployments during upgrades.

Platform detection determines exposure method (Route for OpenShift, LoadBalancer for Kubernetes), sets default admin password generation policy, configures database enablement, and establishes persistence defaults.

Upgrade defaulting detects current version from existing deployments, extracts configuration from running pods, applies only necessary defaults, and preserves user customizations.

### Extensions System

The `operator/pkg/extensions/extension.go` defines Extension interface with Name(), Apply() for resource modification, and ReconciliationExtension adding BeforeReconcile() and AfterReconcile() hooks.

**CA Certificate Extension** (`pkg/central/extensions/ca_rotation.go`): Checks if rotation needed, generates new CA, and updates resources with new certificates.

**Password Generation Extension** (`pkg/central/extensions/password.go`): Generates admin password if not provided, creates bcrypt hash, stores in htpasswd secret.

## Helm Values Translation

Central translation in `operator/pkg/central/values/translation/translation.go` converts CRD fields to Helm chart values: central image/resources/persistence/exposure configuration, database settings, scanner and scanner-v4 configurations.

SecuredCluster translation (`pkg/securedcluster/values/translation/translation.go`) maps cluster identification, admission control settings, collector configuration, and scanner options.

## CA Rotation

Rotation triggers on CA expiration within 30 days or manual request via annotation. The process generates new CA certificate, updates Central TLS secret, restarts Central pods for reconnection, updates SecuredCluster resources, with sensors automatically reconnecting.

Manual rotation: Add annotation `platform.stackrox.io/rotate-ca: "2026-03-13T12:00:00Z"` to Central CR metadata.

## Development

Adding CRD fields (see `operator/EXTENDING_CRDS.md`):

1. Add field to `apis/platform/v1alpha1/*_types.go`
2. Regenerate CRDs: `make manifests`
3. Add default in `pkg/defaults/*_defaults.go`
4. Add translation in `pkg/*/values/translation/translation.go`
5. Add tests and update documentation

Local development: `make install` to install CRDs, `make run` to run operator locally, then apply CR in another terminal.

Testing: `make test` for unit tests, `make test-e2e` for end-to-end, or `go test -v ./operator/pkg/central/reconciliation/...` for specific packages.

## Recent Changes

Recent work addressed ROX-32630 (OCP console plugin deployment), ROX-32412 (compliance metrics scraping), ROX-30937 (process baseline auto-locking config), ROX-30608 (new admission controller config endpoint), ROX-30278 (static admission controller fields and enforce setting), ROX-30034 (new admission controller failure policy options), plus feature flag integration improvements and enhanced upgrade scenario defaulting.

## Implementation

**CRDs**: `operator/apis/platform/v1alpha1/central_types.go`, `operator/apis/platform/v1alpha1/securedcluster_types.go`
**Reconciliation**: `operator/pkg/central/reconciliation/reconciliation.go`, `operator/pkg/securedcluster/reconciliation/reconciliation.go`
**Defaulting**: `operator/pkg/defaults/central_defaults.go`, `operator/pkg/defaults/securedcluster_defaults.go`
**Translation**: `operator/pkg/central/values/translation/translation.go`, `operator/pkg/securedcluster/values/translation/translation.go`
**Extensions**: `operator/pkg/central/extensions/ca_rotation.go`, `operator/pkg/central/extensions/password.go`
