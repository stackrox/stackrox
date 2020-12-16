# Working with helm

The currently maintained helm charts consists of into the `stackrox-central-services`
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

### Deprecated charts

The following charts are deprecated and not under active development anymore.

 - `./helm/DEPRECATEDcentralchart`
 - `./helm/DEPRECATEDscannerchart`
 - `./helm/sensorchart`

## Developing helm charts

To extend templating to, e.g., `Chart.yaml`, which in a normal helm chart cannot be templated `.htpl` files are used.
The main difference is that `.htpl` files are rendered before loading the files as a Helm chart
(i.e., the rendered `.htpl` file is loaded for the helm chart).

### Workflow

This example shows how to work with the `stackrox central services` chart.

```
# Go to rox root
$ cdrox

# Building roxctl compiles the helm charts into the binary
$ make cli

# Receive the rendered helm chart from roxctl
$ ./bin/darwin/roxctl helm output central-services

# Install the helm chart
$ helm upgrade --install -n stackrox stackrox-central-services --create-namespace  ./stackrox-central-services-chart \
    -f ./dev-tools/helm/central-services/docker-values-public.yaml \
    --set-file licenseKey=./deploy/common/dev-license.lic \
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
