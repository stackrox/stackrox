# StackRox Monitoring Chart

INTERNAL USE ONLY. Deploys Prometheus, Grafana, Alertmanager, and kube-state-metrics for StackRox development and debugging.

## Install

```bash
export PAGERDUTY_INTEGRATION_KEY=dummy   # required; real key for PagerDuty
helm dependency update deploy/charts/monitoring
helm upgrade --install monitoring deploy/charts/monitoring \
  -n stackrox --create-namespace \
  -f deploy/charts/monitoring/values.yaml
```

`PAGERDUTY_INTEGRATION_KEY` is substituted into the Alertmanager config. Use a dummy value when you do not need real paging.

Roxie can install the same chart as add-on release `roxie-addon-monitoring` via `central.availableAddOns.monitoring`.

## OpenShift

When the cluster exposes `security.openshift.io/v1/SecurityContextConstraints`, the chart:

1. Creates a namespace Role that allows `use` of SCC `nonroot-v2`.
2. Binds that Role to the chart ServiceAccounts: `monitoring`, `<release>-alertmanager`, and `<release>-kube-state-metrics`.

Subchart pods use UID/GID `1000`/`2000` (same as the parent monitoring Deployment) so they satisfy `nonroot-v2` without needing `anyuid`. On vanilla Kubernetes the SCC Role/RoleBinding is not rendered.

Do not set alertmanager / kube-state-metrics `nameOverride` or `fullnameOverride` unless you also update the SCC RoleBinding subjects in `templates/00-serviceaccount.yaml` to match the generated ServiceAccount names.

## Resources

Default Prometheus container requests/limits are `512Mi`/`1Gi` memory. For smaller or contended workers:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "250m"
```

## Design notes

- Prefer `nonroot-v2` over `anyuid`: least privilege, namespace-scoped `use` bindings only.
- Prefer parent `values.yaml` overrides over forking the upstream alertmanager / kube-state-metrics charts.
- Pod-level `runAsUser` on Alertmanager covers the configmap-reload sidecar, which has no container `securityContext` upstream.
