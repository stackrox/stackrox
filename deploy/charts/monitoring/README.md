# StackRox Monitoring Chart

INTERNAL USE ONLY. Deploys Prometheus, Grafana, Alertmanager, and kube-state-metrics for StackRox development and debugging.

## Install

```bash
export PAGERDUTY_INTEGRATION_KEY=dummy   # required; real key for PagerDuty
helm dependency update deploy/charts/monitoring
# ${PAGERDUTY_INTEGRATION_KEY} in values.yaml needs envsubst, or pass --set below.
envsubst < deploy/charts/monitoring/values.yaml > /tmp/monitoring-values.yaml
helm upgrade --install monitoring deploy/charts/monitoring \
  -n stackrox --create-namespace \
  -f /tmp/monitoring-values.yaml
```

Or without `envsubst`:

```bash
helm upgrade --install monitoring deploy/charts/monitoring \
  -n stackrox --create-namespace \
  -f deploy/charts/monitoring/values.yaml \
  --set "alertmanager.config.receivers[0].pagerduty_configs[0].service_key=${PAGERDUTY_INTEGRATION_KEY}"
```

`enableMonitoringPSPs` defaults to `false` (PSPs are gone on current clusters). Set `--set enableMonitoringPSPs=true` only on clusters that still expose the PodSecurityPolicy API.

On OpenShift, if Helm reports an `imagePullSecrets` conflict on a ServiceAccount (OpenShift's image-registry controller vs Helm server-side apply), retry with `--force-conflicts`.

Roxie can install the same chart as add-on release `roxie-addon-monitoring` via `central.availableAddOns.monitoring`.

## OpenShift

When the cluster exposes `security.openshift.io/v1/SecurityContextConstraints`, the chart:

1. Creates a namespace Role that allows `use` of SCC `nonroot-v2`.
2. Binds that Role to the chart ServiceAccounts: `monitoring`, `<release>-alertmanager`, and `<release>-kube-state-metrics`.

Subchart pods use UID/GID `1000`/`2000` (same as the parent monitoring Deployment) so they satisfy `nonroot-v2` without needing `anyuid`. On vanilla Kubernetes the SCC Role/RoleBinding is not rendered.

Do not set alertmanager / kube-state-metrics `nameOverride` or `fullnameOverride` unless you also update the SCC RoleBinding subjects in `templates/00-serviceaccount.yaml` to match the generated ServiceAccount names.

## Resources

Prometheus defaults to a `2Gi` memory request, no memory limit, and a `1000m` CPU limit. Fixed memory limits OOMKill this chart under normal scrape load; omit that limit and let the process grow. Set `resources.limits.memory` in values if you need a hard cap.

## Design notes

- Prefer `nonroot-v2` over `anyuid`: least privilege, namespace-scoped `use` bindings only.
- Prefer parent `values.yaml` overrides over forking the upstream alertmanager / kube-state-metrics charts.
- Pod-level `runAsUser` on Alertmanager covers the configmap-reload sidecar, which has no container `securityContext` upstream.
- Docker Hub `imagePullSecrets` (`stackrox`) are set on Pods, not ServiceAccounts, so OpenShift's SA dockercfg controller does not fight Helm on upgrade.
