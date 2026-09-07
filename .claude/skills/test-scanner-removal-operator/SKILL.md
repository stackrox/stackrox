---
name: test-scanner-removal-operator
description: "Reference for deploying ACS via Operator (roxie) for Scanner V2 removal testing: install, upgrade, pre-4.8 simulation, cleanup"
disable-model-invocation: true
---

# ACS Operator (roxie) Deployment Reference

How to deploy and manage ACS via the Operator using roxie for testing Scanner V2 removal.
roxie lives at `~/go/src/github.com/stackrox/roxie/`.

## Namespaces

By default, roxie creates `acs-central` and `acs-sensor` namespaces.
With `--single-namespace`, everything goes into `stackrox`.

## Scanner configuration

Scanner settings are configured via roxie config files or `--set` flags that map
to CR spec fields.

### Config file format

```yaml
# Example: enable Scanner V2, default Scanner V4
central:
  spec:
    scanner:
      scannerComponent: Enabled
    scannerV4:
      scannerComponent: Enabled

securedCluster:
  spec:
    scanner:
      scannerComponent: Enabled
    scannerV4:
      scannerComponent: Enabled
```

### Mapping install settings to CR values

| Setting | V2 CR field (`scanner.scannerComponent`) | V4 CR field (`scannerV4.scannerComponent`) |
|---|---|---|
| default | omit (don't include in config) | omit |
| enabled | `Enabled` | `Enabled` |
| disabled | `Disabled` | `Disabled` |

In 5.0, the V2 field is marked obsolete and ignored, but setting it tests graceful handling.

### Using --set instead of a config file

```bash
roxie deploy --tag <version> \
  --set central.spec.scanner.scannerComponent=Enabled \
  --set central.spec.scannerV4.scannerComponent=Disabled \
  --set securedCluster.spec.scanner.scannerComponent=Enabled \
  --set securedCluster.spec.scannerV4.scannerComponent=Disabled
```

## Install

### Full deployment (Central + Secured Cluster)

```bash
roxie deploy --tag 4.11.3 --envrc /tmp/roxie-envrc -c <config-file>
```

`--envrc` avoids spawning a sub-shell (required for automation).
`--early-readiness` makes roxie only wait for central/sensor, not all workloads.

### Central only

```bash
roxie deploy central --tag 4.11.3 --envrc /tmp/roxie-envrc -c <config-file>
```

### Secured Cluster only

```bash
roxie deploy secured-cluster --tag 4.11.3 --envrc /tmp/roxie-envrc -c <config-file>
```

roxie handles CRS/init-bundle generation automatically — no manual step needed.

## Upgrade (4.11.3 → 5.0)

To upgrade, deploy only the operator with the new tag. This upgrades the operator,
which then reconciles the existing CRs with the new version:

```bash
roxie deploy operator --tag <5.0-tag> --envrc /tmp/roxie-envrc
```

The 5.0 tag is the output of `make tag` in the stackrox repo (e.g., `5.0.x-223-gde68adf2a2`).
No need to wait for Scanner V2 to be ready before upgrading — only Central needs to be up.

## Pre-4.8 simulation

To simulate an installation that was originally done before 4.8 (when Scanner V4 was
not yet the default), apply these annotations after installing 4.11.3:

```bash
kubectl annotate central stackrox-central-services -n acs-central \
  feature-defaults.platform.stackrox.io/scannerV4=Disabled --overwrite

kubectl annotate securedcluster stackrox-secured-cluster-services -n acs-sensor \
  feature-defaults.platform.stackrox.io/scannerV4=Disabled --overwrite
```

Adjust namespace to `stackrox` if using `--single-namespace`.

This makes the operator treat the Scanner V4 default as Disabled, mimicking a cluster
that was installed pre-4.8 and upgraded to 4.11.3 without explicitly enabling V4.
The operator will reconcile and remove V4 components. Wait for reconciliation before
proceeding with the upgrade.

## Verification commands

### Check deployments

```bash
# Central namespace (acs-central or stackrox)
kubectl get deployments -n <central-ns> \
  -o custom-columns='NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image'

# Secured Cluster namespace (acs-sensor or stackrox)
kubectl get deployments -n <sc-ns> \
  -o custom-columns='NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,IMAGE:.spec.template.spec.containers[0].image'
```

Scanner deployment names: `scanner`, `scanner-db` (V2), `scanner-v4-indexer`,
`scanner-v4-matcher`, `scanner-v4-db` (V4).

### Check CR status (operator-specific)

```bash
kubectl get central stackrox-central-services -n <central-ns> \
  -o jsonpath='{range .status.conditions[*]}{.type}: {.status} — {.message}{"\n"}{end}'

kubectl get securedcluster stackrox-secured-cluster-services -n <sc-ns> \
  -o jsonpath='{range .status.conditions[*]}{.type}: {.status} — {.message}{"\n"}{end}'
```

Verify neither CR is in `Irreconcilable` state:
```bash
kubectl get central stackrox-central-services -n <central-ns> \
  -o jsonpath='{.status.conditions[?(@.type=="Irreconcilable")].status}'
# Should be empty or "False"
```

### Check Central env vars

```bash
kubectl get deployment central -n <central-ns> \
  -o jsonpath='{range .spec.template.spec.containers[?(@.name=="central")].env[*]}{.name}={.value}{"\n"}{end}' | \
  grep -E "^ROX_(LEGACY_SCANNER|SCANNER_V4)="
```

Post-5.0: `ROX_LEGACY_SCANNER` should be `false`, `ROX_SCANNER_V4` should be `true`
(when V4 is enabled).

### Check sensor env vars

```bash
kubectl get deployment sensor -n <sc-ns> \
  -o jsonpath='{range .spec.template.spec.containers[?(@.name=="sensor")].env[*]}{.name}={.value}{"\n"}{end}' | \
  grep -E "^ROX_(LOCAL_IMAGE_SCANNING_ENABLED|SCANNER_V4|SCANNER_GRPC_ENDPOINT)="
```

## Cleanup

```bash
roxie teardown
```

Or for specific components:
```bash
roxie teardown central
roxie teardown secured-cluster
```
