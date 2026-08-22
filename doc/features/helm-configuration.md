# Helm Configuration and Meta-Templating

Two-stage templating system enabling customized Helm charts for different product flavors, build types, and deployment modes from a single source tree.

**Primary Packages**: `pkg/helm`, `image/templates/helm`

## What It Does

StackRox uses "meta-templating" to process chart files before standard Helm rendering. This separates build-time decisions (flavor, versions, feature flags) from runtime configuration (user values). Custom delimiters `[< >]` handle meta-templates while standard `{{ }}` serve Helm templates.

The system generates charts for StackRox vs RHACS flavors, release vs development builds, and various deployment modes without maintaining separate chart copies.

## Architecture

### Two-Stage Rendering

**Stage 1 - Meta-templating**: Processes `.htpl` files during roxctl execution, operator startup, or Central chart serving. Uses `[< >]` delimiters, takes chart templates + MetaValues, outputs standard Helm charts.

**Stage 2 - Helm Rendering**: Standard `helm install/upgrade/template` processing. Uses `{{ }}` delimiters, takes charts + user values, outputs Kubernetes manifests.

### MetaValues Structure

The `pkg/helm/charts/meta.go` MetaValues type contains version information (ChartVersion, MainVersion), image configuration (registry, remote, tags for Central/DB/Collector/Scanner), chart metadata (repo URL, icon, flavor), feature flags map, and deployment settings (operator mode, admission controller config, offline mode, baseline auto-locking).

Factory function `GetMetaValuesForFlavor()` creates populated instances for RHACS_BRANDING, STACKROX_BRANDING, and DEVELOPMENT_BUILD flavors.

### Chart Template System

The `pkg/helm/template/chart_template.go` ChartTemplate type handles meta-template processing. File type handling:
- `.htpl`: Meta-template files (suffix removed after rendering)
- `.hnotpl`: Explicitly non-template (escape hatch)
- Other files: Pass through unchanged

Key methods include `Load()` for loading from buffered files, `InstantiateRaw()` for rendering with meta-values returning raw files, and `InstantiateAndLoad()` for rendering and loading as Helm chart.

Special features: `.helmtplignore` for meta-template filtering, `helmTplKeepEmptyFile` function to force keeping empty files, automatic empty file suppression.

## Charts

### stackrox-central-services

Located in `image/templates/helm/stackrox-central/`, deploys Central (API server, policy engine, UI), Central DB (PostgreSQL 15), Scanner (legacy, optional), Scanner-v4 (indexer, matcher, db), and Config Controller.

Structure includes `Chart.yaml.htpl` and `values.yaml.htpl` for metadata, `config/` for default config files, `internal/` with config-shape.yaml schema, defaults.yaml.htpl, and platform overrides, plus `templates/` with 40+ manifest files.

Key templates: `01-central-13-deployment.yaml.htpl` (Central deployment), `01-central-12-central-db.yaml` (DB StatefulSet), `01-central-08-configmap.yaml.htpl` (configuration), `02-config-controller-02-deployment.yaml` (controller).

### stackrox-secured-cluster-services

Located in `image/templates/helm/stackrox-secured-cluster/`, deploys Sensor, Admission Controller, Collector DaemonSet, and optional Scanner/Scanner-v4.

Structure includes chart metadata, `internal/` with config-shape.yaml schema, compatibility-translation.yaml for legacy values, and multi-stage defaults in `defaults/` (00-bootstrap through 70-scanner-v4).

Multi-stage defaults process in alphanumeric order: 00-bootstrap.yaml.htpl (API server lookups), 10-env.yaml.htpl (platform detection), 20-tls-files.yaml (certificates), 30-base-config.yaml.htpl (core config), 40-resources.yaml (limits), 50-images.yaml.htpl (image refs), 70-scanner-v4.yaml.htpl (Scanner-v4).

Key templates: `sensor.yaml.htpl`, `admission-controller.yaml`, `collector.yaml.htpl`.

## Configuration System

### Config Shape

The `internal/config-shape.yaml` defines all possible values with types. Example entries specify null types indicating string, bool, or dict values.

### Defaults

The `internal/defaults.yaml.htpl` provides meta-templated default values. Examples include config file references with fallback paths, image names/tags from MetaValues, and resource requests/limits.

### Expandables

The `internal/expandables.yaml` marks fields supporting environment variable expansion using ${ENV_VAR} syntax.

### Template Initialization

The `templates/_init.tpl.htpl` loads config-shape, applies defaults, merges user values, performs expansions, validates, and exports to `._rox` namespace. All templates access final configuration via `._rox`.

## Common Customizations

**Image Overrides**: Set `image.registry` and `image.tag` for custom registries.

**Resource Limits**: Adjust `central.resources`, `sensor.resources` requests and limits.

**External Database**: Set `central.db.enabled: false` and configure `external` settings.

**Scanner-v4**: Configure replicas, resources, autoscaling for indexer and matcher.

**Admission Controller**: Set listen/enforce flags, timeout, bypass policy.

**Network Policies**: Enable with `network.policies.enabled: true`.

## Instantiation

**roxctl**: Commands like `roxctl helm output central-services --output-dir ./chart` invoke `roxctl/helm/output/output.go`.

**Operator**: Uses `image/embed_charts.go` via `image.GetDefaultImage().LoadChart()` with flavor-specific MetaValues.

**Central**: Service in `central/helmcharts/service_impl.go` serves instantiated charts via gRPC for sensor bundles.

## Testing

The `pkg/helm/charts/testutils/` framework uses `github.com/stackrox/helmtest` for snapshot testing. Tests use `RunHelmTestSuite()` with flavor overrides and MetaValuesOverridesFunc for custom settings.

Test data structure: `tests/testdata/helmtest/test-case-1/` contains `values.yaml` input and `expected.yaml` output.

Running tests: `go test -v ./pkg/helm/charts/tests/centralservices` for specific charts.

## Development

**Adding New Values**:

1. Add to `internal/config-shape.yaml` with type annotation
2. Add default in `internal/defaults.yaml.htpl`
3. Use in template via `._rox.newFeature.setting`
4. Add tests in `pkg/helm/charts/tests/`

See `image/templates/CHANGING_CHARTS.md` for detailed workflow.

## Utilities

**Custom Rendering** (`pkg/helm/util/render.go`): Allows rendering for different K8s versions via Options type with KubeVersion, HelmVersion, APIVersions overrides.

**File Filtering** (`pkg/helm/util/filter.go`): Implements `.helmignore` support with pattern matching, root-only paths, directory-only patterns, and negation.

**Values Utilities** (`pkg/helm/util/values.go`): Functions for creating nested maps (`ValuesForKVPair`) and merging (`CoalesceTables`).

## Integration

**Operator Translation**: Modules in `operator/internal/*/values/translation/` convert CRD specs to Helm values with flavor-specific MetaValues.

**Central Deployment**: The `central/clusters/deployer.go` generates sensor bundles with certificates, packages charts with defaults, and injects cluster config.

**roxctl**: The `roxctl/helm/output/output.go` formats charts for kubectl apply, generates archives, supports debug mode with feature flags.

## Recent Changes

Recent enhancements addressed ROX-32881 (policy webhook gating), ROX-33319 (admission controller monitoring), ROX-32630 (OCP plugin in SecuredCluster), ROX-32412 (compliance metrics), ROX-30729 (fact service feature flag), ROX-30937/30578 (baseline auto-locking), ROX-30608 (new admission config endpoint), ROX-30278 (static admission fields), and ROX-30034 (failure policy options).

## Implementation

**Core**: `pkg/helm/charts/meta.go`, `pkg/helm/template/chart_template.go`, `pkg/helm/template/extra_funcs.go`
**Charts**: `image/templates/helm/stackrox-central/`, `image/templates/helm/stackrox-secured-cluster/`
**Testing**: `pkg/helm/charts/testutils/`, `pkg/helm/charts/tests/`
**Utilities**: `pkg/helm/util/render.go`, `pkg/helm/util/filter.go`, `pkg/helm/util/values.go`
