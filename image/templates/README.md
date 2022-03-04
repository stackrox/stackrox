# Working with helm

The currently maintained helm charts consists of the `stackrox-central-services`
and `stackrox-secured-cluster-services` charts.

Helm charts are distributed by `https://charts.stackrox.io` and [stackrox/helm-charts](https://github.com/stackrox/helm-charts).
The [stackrox/release-artifacts](https://github.com/stackrox/release-artifacts) repository takes care of the publishing automation.

## StackRox central services

Location: `./helm/stackrox-central`

Installs:
 - `central`
 - `scanner`

## StackRox secured cluster services

Location: `./helm/stackrox-secured-cluster`

Installs:
 - `admission-controller`
 - `sensor`
 - `collector`

## Developing helm charts

### Meta templating

To extend templating to, e.g., `Chart.yaml`, which in a normal helm chart cannot be templated `.htpl` files are used, which we call "meta templating".
The main difference is that `.htpl` files are rendered before loading the files as a Helm chart
(i.e., the rendered `.htpl` file is loaded for the helm chart). This pre-rendering takes place whenever the Helm charts are instantiated, e.g. during use by the operator or via roxctl.

To render files at the meta templating stage the `roxctl helm output` commands are used.

Meta templating does the following:

 - Merge files from `./shared/*` into both charts
 - Renders `.htpl` files based on passed values, see `pkg/helm/charts/meta.go`

### Workflow

This example shows how to work with the `stackrox central services` chart.

```
# Go to rox root
$ cdrox

# Receive the rendered helm chart from roxctl
# To use a custom template path use the `--debug-path=</path/to/templates>` argument.
$ ./bin/darwin/roxctl helm output central-services --image-defaults=development_build --debug

# Install the helm chart
$ helm upgrade --install -n stackrox stackrox-central-services --create-namespace  ./stackrox-central-services-chart \
    -f ./dev-tools/helm/central-services/docker-values-public.yaml \
    --set imagePullSecrets.username=<USERNAME> \
    --set imagePullSecrets.password=<PASSWORD>

# List all helm releases
$ helm list -n stackrox

# To uninstall central, run:
$ helm uninstall stackrox-central-services -n stackrox

# Delete the pvc if you want to reset the database
$ kubectl -n stackrox delete pvc stackrox-db

# To access central, forward port 443:
$ kubectl -n stackrox port-forward svc/central 8000:443 &
```

### Changing the charts

See [this document](CHANGING_CHARTS.md) which shows how to change the charts, for example to add a new values field.

### Testing

To test helm chart changes see `pkg/helm/charts/tests/{centralservices,securedclusterservices}`.

e.g.
```
$ cdrox
$ cd pkg/helm/charts/tests/centralservices
# go test -v
```

Tests are based on the [`helmtest` testing framework](https://github.com/stackrox/helmtest), see its [documentation](https://github.com/stackrox/helmtest/tree/main/docs) for an overview of `helmtest`. Some tips for using `helmtest`:

- Helm tests are launched from `pkg/helm/charts/tests/{centralservices,securedclusterservices}/helmtest_test.go`, that
  define regular Go unit tests based on the standard `testing` package. `helmtest` synthesizes children tests for those,
  that can be run individually with `go test`, which is useful for quick iteration. For example the tests in
  `pkg/helm/charts/tests/securedclusterservices/testdata/helmtest/audit-logs.test.yaml` can be run with
  `go test -v github.com/stackrox/rox/pkg/helm/charts/tests/securedclusterservices -run TestWithHelmtest/testdata/helmtest/audit-logs.test.yaml`.
- When writing a test, replacing assertions with the [`helmtest` function](https://github.com/stackrox/helmtest/blob/main/docs/functions.md)
  `print` can be helpful to inspect the objects where assertions are applied. To get YAML formatted output use `toyaml | print`.
- [`helmtest` documentation on World Model](https://github.com/stackrox/helmtest/blob/main/docs/world-model.md) specifies
  how to access from a test the different k8s objects in the rendered template. In particular `.objects` contains all
  k8s objects, but all object types are available as a map in the root object for easier access, e.g. `.deployments`,
  `clusterroles`, etc. This helps writing concise test: for example `.networkpolicys["scanner-slim"]` is equivalent to
  `[.objects[] | select(.kind == "NetworkPolicy" and .metadata.name == "scanner-slim")][0]`.
