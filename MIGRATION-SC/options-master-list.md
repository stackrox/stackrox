# roxctl sensor generate - Master Options List

This document lists every CLI option discovered across both modes of
`roxctl sensor generate`, noting which modes each option is available in.

Modes:
- **K8S**: `roxctl sensor generate k8s`
- **OS**: `roxctl sensor generate openshift`

## OpenShift-specific options

| Option | Default | K8S | OS | Description |
|--------|---------|-----|-----|-------------|
| `--disable-audit-logs` | `false` | - | Y | Disable audit log collection for runtime detection |
| `--openshift-version` | `0` | - | Y | OpenShift major version to generate deployment files for |

## Global options (available in both modes)

### Options that affect generated manifests

| Option | Default | Description |
|--------|---------|-------------|
| `--admission-controller-disable-bypass` | `false` | Disable the bypass annotations for the admission controller |
| `--admission-controller-enforcement` | `true` | Enforce security policies on the admission review request |
| `--admission-controller-fail-on-error` | `false` | Fail the admission review request in case of errors or timeouts |
| `--auto-lock-process-baselines` | `false` | Locks process baselines when their deployments leave the observation period |
| `--central` | `central.stackrox:443` | Endpoint that sensor should connect to |
| `--collection-method` | `default` | Which collection method to use for runtime support (none, default, core_bpf) |
| `--collector-image-repository` | *(none)* | Image repository collector should be deployed with |
| `--create-upgrader-sa` | `true` | Whether to create the upgrader service account |
| `--disable-tolerations` | `false` | Disable tolerations for tainted nodes |
| `--enable-pod-security-policies` | `false` | Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes) |
| `--istio-support` | *(none)* | Generate deployment files supporting the given Istio version |
| `--main-image-repository` | *(none)* | Image repository sensor should be deployed with |
| `--name` | *(none)* | Cluster name to identify the cluster |

### Options with no impact on generated manifests

These options are client-side only or control output format.

| Option | Default | Description |
|--------|---------|-------------|
| `--ca` | *(none)* | Path to a custom CA certificate to use (PEM format) |
| `--continue-if-exists` | `false` | Continue with downloading the sensor bundle even if the cluster already exists |
| `--direct-grpc` | `false` | Client-side gRPC connection mode |
| `-e`, `--endpoint` | `localhost:8443` | Client connection endpoint |
| `--force-http1` | `false` | Client connection option |
| `-h`, `--help` | `false` | Shows help |
| `--insecure` | `false` | Client connection option |
| `--insecure-skip-tls-verify` | `false` | Client TLS validation |
| `--no-color` | `false` | Output color formatting |
| `--output-dir` | *(none)* | Output directory for bundle contents |
| `-p`, `--password` | *(none)* | Password for basic auth |
| `--plaintext` | `false` | Client connection option |
| `--retry-timeout` | `20s` | Timeout after which API requests are retried |
| `-s`, `--server-name` | *(none)* | Client TLS SNI |
| `-t`, `--timeout` | `5m0s` | Timeout for API requests |
| `--token-file` | *(none)* | Client authentication |
| `--use-current-k8s-context` | `false` | Client connection option |
