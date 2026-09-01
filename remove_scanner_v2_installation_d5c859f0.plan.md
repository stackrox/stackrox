---
name: Remove Scanner v2 Installation
overview: Remove deprecated Scanner v2 (Clairify/StackRox Scanner) from all installation methods -- Helm charts, operator, deploy scripts, and roxctl -- while preserving backward compatibility for existing CRs and keeping the Go runtime code gated behind the existing ROX_LEGACY_SCANNER feature flag.
todos:
  - id: helm-central-defaults
    content: Change scanner.disable from false to true in central Helm chart defaults (defaults.yaml.htpl) and clean up scanner v2 image/resource defaults
    status: pending
  - id: helm-sc-defaults
    content: "Disable scanner v2 in secured cluster Helm chart: update 70-scanner.yaml, _scanner-defaulting.tpl, and _init.tpl.htpl"
    status: pending
  - id: helm-values-docs
    content: Update Helm values examples, READMEs, and NOTES.txt to remove scanner v2 references and outdated 'scanner v4 requires scanner v2' language
    status: pending
  - id: operator-types
    content: Mark CRD Scanner fields as deprecated/ignored in central_types.go and securedcluster_types.go
    status: pending
  - id: operator-translation
    content: Update operator values translation to always emit scanner.disable=true instead of translating Scanner CR fields
    status: pending
  - id: operator-defaults
    content: Remove scanner v2 defaulting logic in operator defaults and auto-sense code
    status: pending
  - id: operator-extensions
    content: Update operator extension reconcilers (scanner DB password, TLS) to be no-ops for scanner v2
    status: pending
  - id: deploy-scripts
    content: "Update deploy scripts: set SCANNER_SUPPORT=false, remove scanner v2 image exports, update local-dev-values, update k8sbased.sh scanner v2 deploy logic"
    status: pending
  - id: roxctl
    content: Remove scanner v2 image flags from roxctl scanner generate and roxctl central generate commands
    status: pending
  - id: central-init-tpl
    content: "Update Central _init.tpl.htpl: force _legacyScannerEnabled=false, update ROX_LEGACY_SCANNER env injection logic"
    status: pending
  - id: helm-unit-tests
    content: "Update Helm unit tests: scanner.test.yaml, scanner-slim.test.yaml, and scanner-v4.test.yaml coexistence tests"
    status: pending
  - id: operator-tests
    content: "Update operator tests: translation tests, defaulting tests, kuttl E2E test fixtures"
    status: pending
  - id: e2e-bats
    content: "Update E2E bats tests: flip verify_scannerV2_deployed to verify_no_scannerV2_deployed, remove v2 resource settings from Helm values, remove v2 HPA patches"
    status: pending
isProject: false
---

# Remove Scanner v2 from All Installation Methods

## Background

Scanner v2 (also called "StackRox Scanner" or "Clairify") is the legacy vulnerability scanner. Scanner v4 (ClairCore-based) is its replacement. The codebase already has a `ROX_LEGACY_SCANNER` feature flag (default: `enabled`) gating the runtime Go code. This plan removes Scanner v2 from **installation/deployment paths only** while keeping the runtime code intact behind the feature flag.

## Key Files and Components

### 1. Helm Charts -- Central Services

**Scanner v2 template files to disable** (all in `image/templates/helm/shared/templates/`):
- `02-scanner-00-serviceaccount.yaml` -- ServiceAccount
- `02-scanner-01-security.yaml` -- SCC/SecurityContext
- `02-scanner-01-psps.yaml` -- PodSecurityPolicy
- `02-scanner-02-db-password-secret.yaml` -- DB password Secret
- `02-scanner-03-tls-secret.yaml` -- TLS Secret
- `02-scanner-04-scanner-config.yaml` -- ConfigMap
- `02-scanner-05-network-policy.yaml` -- NetworkPolicy
- `02-scanner-06-deployment.yaml.htpl` -- Deployment (scanner + scanner-db)
- `02-scanner-07-service.yaml` -- Service
- `02-scanner-08-hpa.yaml` -- HPA

All of these are already gated on `{{- if not ._rox.scanner.disable -}}`. The change is to **force `scanner.disable=true`** so these templates never render.

**Approach**: In [defaults.yaml.htpl](image/templates/helm/stackrox-central/internal/defaults.yaml.htpl), change `scanner.disable: false` to `scanner.disable: true`. This single change disables the entire Scanner v2 stack in central-services Helm chart. Remove the scanner v2 image defaults and resource defaults from this file. Any user-supplied `scanner.*` values are harmless: since all `02-scanner-*` templates are gated on `scanner.disable`, those values are write-only and never consumed.

### 2. Helm Charts -- Secured Cluster

**Scanner v2 defaults to disable**:
- [70-scanner.yaml](image/templates/helm/stackrox-secured-cluster/internal/defaults/70-scanner.yaml) -- default values for local scanner
- [_scanner-defaulting.tpl](image/templates/helm/stackrox-secured-cluster/templates/_scanner-defaulting.tpl) -- auto-sense defaulting logic
- [_init.tpl.htpl](image/templates/helm/stackrox-secured-cluster/templates/_init.tpl.htpl) -- initialization that calls `srox.scannerInit`

**Approach**: Set scanner v2 to always-disabled in the secured cluster defaults. Update the `_scanner-defaulting.tpl` so that scanner v2 is always set to `disable: true` regardless of user input or auto-sense logic. In `_init.tpl.htpl`, skip the `srox.scannerInit` call for scanner v2 (but keep scanner v4 init). Update the `$anyScannerEnabled` logic to only consider Scanner v4.

### 3. Helm Charts -- Shared Init and Config

These files become dead code since `scanner.disable=true` prevents all callers from reaching them:
- [_scanner_init.tpl.htpl](image/templates/helm/shared/templates/_scanner_init.tpl.htpl) -- scanner v2 init template (never called)
- [scanner-config-shape.yaml](image/templates/helm/shared/internal/scanner-config-shape.yaml) -- config shape (values accepted but never consumed)
- [config-templates/scanner/config.yaml.tpl](image/templates/helm/shared/config-templates/scanner/config.yaml.tpl) -- scanner config template (never rendered)

No changes needed in these files -- they can be cleaned up in a follow-up.

### 4. Helm Values Examples and Documentation

- [values-public.yaml.example.htpl](image/templates/helm/stackrox-central/values-public.yaml.example.htpl) -- remove/update scanner v2 sections
- [values.yaml.htpl](image/templates/helm/stackrox-central/values.yaml.htpl) -- update `scanner.disable=true` example
- [values-scanner.yaml.example](image/templates/helm/stackrox-secured-cluster/values-scanner.yaml.example) -- remove scanner v2 example
- [README.md.htpl](image/templates/helm/stackrox-central/README.md.htpl) and [README.md.htpl](image/templates/helm/stackrox-secured-cluster/README.md.htpl) -- update docs
- Remove outdated references in values examples that say "Scanner V4 cannot be used without scanner v2"

### 5. Operator CRD Types

Follow the existing "Obsolete field" pattern used for `Persistence *ObsoletePersistence`, `IsEnabled`, `ForceCollection`, etc.:
- Doc comment: `// Obsolete field. This field will be removed in a future release.`
- Add xDescriptors `{"urn:alm:descriptor:com.tectonic.ui:hidden"}` to hide from OLM UI

**File**: [central_types.go](operator/api/v1alpha1/central_types.go)

Keep `Scanner *ScannerComponentSpec` field in `CentralSpec` but mark as obsolete:
```go
// Obsolete field. This field will be removed in a future release.
// Scanner V2 has been removed. This field is ignored.
//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
Scanner *ScannerComponentSpec `json:"scanner,omitempty"`
```

**File**: [securedcluster_types.go](operator/api/v1alpha1/securedcluster_types.go)

Same pattern for `Scanner *LocalScannerComponentSpec`.

### 6. Operator Values Translation

**File**: [operator/internal/central/values/translation/translation.go](operator/internal/central/values/translation/translation.go)

In function `translate()`, stop emitting `scanner` Helm values from the CR. Change:
```go
v.AddChild("scanner", getCentralScannerComponentValues(c.Spec.Scanner, deploymentDefaults))
```
to emit a values builder that always sets `scanner.disable: true`.

**File**: [operator/internal/securedcluster/values/translation/translation.go](operator/internal/securedcluster/values/translation/translation.go)

Same pattern for the secured cluster translation.

### 7. Operator Defaults

**Scanner V2 defaulting -- remove**:
- [operator/internal/central/defaults/static.go](operator/internal/central/defaults/static.go) -- remove scanner v2 analyzer scaling defaults
- [operator/internal/securedcluster/defaults/static.go](operator/internal/securedcluster/defaults/static.go) -- remove scanner v2 analyzer scaling defaults
- [operator/internal/securedcluster/scanner/auto_sense.go](operator/internal/securedcluster/scanner/auto_sense.go) -- scanner v2 auto-sense should always result in disabled
- [operator/internal/securedcluster/scanner/defaults.go](operator/internal/securedcluster/scanner/defaults.go) -- stop defaulting scanner v2 to AutoSense

**Scanner V4 defaulting -- simplify by ignoring the feature-defaults annotation**:

The `feature-defaults.platform.stackrox.io/scannerV4` annotation was used to distinguish new installs (V4 enabled) from upgrades (V4 disabled, to avoid surprising users who still ran V2). Now that V2 is removed, V4 must always be on, and the upgrade-vs-install distinction is obsolete.

- [operator/internal/central/defaults/scanner_v4.go](operator/internal/central/defaults/scanner_v4.go):
  - `CentralScannerV4ComponentPolicy`: **ignore** the annotation when deciding the default. Always return `Enabled` (unless user explicitly set `Disabled` in the CR spec).
  - `centralScannerV4Defaulting`: **write** the annotation as `Enabled` if absent or different, to support downgrade scenarios where an older operator reads it.

- [operator/internal/securedcluster/defaults/scanner_v4.go](operator/internal/securedcluster/defaults/scanner_v4.go):
  - `SecuredClusterScannerV4ComponentPolicy`: same pattern -- always return `AutoSense` (unless user explicitly set `Disabled`), ignoring the annotation.
  - `securedClusterScannerV4Defaulting`: write `AutoSense` annotation for downgrade support.

- [operator/api/v1alpha1/central_defaults.go](operator/api/v1alpha1/central_defaults.go) -- no change needed (the `ScannerV4ComponentDefault` -> `nil` clearing logic in `MergeCentralDefaultsIntoSpec` is still correct)
- [operator/api/v1alpha1/securedcluster_defaults.go](operator/api/v1alpha1/securedcluster_defaults.go) -- same, no change needed

### 8. Operator Extension Reconcilers

**Files**:
- [operator/internal/central/extensions/reconcile_scanner_db_password_test.go](operator/internal/central/extensions/reconcile_scanner_db_password_test.go) -- scanner DB password reconciliation
- Related TLS reconciler tests

Scanner v2 password/TLS reconcilers should be updated: if the operator no longer deploys scanner v2, these extensions should be no-ops for the scanner v2 case.

### 9. Deploy Scripts

**File**: [deploy/common/env.sh](deploy/common/env.sh)
- `SCANNER_SUPPORT` defaults to `true` -- change to `false`
- `SENSOR_SCANNER_SUPPORT` defaults to `false` -- keep as is (already disabled)

**File**: [deploy/common/deploy.sh](deploy/common/deploy.sh)
- Remove `SCANNER_IMAGE_REPO`, `SCANNER_IMAGE_TAG`, `SCANNER_IMAGE`, `SCANNER_DB_IMAGE_REPO`, `SCANNER_DB_IMAGE` exports (scanner v2 images)
- Keep Scanner V4 image exports

**File**: [deploy/common/local-dev-values.yaml](deploy/common/local-dev-values.yaml)
- Set `scanner.disable: true`

**File**: [deploy/common/scanner-local-patch.yaml](deploy/common/scanner-local-patch.yaml)
- Remove or empty this file (it patches scanner v2 for local dev)

### 10. Central Helm `_init.tpl.htpl` -- Scanner v2 Feature Flag Injection

**File**: [image/templates/helm/stackrox-central/templates/_init.tpl.htpl](image/templates/helm/stackrox-central/templates/_init.tpl.htpl)

This file sets `_legacyScannerEnabled` based on `scanner.disable` and injects `ROX_LEGACY_SCANNER` as an environment variable on the Central deployment. With scanner v2 always disabled, this needs to be updated so `_legacyScannerEnabled` is always `false` (and `ROX_LEGACY_SCANNER=false` is injected on Central).

**File**: [image/templates/helm/stackrox-central/templates/01-central-13-deployment.yaml.htpl](image/templates/helm/stackrox-central/templates/01-central-13-deployment.yaml.htpl)

Reads `_legacyScannerEnabled` to emit `ROX_LEGACY_SCANNER` env var on Central -- will now always be `false`.

### 11. roxctl Scanner Commands

**File**: [roxctl/scanner/generate/generate.go](roxctl/scanner/generate/generate.go)
- The `--scanner-image` flag references Scanner v2 image. Remove this flag (keep `--scanner-v4-image` and `--scanner-v4-db-image`).

**File**: [roxctl/central/generate/k8s.go](roxctl/central/generate/k8s.go)
- Flags `--scanner-image`, `--scanner-db-image` for V2. Remove these.

**File**: [roxctl/central/generate/interactive.go](roxctl/central/generate/interactive.go)
- Interactive prompts for scanner/scanner-db images. Remove V2 prompts.

### 12. Deploy Script `k8sbased.sh`

**File**: [deploy/common/k8sbased.sh](deploy/common/k8sbased.sh)

Core deploy logic: passes `--scanner-image`/`--scanner-db-image` to roxctl; sets Helm `scanner.disable`; launches V2 scanner; applies `scanner-patch.yaml`, `scanner-local-patch.yaml`, `scanner-hpa-patch.yaml`. Update to skip all scanner v2 deploy steps.

**Related patch files to remove/empty**:
- [deploy/common/scanner-patch.yaml](deploy/common/scanner-patch.yaml) -- V2 replicas=2
- [deploy/common/scanner-local-patch.yaml](deploy/common/scanner-local-patch.yaml) -- V2 local resources
- [deploy/common/scanner-hpa-patch.yaml](deploy/common/scanner-hpa-patch.yaml) -- V2 HPA

### 13. Secured Cluster Image Defaults

**File**: [image/templates/helm/stackrox-secured-cluster/internal/defaults/50-images.yaml.htpl](image/templates/helm/stackrox-secured-cluster/internal/defaults/50-images.yaml.htpl)

Remove the `image.scanner` and `image.scannerDb` sections (registry, name, tag, repository, fullRef) -- these reference the slim scanner v2 images and are dead code with scanner v2 always disabled. Keep `image.scannerV4` and `image.scannerV4DB`.

### 14. Helm Unit Tests

**Critical test files**:
- [pkg/helm/charts/tests/centralservices/testdata/helmtest/scanner.test.yaml](pkg/helm/charts/tests/centralservices/testdata/helmtest/scanner.test.yaml) -- Full scanner v2 chart tests (deployment, services, netpol, PSP). These must be updated: tests that assert scanner v2 resources exist will fail since scanner is now always disabled.
- [pkg/helm/charts/tests/centralservices/testdata/helmtest/scanner-v4.test.yaml](pkg/helm/charts/tests/centralservices/testdata/helmtest/scanner-v4.test.yaml) -- V4 coexistence tests that assert "enabling V4 keeps V2 enabled". Update to reflect V2 is always disabled.
- [pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/scanner-slim.test.yaml](pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/scanner-slim.test.yaml) -- Secured cluster slim scanner tests.
- [pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/scanner-v4.test.yaml](pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/scanner-v4.test.yaml) -- V2 default/upgrade rules.

### 15. Operator Tests

- Translation tests: `operator/internal/central/values/translation/translation_test.go`, `operator/internal/securedcluster/values/translation/translation_test.go`, `operator/internal/values/translation/translation_test.go`
- Defaulting tests: `operator/internal/central/extensions/reconcile_defaulting_test.go`, `operator/internal/securedcluster/extensions/reconcile_defaulting_test.go`
- Scanner DB password: `operator/internal/central/extensions/reconcile_scanner_db_password_test.go`
- TLS tests: `operator/internal/central/extensions/reconcile_tls_test.go`
- Auto-sense: `operator/internal/securedcluster/scanner/auto_sense_test.go`
- Kuttl E2E fixtures: `operator/tests/common/central-cr-assert.yaml`, `central-cr-without-scanner-v4.yaml`, and other files that assert `scanner`/`scanner-db` deployments exist
- Keep [tests/feature_flag_test.go](tests/feature_flag_test.go) as-is (it already skips `ROX_LEGACY_SCANNER`)

### 16. E2E Bats Tests

**File**: [tests/e2e/run-scanner-v4-install.bats](tests/e2e/run-scanner-v4-install.bats)

This file calls `verify_scannerV2_deployed` (~20 times) across all test cases. With scanner v2 removed, specific assertions must flip to `verify_no_scannerV2_deployed` (helper already exists at line ~1090).

**Principle**: Only flip v2 verification calls. Do NOT remove or modify any deployment configuration in the test suite (scanner v2 resource settings, HPA patches, `SENSOR_SCANNER_SUPPORT` exports, etc.) -- these are harmless since v2 is always disabled, and keeping them minimizes the diff.

#### Test-by-test changes

##### (a) "Upgrade from old Helm chart to HEAD Helm chart with Scanner v4 enabled" (line ~445)

Structure: deploy old chart (central + sensor), upgrade to HEAD chart.

- **Old chart deployment** (lines 465, 487): Keep `verify_scannerV2_deployed` -- the old chart still deploys v2.
- **After upgrade to HEAD** (lines 474, 495): Change to `verify_no_scannerV2_deployed` -- HEAD chart must not deploy v2.
- Consider adding `verify_deployment_deletion_with_timeout` for `scanner` and `scanner-db` before the no-v2 assertion, to handle any deletion lag during the Helm upgrade.

**Note**: The upgrade uses `--reuse-values`, which carries forward the old chart's `scanner.disable: false`. The Helm chart init template (plan item 10) must unconditionally force `scanner.disable=true`, overriding any user-supplied or reused value. This is a hard requirement of the Helm chart implementation, not just a default change.

##### (b) "Fresh installation of HEAD Helm charts in different namespaces and toggling Scanner V4" (line ~502)

- Lines 540, 546, 567, 574: Flip all `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (c) "Fresh installation of HEAD Helm charts in the same namespace and toggling Scanner V4" (line ~604)

- Lines 632, 659: Flip `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (d) "Fresh installation of HEAD Helm charts with Scanner V4 enabled in multi-namespace mode" (line ~694)

- Line 712: Flip `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (e) "[Manifest Bundle] Fresh installation without Scanner V4, adding Scanner V4 later" (line ~728)

- Line 747: Flip `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (f) "[Operator] Fresh installation with Scanner V4 enabled" (line ~768)

- Line 792: Flip `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (g) "[Operator] Fresh multi-namespace installation with Scanner V4 enabled" (line ~804)

- Lines 829, 833: Flip both `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (h) "[Operator] Upgrade multi-namespace installation" (line ~840)

- **Before upgrade** (lines 866, 869): Keep `verify_scannerV2_deployed` -- old operator still deploys v2.
- **After upgrade** (lines 888, 891): Flip to `verify_no_scannerV2_deployed`.

##### (i) "Fresh installation using roxctl with Scanner V4 enabled" (line ~928)

- Line 945: Flip `verify_scannerV2_deployed` to `verify_no_scannerV2_deployed`.

##### (j) "Upgrade from old version without Scanner V4 to HEAD with Scanner V4 enabled" (line ~952)

- **Before upgrade** (line 968): Keep `verify_scannerV2_deployed` -- old roxctl deploys v2.
- **After upgrade** (line 980): Flip to `verify_no_scannerV2_deployed`.

## What We Are NOT Changing (follow-up work)

- `ROX_LEGACY_SCANNER` feature flag definition and its Go default remain `enabled` -- runtime Go code continues to work
- `pkg/scanners/clairify/` package -- untouched
- `central/cve/fetcher/` orchestrator CVE scanning code -- untouched
- `central/sensor/service/pipeline/nodeinventory/` and `nodes/` pipelines -- untouched
- `sensor/common/scannerclient/` V2 gRPC client -- untouched
- `central/scannerdefinitions/` V2 definitions handler -- untouched
- `central/scanner/handler.go` scanner zip bundle generation -- untouched
- UI hooks (`useIsLegacyScannerEnabled`) and conditional rendering -- untouched
- CRD fields (kept for backward compat, marked deprecated)
- CI/CD workflows (`.github/workflows/update_scanner_periodic.yaml`, `.tekton/retag-scanner*.yaml`, etc.) -- separate follow-up
- `SCANNER_VERSION` file -- separate follow-up
- `scripts/ci/is-scanner-v2-available.sh` and CI-specific scripts -- separate follow-up
- QA backend tests (`qa-tests-backend/`) -- separate follow-up
- E2E bats tests -- covered in this plan (see todo `e2e-bats`)

## Execution Order

The work is organized to minimize risk: defaults changes first (which disable scanner v2 through existing conditionals), then cleanup, then tests.
