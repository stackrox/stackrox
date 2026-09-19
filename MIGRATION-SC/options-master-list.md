# roxctl sensor generate - Master Options List

This document lists every CLI option discovered across both modes of
`roxctl sensor generate`, noting which modes each option is available in,
how each option affects the generated manifests, and how to detect from
manifests/cluster whether the option was used.

Modes:
- **K8S**: `roxctl sensor generate k8s`
- **OS**: `roxctl sensor generate openshift`

## OpenShift-specific options

| Option | Default | K8S | OS | Description |
|--------|---------|-----|-----|-------------|
| `--disable-audit-logs` | `false` | - | Y | Disable audit log collection for runtime detection |
| `--openshift-version` | `0` | - | Y | OpenShift major version (only `4` is supported) |

### `--disable-audit-logs`

**No impact on generated manifests.** This is a server-side cluster configuration
setting stored in Central, not reflected in the sensor bundle files.

### `--openshift-version`

Default `0` resolves to version 4. `--openshift-version=4` produces identical
output to the default. `--openshift-version=3` is rejected with an error
("only '4' is currently supported").

## Global options (available in both modes)

### `--admission-controller-enforcement`

Default: `true`. When set to `false`, removes the `policyeval.stackrox.io`
ValidatingWebhookConfiguration entry from `admission-controller.yaml` (the
enforcement webhook that evaluates security policies on CREATE/UPDATE of
workload resources). The `check.stackrox.io` image-check webhook remains.

Detection: check if `policyeval.stackrox.io` webhook exists in `admission-controller.yaml`.

### `--admission-controller-fail-on-error`

Default: `false`. Changes `failurePolicy` from `"Ignore"` to `"Fail"` on the
ValidatingWebhookConfigurations in `admission-controller.yaml`. Affects both
the `check.stackrox.io` and `policyeval.stackrox.io` webhooks (if present).

Detection: check `failurePolicy` on the admission webhooks.

### `--central`

Default: `central.stackrox:443`. Changes the `ROX_CENTRAL_ENDPOINT` env var
value in `sensor.yaml` (sensor Deployment). Also reflected in `NOTES.txt`.

Detection: `ROX_CENTRAL_ENDPOINT` env var on the sensor Deployment.

### `--collection-method`

Default: `default` (resolves to `core_bpf`). When set to `none`, the collector
container is completely removed from the collector DaemonSet in `collector.yaml`
(only the compliance container remains). `core_bpf` produces identical output
to the default.

Detection: presence/absence of the `collector` container in the collector DaemonSet.

### `--collector-image-repository`

Default: derived from `--main-image-repository`. Changes the collector container
image in `collector.yaml`. Also changes the registry auth URL in `sensor.sh`.

Detection: collector container image in the collector DaemonSet.

### `--main-image-repository`

Default: derived from Central's image flavor. Changes the main image in:
- `sensor.yaml` (sensor Deployment)
- `admission-controller.yaml` (admission-controller Deployment)
- `collector.yaml` (compliance container image)

Also changes the collector image registry (derived from main) and setup script
registry URLs in `sensor.sh`.

Detection: container images on sensor, admission-controller, compliance containers.

### `--create-upgrader-sa`

Default: `true`. When set to `false`, removes `upgrader-serviceaccount.yaml`
entirely and changes `sensor.sh` to print a message about manual creation
instead of auto-applying the upgrader service account.

Detection: presence/absence of `upgrader-serviceaccount.yaml` or the
`sensor-upgrader` ServiceAccount.

### `--disable-tolerations`

Default: `false`. When set to `true`, removes the `tolerations` block
(`operator: Exists`) from the collector DaemonSet pod spec in `collector.yaml`.

Detection: presence/absence of tolerations on the collector DaemonSet.

### `--enable-pod-security-policies`

Default: `false`. When set to `true`, adds 3 new files:
- `admission-controller-pod-security.yaml`
- `collector-pod-security.yaml`
- `sensor-pod-security.yaml`

Each contains PodSecurityPolicy + ClusterRole + RoleBinding resources.

Detection: presence of these files / PSP resources.

### `--istio-support`

Default: none. Appends an Istio `DestinationRule` resource to `sensor.yaml`
(disabling Istio mTLS for port 443 on `sensor.stackrox.svc.cluster.local`).
Also adds an informational note to `NOTES.txt`.

Detection: presence of `DestinationRule` resources.

### `--name`

The cluster name. Stored in the `helm-effective-cluster-name` Secret's
`cluster-name` data field in `sensor.yaml`. Also appears in `NOTES.txt`.
The `--name` flag is required for `sensor generate`.

Detection: `helm-effective-cluster-name` Secret.

## Options with no impact on generated manifests

These options either affect server-side cluster configuration in Central
(not reflected in manifests), are client-side only, or control output format.

| Option | Default | Reason |
|--------|---------|--------|
| `--admission-controller-disable-bypass` | `false` | Server-side config stored in Central |
| `--auto-lock-process-baselines` | `false` | Server-side config stored in Central |
| `--ca` | *(none)* | Client-side TLS trust |
| `--continue-if-exists` | `false` | Controls CLI behavior, not manifests |
| `--disable-audit-logs` | `false` | Server-side config (OpenShift only) |
| `--direct-grpc` | `false` | Client-side connection mode |
| `-e`, `--endpoint` | `localhost:8443` | Client connection endpoint |
| `--force-http1` | `false` | Client connection option |
| `-h`, `--help` | `false` | Shows help |
| `--insecure` | `false` | Client connection option |
| `--insecure-skip-tls-verify` | `false` | Client TLS validation |
| `--no-color` | `false` | Output color formatting |
| `--openshift-version` | `0` | Only `4` supported; `0` and `4` produce identical output |
| `--output-dir` | *(none)* | Output directory path |
| `-p`, `--password` | *(none)* | Client authentication |
| `--plaintext` | `false` | Client connection option |
| `--retry-timeout` | `20s` | Client retry timing |
| `-s`, `--server-name` | *(none)* | Client TLS SNI |
| `-t`, `--timeout` | `5m0s` | Client request timeout |
| `--token-file` | *(none)* | Client authentication |
| `--use-current-k8s-context` | `false` | Client connection option |
