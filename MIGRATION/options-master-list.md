# roxctl central generate - Master Options List

This document lists every CLI option discovered across all four modes of
`roxctl central generate`, noting which modes each option is available in,
how each option affects the generated kubectl manifests, and a `kubectl get`
command to infer from a running cluster whether the option was used.

Modes:
- **OS-PVC**: `roxctl central generate openshift pvc`
- **OS-HP**: `roxctl central generate openshift hostpath`
- **K8S-PVC**: `roxctl central generate k8s pvc`
- **K8S-HP**: `roxctl central generate k8s hostpath`

## Storage-specific options (PVC only)

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP |
|--------|---------|--------|-------|---------|--------|
| `--db-name` | `central-db` | Y | - | Y | - |
| `--db-size` | `100` | Y | - | Y | - |
| `--db-storage-class` | *(none)* | Y | - | Y | - |

### `--db-name`

Changes PVC name in `central/01-central-11-db-pvc.yaml` and the corresponding
`claimName` reference in the central-db Deployment (`central/01-central-12-central-db.yaml`).
Impact identical across OS-PVC and K8S-PVC.

```
kubectl get pvc -n stackrox -o jsonpath='{.items[*].metadata.name}'
```

### `--db-size`

Changes `spec.resources.requests.storage` in the PVC (`central/01-central-11-db-pvc.yaml`).
Default is `100Gi`. Impact identical across OS-PVC and K8S-PVC.

```
kubectl get pvc central-db -n stackrox -o jsonpath='{.spec.resources.requests.storage}'
```

### `--db-storage-class`

Adds `spec.storageClassName` to the PVC (`central/01-central-11-db-pvc.yaml`).
Absent by default (uses cluster default StorageClass). Impact identical across OS-PVC and K8S-PVC.

```
kubectl get pvc central-db -n stackrox -o jsonpath='{.spec.storageClassName}'
```

## Storage-specific options (hostpath only)

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP |
|--------|---------|--------|-------|---------|--------|
| `--db-hostpath` | `/var/lib/stackrox-central` | - | Y | - | Y |
| `--db-node-selector-key` | *(none)* | - | Y | - | Y |
| `--db-node-selector-value` | *(none)* | - | Y | - | Y |

### `--db-hostpath`

Changes `hostPath.path` in the `disk` volume of the central-db Deployment
(`central/01-central-12-central-db.yaml`). Impact identical across OS-HP and K8S-HP.

```
kubectl get deployment central-db -n stackrox -o jsonpath='{.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path}'
```

### `--db-node-selector-key` / `--db-node-selector-value`

These must be specified together. Adds a `nodeSelector` to the central-db Deployment
pod spec (`central/01-central-12-central-db.yaml`). Absent by default.
Impact identical across OS-HP and K8S-HP.

```
kubectl get deployment central-db -n stackrox -o jsonpath='{.spec.template.spec.nodeSelector}'
```

## OpenShift-specific options

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP |
|--------|---------|--------|-------|---------|--------|
| `--openshift-monitoring` | `auto` | Y | Y | - | - |
| `--openshift-version` | `0` | Y | Y | - | - |

### `--openshift-monitoring`

Default `auto` resolves to enabled on OpenShift. `--openshift-monitoring=true`
produces identical output to the default. `--openshift-monitoring=false` removes:
- `central/99-openshift-monitoring.yaml` (ServiceMonitor, PrometheusRule, RBAC for prometheus-k8s)
- `central/99-scanner-v4-openshift-monitoring.yaml` (ServiceMonitor, RBAC for scanner-v4)
- `central-monitoring-tls` NetworkPolicy from `central/01-central-10-networkpolicy.yaml`
- Monitoring port 9091, `ROX_ENABLE_SECURE_METRICS` env var, monitoring-tls volume/mount from central Deployment
- Monitoring port 9091, env vars, volume/mounts from scanner-v4 indexer/matcher Deployments
- `serving-cert-secret-name` annotation and monitoring port from central and scanner-v4 Services

Impact identical across OS-PVC and OS-HP.

```
kubectl get servicemonitor central-monitor-stackrox -n openshift-monitoring 2>/dev/null && echo "monitoring=true/auto" || echo "monitoring=false"
```

### `--openshift-version`

Default `0` resolves to version 4. `--openshift-version=4` produces identical output
to default. `--openshift-version=3` includes all changes from `--openshift-monitoring=false`
plus:
- Removes `central/00-injected-ca-bundle.yaml` (OCP4 trusted CA injection ConfigMap)
- Removes `scanner-v4/02-scanner-v4-01-security.yaml` (SCC Role/RoleBinding for nonroot-v2, restricted-v2)
- Removes OAuth redirect annotations from central ServiceAccount
- Removes `ROX_ENABLE_OPENSHIFT_AUTH` env var from central Deployment
- Removes `openshift.io/required-scc` annotations from all Deployments
- Removes `trusted-ca-volume` mounts from central, scanner, scanner-v4 Deployments
- Adds explicit `securityContext` (fsGroup/runAsUser: 4000) to config-controller Deployment

Impact identical across OS-PVC and OS-HP.

```
kubectl get configmap injected-cabundle-stackrox-central-services -n stackrox 2>/dev/null && echo "version=4/auto" || echo "version=3"
```

## Global options (available in all modes)

### `--main-image`

Default: derived from `--image-defaults`. Changes the Central and config-controller
container images in their Deployments. Also changes the registry prefix for scanner-v4
images and the setup script registry URL. Wide blast radius.

Files affected: `central/01-central-13-deployment.yaml`, `central/02-config-controller-02-deployment.yaml`,
`central/scripts/setup.sh`, `scanner/scripts/setup.sh`,
`scanner-v4/02-scanner-v4-07-{db,indexer,matcher}-deployment.yaml`.

Impact identical across all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--central-db-image`

Default: derived from `--image-defaults`. Changes both container images (init + main)
in the central-db Deployment (`central/01-central-12-central-db.yaml`). Clean, targeted.

Impact identical across all 4 modes.

```
kubectl get deployment central-db -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--scanner-image`

Default: derived from `--image-defaults`. Changes scanner container image in
`scanner/02-scanner-06-deployment.yaml`. Also changes the image pull secret name
from `stackrox` to `stackrox-scanner` and registry URL in `scanner/scripts/setup.sh`.

Impact identical across all 4 modes.

```
kubectl get deployment scanner -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--scanner-db-image`

Default: derived from `--image-defaults`. Changes scanner-db container images (init + main)
in `scanner/02-scanner-06-deployment.yaml`. Clean, targeted — does not affect setup scripts.

Impact identical across all 4 modes.

```
kubectl get deployment scanner-db -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--scanner-v4-image`

Default: derived from `--image-defaults`. Changes scanner-v4 indexer and matcher images in
`scanner-v4/02-scanner-v4-07-{indexer,matcher}-deployment.yaml`.

**Note:** Only the image name+tag portion is used; the provided registry is silently
discarded and replaced with the default registry prefix. This is inconsistent with
`--main-image`, `--central-db-image`, `--scanner-image`, and `--scanner-db-image`.

Impact identical across all 4 modes.

```
kubectl get deployment scanner-v4-indexer -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--scanner-v4-db-image`

Default: derived from `--image-defaults`. Changes scanner-v4-db container images (init + main)
in `scanner-v4/02-scanner-v4-07-db-deployment.yaml`.

**Note:** Same registry-discarding behavior as `--scanner-v4-image`.

Impact identical across all 4 modes.

```
kubectl get deployment scanner-v4-db -n stackrox -o jsonpath='{.spec.template.spec.containers[0].image}'
```

### `--image-defaults`

Default: `rhacs`. When set to `opensource`, switches all images from
`registry.redhat.io/advanced-cluster-security/rhacs-*-rhel8` to
`quay.io/stackrox-io/*` and updates setup script registry URLs.
Affects every deployment file and both setup scripts.

Impact identical across all 4 modes.

```
kubectl get deploy -n stackrox -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.image}{" "}{end}{"\n"}{end}'
```

### `--lb-type`

Default: `none`. Controls how Central is exposed externally by adding
`central/01-central-15-exposure.yaml`:
- `lb`: Adds a `LoadBalancer` Service named `central-loadbalancer` (port 443, externalTrafficPolicy: Local)
- `np`: Adds a `NodePort` Service named `central-loadbalancer` (port 443, with GCP app-protocol annotations)
- `route` (OpenShift only): Adds two OpenShift Routes — `central` (passthrough TLS with redirect) and `central-mtls` (host: central.stackrox)
- `none`: No exposure file added (default)

Impact identical across applicable modes.

```
kubectl get svc central-loadbalancer -n stackrox -o jsonpath='{.spec.type}' 2>/dev/null || kubectl get route central -n stackrox 2>/dev/null
```

### `--plaintext-endpoints`

Default: none. Adds `ROX_PLAINTEXT_ENDPOINTS` env var with the specified value to
7 containers across 6 Deployments: central, config-controller, scanner (both containers),
scanner-v4-db, scanner-v4-indexer, scanner-v4-matcher. No new files.

Impact identical across all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_PLAINTEXT_ENDPOINTS")].value}'
```

### `--istio-support`

Default: none. Appends Istio `DestinationRule` resources (apiVersion `networking.istio.io/v1alpha3`)
to 5 service YAML files. Each rule disables Istio mTLS for the service's ports since
StackRox uses its own mTLS. Affected files:
- `central/01-central-14-service.yaml`
- `scanner/02-scanner-07-service.yaml` (2 rules: scanner + scanner-db)
- `scanner-v4/02-scanner-v4-08-{db,indexer,matcher}-service.yaml`

Impact identical across all 4 modes.

```
kubectl get destinationrules -n stackrox
```

### `--declarative-config-config-maps`

Default: `[]`. Adds a read-only volumeMount at
`/run/stackrox.io/declarative-configuration/<name>` and a corresponding
configMap volume (optional: true) to the Central Deployment
(`central/01-central-13-deployment.yaml`). No new files.

Impact identical across all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{.spec.template.spec.volumes[?(@.configMap.name=="<name>")]}'
```

### `--declarative-config-secrets`

Default: `[]`. Same as `--declarative-config-config-maps` but mounts a Secret
volume instead of a ConfigMap. Adds volumeMount + secret volume (optional: true)
to Central Deployment. No new files.

Impact identical across all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{.spec.template.spec.volumes[?(@.secret.secretName=="<name>")]}'
```

### `--default-tls-cert` / `--default-tls-key`

These must be specified together. Adds a new file
`central/01-central-06-default-tls-cert-secret.yaml` containing a `kubernetes.io/tls`
Secret named `central-default-tls-cert` with the provided cert and key.
No changes to existing files.

Impact identical across all 4 modes.

```
kubectl get secret central-default-tls-cert -n stackrox
```

### `--disable-admin-password`

Default: `false`. Adds a `central.adminPassword.value` entry to the generated-values
secret (`central/99-generated-values-secret.yaml`). The `password` file is empty.
No other manifest structural changes.

Impact identical across all 4 modes.

```
kubectl get secret -n stackrox -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep stackrox-generated
```

### `--enable-pod-security-policies`

Default: `false`. Adds 4 new files containing PodSecurityPolicy resources (deprecated K8s API):
- `central/01-central-02-psps.yaml` — PSP `stackrox-central` + ClusterRole + RoleBinding
- `central/01-central-02-db-psps.yaml` — PSP `stackrox-central-db` + ClusterRole + RoleBinding
- `scanner/02-scanner-01-psps.yaml` — PSP `stackrox-scanner` + ClusterRole + RoleBinding
- `scanner-v4/02-scanner-v4-01-psps.yaml` — PSP `stackrox-scanner-v4` + ClusterRole + RoleBinding

No changes to existing files. Impact identical across all 4 modes.

```
kubectl get podsecuritypolicy stackrox-central stackrox-central-db stackrox-scanner stackrox-scanner-v4 2>/dev/null
```

### `--enable-telemetry=false`

Default: `true`. In central Deployment (`central/01-central-13-deployment.yaml`),
replaces `ROX_TELEMETRY_ENDPOINT` and `ROX_TELEMETRY_API_WHITELIST` env vars with
`ROX_TELEMETRY_STORAGE_KEY_V1=DISABLED`. Impact identical across all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{range .spec.template.spec.containers[0].env[*]}{.name}={.value}{"\n"}{end}' | grep TELEMETRY
```

### `--offline`

Default: `false`. Changes `ROX_OFFLINE_MODE` env var from `"false"` to `"true"` in
central Deployment (`central/01-central-13-deployment.yaml`). Impact identical across
all 4 modes.

```
kubectl get deployment central -n stackrox -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_OFFLINE_MODE")].value}'
```

## Options with no impact on generated manifests

These options are client-side only, control output format, or only affect
randomized values (passwords, certs) without changing manifest structure.

| Option | Default | Reason |
|--------|---------|--------|
| `--backup-bundle` | *(none)* | Not tested — requires actual backup bundle file |
| `--ca` | *(none)* | Client-side TLS trust option — silently ignored by `central generate` |
| `--direct-grpc` | `false` | Client-side gRPC connection mode |
| `-e`, `--endpoint` | `localhost:8443` | Client connection endpoint |
| `--force-http1` | `false` | Client connection option |
| `-h`, `--help` | `false` | Shows help |
| `--insecure` | `false` | Client connection option |
| `--insecure-skip-tls-verify` | `false` | Client TLS validation |
| `--no-color` | `false` | Output color formatting |
| `--output-dir` | *(none)* | Output directory path |
| `--output-format` | `kubectl` | Changes output format entirely (not tested — we only use kubectl) |
| `-p`, `--password` | *(autogenerated)* | Only affects password value, not manifest structure |
| `--plaintext` | `false` | Client connection option |
| `-s`, `--server-name` | *(none)* | Client TLS SNI |
| `--token-file` | *(none)* | Client authentication |
| `--use-current-k8s-context` | `false` | Client connection option |
