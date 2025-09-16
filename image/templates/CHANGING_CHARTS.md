# Changing StackRox Helm charts

## Helm Template Coding Style Guidelines

### Whitespace Elimination

Given that the sole purpose of the Helm templating is to render out Kubernetes manifests in YAML format,
it is critical that the rendering process does not mangle with whitespace indentation in an undesired manner.

Helm templating -- just like the underlying Go templating engine -- supports different mechanisms for
whitespace trimming, see [Go templating docs](https://pkg.go.dev/text/template#hdr-Text_and_spaces) for more information on this.

For our Helm charts we have settled on the following coding style:

For in-line interpolation we usually do not require any whitespace trimming. For example, we can write the templating code as follows:
```
metadata:
  namespace: {{ ._rox._namespace }}
```
```
      env:
        - name: ROX_OFFLINE_MODE
          value: {{ ._rox.env.offlineMode | quote }}
```

For line-based flow-control we use the left hand side whitespace trimming (`{{- ... }}`) exclusively. For example:
```
        {{- if ._rox.central.telemetry.storage.endpoint }}
        - name: ROX_TELEMETRY_ENDPOINT
          value: {{ ._rox.central.telemetry.storage.endpoint | quote }}
        {{- end }}
```
```
imagePullSecrets:
{{- range $secretName := ._rox.imagePullSecrets._names }}
- name: {{ quote $secretName }}
{{- end }}
```
This style provides consistency for the templating syntax and it enables us to indent the templating directives
without having any extra whitespace ending up in the rendered output or semantically required whitespace being silently removed.

## Add new values field to StackRox Helm Chart

This section describes how to add a new field to the Helm values, with unit tests.

[Small reference implementation](https://github.com/stackrox/stackrox/commit/98cc6bcd16f6d27170ab190d21e0ce8b835132b4) for the simple clusterLabels field.

## Breaking changes

Helm values are similar to public APIs or CLI parameters. When parameters change they might break CI pipelines or existing scripts.

* Helm values structure must be always backward compatible
* Defaults can change when the change is not going to break things
* Location and names of Helm charts, existing installations can't upgrade without updating to the new Helm chart metadata

### Notes / Tips

- Look at the [README](README.md) to see how to work with the Helm charts
- To install and update `SecuredClusters` faster in local development take a look at the `./dev-tools/upgrade-dev-secured-cluster.sh` script
- `.htpl` files are rendered by `roxctl`, you always need to render Helm charts via `roxctl` helm output

### Add a field to CentralServices or SecuredCluster chart:

This section describes how to add a simple field to a StackRox Helm chart and read it in the templates, and later add a unit test.

Suppose we add a field `clusterDescription` to the `SecuredClusterServices` chart and want to write it to as a label to the sensor deployment.

1. Locate the Helm chart you want to extend under `image/templates/helm/stackrox-secured-cluster`
1. Add the field `clusterDescription` to the `internal/config-shape.yaml` at the root level
1. The value is directly translated into the `._rox` variable which happens in the `stackrox-secured-clusters/templates/init.tpl.htpl`
1. If appropriate, add a default value in `internal/defaults.yaml.htpl`
1. The `stackrox-secured-clusters/templates` directory contains the later rendered templates
1. To read the value now add it to the Sensor deployment in `sensor.yaml.htpl`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sensor
  namespace: {{ ._rox._namespace }}
  labels:
    {{- include "srox.labels" (list . "deployment" "sensor") | nindent 4 }}
    app: sensor
    auto-upgrade.stackrox.io/component: "sensor"
    stackrox.io/description: "{{ ._rox.clusterDescription }}"
```

### Add a cluster config field:

Making a change that affects the Secured Cluster chart's cluster configuration (which is persisted in
Central and displayed in the UI) is more complex because the Helm Cluster
configuration is tracked in Central and needs adjustments to its conversion
logic.

1. Locate the Helm chart you want to extend under `image/templates/helm/stackrox-secured-cluster`
1. Add the desired field to the `config-shape.yaml.tpl` and add the type as a comment
1. Add the field to the `proto/storage/cluster.proto:CompleteClusterConfig` message, this is later used to keep track of the Helm configuration
1. Add the config field to the `image/templates/helm/stackrox-secured-cluster/internal/cluster-config.yaml.tpl` file. This is later rendered and mounted as a file from the [helm-cluster-config secret](helm/stackrox-secured-cluster/templates/cluster-config.yaml) into Sensor.
1. Add the conversion logic from the Helm config to a Cluster proto in `central/cluster/datastore/datastore_impl.go:configureFromHelmConfig()`. This conversion updates or creates the cluster to the returned `Cluster` proto.
   The conversion takes in the data read from Sensor from its `helm-cluster-config` secret
1. Regenerate `proto-srcs` and recompile central and sensor and deploy them. (You may want to mount binaries into pods directly with `./dev-tools/enable-hotreload.sh <component>`
   Add documentation to either the `public-values.yaml` or the `private-values.yaml`
1. After redeployment test if your field is applied to the Cluster config in Centrals Cluster Config UI.
   You can uninstall your SecuredCluster instance and reinstall a new one:

```
# Uninstall existing helm installations if necessary
$ helm -n stackrox uninstall stackrox-secured-cluster-services
 
# Create a new values.yaml with your desired test values.
$ cat > test-values.yaml <<- EOM
helmManaged: true
clusterLabels:
  value1: my-value1
EOM
 
# Reinstall helm installation defaulting to Central instances created by
# the deploy scripts
$ ./dev-tools/upgrade-dev-secured-cluster.sh -f test-values.yaml
```
