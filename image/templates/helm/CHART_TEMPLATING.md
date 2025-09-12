# Helm Chart Templating in ACS

This document explains how we produces Helm charts from the codebase in ACS, focusing on the meta-templating system that allows us to customize charts based on build-time parameters.

## Overview

ACS maintains two Helm charts:
- `stackrox-central` - Central Services chart
- `stackrox-secured-cluster` - Secured Cluster Services chart

These charts use a custom "meta-templating" system that processes chart files before they become standard Helm charts. This allows us to influence the final chart structure based on build-time parameters like version strings, image references, flavor (opensource vs RHACS/), feature flags and more.

## Meta-Templating System

### Why Meta-Templating?

Meta-templating solves the challenge of producing different Helm chart configurations from a single source. It allows us to:

1. **Inject build-time values**: Version strings, image references, and registry information
2. **Support multiple flavors**: Generate different charts for opensource vs RHACS distributions  
3. **Feature Flag integration**: Conditionally include/exclude chart sections based on feature flag settings
4. **Maintain single source**: Avoid duplicating chart templates for different deployment scenarios

Files that require meta-templating are identified by the `.htpl` suffix (Helm template). For example:
- `Chart.yaml.htpl` → `Chart.yaml`
- `_init.tpl.htpl` → `_init.tpl`

### Meta-Template Syntax

Meta-templates use Go templating with custom delimiters `[<` and `>]` instead of the standard `{{` and `}}`:

```yaml
# Standard Helm templating (in final chart):
name: {{ .Values.name }}

# Meta-templating (in .htpl files):
name: stackrox-central-services
version: [< required "" .Versions.ChartVersion >]
appVersion: [< required "" .Versions.MainVersion >]
```

The standard Go template whitespace elimination rules apply:
- `[<-` removes leading whitespace
- `->]` removes trailing whitespace

### Meta-Template Variables (MetaValues)

During meta-templating, variables are accessed via the top-level context (`.`). The available variables are defined in the `MetaValues` struct at `pkg/helm/charts/meta.go` and include:

- **Version Information**: `.Versions.ChartVersion`, `.Versions.MainVersion`
- **Image References**: `.ImageRemote`, `.ImageTag`, `.ScannerImageRemote`
- **Flavor Configuration**: `.ChartRepo.IconURL`, `.ImagePullSecrets`
- **Feature Flags**: `.FeatureFlags` (map of all feature flag settings)
- **Build Information**: `.ReleaseBuild`, `.TelemetryEnabled`

### Feature Flags in Meta-Templates

Feature flags provide a way to customize chart behavior during feature development. They can be accessed in templates like this:

```yaml
[<- if .FeatureFlags.ROX_ADMISSION_CONTROLLER_CONFIG >]
  listenOnCreates: true
[<- else >]
  listenOnCreates: false
[<- end >]
```

Feature Flags are defined in `pkg/features/list.go` and their default values are used during chart instantiation.

## Chart Instantiation Process

**"Instantiating"** a Helm chart refers to the process of processing meta-templated files (`.htpl`) to produce a standard Helm chart. This is distinct from Helm's own "rendering" process that occurs when installing or templating a chart.

### Code References

The chart instantiation process is implemented in several key files:

1. **Main Entry Point**: `roxctl/helm/output/output.go`
   ```go
   renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(cfg.chartTemplatePathPrefix, chartMetaValues)
   ```

2. **Core Implementation**: `image/embed_charts.go`
   ```go
   func (i *Image) LoadAndInstantiateChartTemplate(chartPrefixPath ChartPrefix, metaVals *charts.MetaValues) ([]*loader.BufferedFile, error)
   ```

3. **Template Processing**: `pkg/helm/template/chart_template.go`
   ```go
   func (t *ChartTemplate) InstantiateRaw(metaVals *charts.MetaValues) ([]*loader.BufferedFile, error)
   ```

4. **MetaValues Creation**: `pkg/helm/charts/meta.go`
   ```go
   func GetMetaValuesForFlavor(imageFlavor defaults.ImageFlavor) *MetaValues
   ```

### Using roxctl for Chart Instantiation

The standard way to instantiate Helm charts is using `roxctl helm output`:

```bash
roxctl helm output central-services
roxctl helm output secured-cluster-services
```

### Development Mode with --debug

For quick iteration during Helm chart development, use the `--debug` parameter:

```bash
roxctl helm output central-services --debug
```

**Important**: The `--debug` flag is only available in non-release builds. It causes roxctl to use chart template files directly from the repository instead of the templates embedded in the roxctl binary. This allows for rapid testing of chart changes without rebuilding roxctl.

### Feature Flag Configuration

By default, `roxctl helm output` uses the default feature flag configuration from `pkg/features/list.go`. You can override specific feature flags using environment variables:

```bash
ROX_ADMISSION_CONTROLLER_CONFIG=true ROX_COMPLIANCE_ENHANCEMENTS=false roxctl helm output secured-cluster-services
```

### Legacy Deployment Integration

For the legacy deployment method using "manifest bundles", the Helm charts are instantiated in a particular style that differs from the standard roxctl output. This process also uses the same meta-templating system but may apply different MetaValues depending on the deployment context.

## Terminology

**Meta-templating vs Helm Rendering**: We use "instantiation" for our meta-templating phase to avoid confusion with Helm's own "rendering" process:

- **Chart Instantiation**: Processing `.htpl` files with MetaValues to produce a standard Helm chart
- **Helm Rendering**: Using `helm template` or `helm install` to process a Helm chart with values

This distinction is important when debugging template issues or understanding the two-stage templating process.
