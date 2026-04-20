# roxctl central generate - Master Options List

This document lists every CLI option discovered across all four modes of
`roxctl central generate`, noting which modes each option is available in.

Modes:
- **OS-PVC**: `roxctl central generate openshift pvc`
- **OS-HP**: `roxctl central generate openshift hostpath`
- **K8S-PVC**: `roxctl central generate k8s pvc`
- **K8S-HP**: `roxctl central generate k8s hostpath`

## Storage-specific options (PVC only)

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP | Description |
|--------|---------|--------|-------|---------|--------|-------------|
| `--db-name` | `central-db` | Y | - | Y | - | External volume name for Central DB |
| `--db-size` | `100` | Y | - | Y | - | External volume size in Gi for Central DB |
| `--db-storage-class` | *(none)* | Y | - | Y | - | Storage class name for Central DB |

### `--db-name`

Changes PVC name in `central/01-central-11-db-pvc.yaml` and the corresponding
`claimName` reference in the central-db Deployment (`central/01-central-12-central-db.yaml`).
Impact identical across OS-PVC and K8S-PVC.

```
kubectl get pvc central-db -n stackrox -o jsonpath='{.metadata.name}'
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

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP | Description |
|--------|---------|--------|-------|---------|--------|-------------|
| `--db-hostpath` | `/var/lib/stackrox-central` | - | Y | - | Y | Path on the host |
| `--db-node-selector-key` | *(none)* | - | Y | - | Y | Node selector key (e.g. kubernetes.io/hostname) |
| `--db-node-selector-value` | *(none)* | - | Y | - | Y | Node selector value |

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

| Option | Default | OS-PVC | OS-HP | K8S-PVC | K8S-HP | Description |
|--------|---------|--------|-------|---------|--------|-------------|
| `--openshift-monitoring` | `auto` | Y | Y | - | - | Integration with OpenShift 4 monitoring |
| `--openshift-version` | `0` | Y | Y | - | - | The OpenShift major version (3 or 4) to deploy on |

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

### `--disable-admin-password`

Default: `false`. Adds a `central.adminPassword.value` entry to the generated-values
secret (`central/99-generated-values-secret.yaml`). The `password` file is empty.
No other manifest structural changes. Impact identical across all 4 modes.

```
kubectl get secret -n stackrox -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep stackrox-generated
```
Then inspect the generated-values secret for a `central.adminPassword` key.

### `--password`

Default: autogenerated. Only affects the password value, not manifest structure.
After stripping randomness, produces identical output to the baseline.
**No structural impact on manifests.** Impact identical across all 4 modes.

No `kubectl get` command — the password cannot be inferred from the cluster
(it's stored as a bcrypt hash in the `central-htpasswd` secret).

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

### `--direct-grpc`

Default: `false`. **No impact on generated manifests.** This is a client-side
flag controlling how roxctl connects to Central. Impact identical (zero) across
all 4 modes.

### Remaining global options — not yet tested

| Option | Default | Description | Notes |
|--------|---------|-------------|-------|
| `--ca` | *(none)* | Path to a custom CA certificate to use (PEM format) | |
| `--central-db-image` | *(none)* | The central-db image to use | |
| `--declarative-config-config-maps` | `[]` | List of config maps to add as declarative configuration mounts in central | |
| `--declarative-config-secrets` | `[]` | List of secrets to add as declarative configuration mounts in central | |
| `--default-tls-cert` | *(none)* | PEM cert bundle file | |
| `--default-tls-key` | *(none)* | PEM private key file | |
| `--image-defaults` | `rhacs` | Default container images settings (rhacs, opensource) | |
| `--istio-support` | *(none)* | Generate deployment files supporting the given Istio version | K8S modes add "(kubectl output format only)" |
| `--lb-type` | `none` | The method of exposing Central | OpenShift: route, lb, np, none. K8S: lb, np, none |
| `-i`, `--main-image` | *(none)* | The main image to use | |
| `--plaintext-endpoints` | *(none)* | The ports or endpoints to use for plaintext exposure; comma-separated list | |
| `--scanner-db-image` | *(none)* | The scanner-db image to use | |
| `--scanner-image` | *(none)* | The scanner image to use | |
| `--scanner-v4-db-image` | *(none)* | The scanner-v4-db image to use | |
| `--scanner-v4-image` | *(none)* | The scanner-v4 image to use | |

## Options with no impact on generated manifests

These options are client-side only or control output format, not manifest content.

| Option | Default | Description |
|--------|---------|-------------|
| `--backup-bundle` | *(none)* | Path to backup bundle (not tested — requires actual bundle) |
| `--direct-grpc` | `false` | Client-side gRPC connection mode |
| `-e`, `--endpoint` | `localhost:8443` | Client connection endpoint |
| `--force-http1` | `false` | Client connection option |
| `-h`, `--help` | `false` | Shows help |
| `--insecure` | `false` | Client connection option |
| `--insecure-skip-tls-verify` | `false` | Client TLS validation |
| `--no-color` | `false` | Output color formatting |
| `--output-dir` | *(none)* | Output directory path |
| `--output-format` | `kubectl` | Changes output format entirely (not tested — we only use kubectl) |
| `--password` | *(autogenerated)* | Only affects password value, not manifest structure |
| `--plaintext` | `false` | Client connection option |
| `-s`, `--server-name` | *(none)* | Client TLS SNI |
| `--token-file` | *(none)* | Client authentication |
| `--use-current-k8s-context` | `false` | Client connection option |
