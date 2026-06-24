# Deploying StackRox Dev Versions

## Deploying with roxie (recommended)

The legacy deploy scripts have served us well, but over time their configuration
grew into a maze of environment variables scattered across many files, each
subtly influencing deployment behavior. This makes them hard to maintain,
hard to debug, and hard to reproduce deployment configurations easily.

**[roxie](https://github.com/stackrox/roxie)** is the ACS/StackRox deployment
tool intended to replace the deployment shell scripts entirely.

Its core idea is simple: It uses **one self-contained configuration
file for reproducible deployments**, instead of implicitly picking up dozens of
environment variables.

### Running roxie directly

The recommended way is to install roxie natively on your machine (see
https://github.com/stackrox/roxie for installation instructions) and use it
directly from the command line, e.g.

```bash
roxie deploy --tag <stackrox main image tag>
```

for deploying the whole stack consisting of the operator, Central and SecuredCluster.

#### Shell script wrapper

For addressing muscle memory habits, `deploy.sh` can invoke roxie under the hood when you
opt-in into this behavior:

```bash
export USE_ROXIE_DEPLOY=true
/deploy/deploy.sh
```

Any extra arguments you pass to `deploy.sh` are forwarded directly to roxie,
so you can use roxie CLI flags even with `deploy.sh`, e.g.:

```bash
./deploy/deploy.sh --tag 4.11.0
```

The roxie backend in `deploy.sh` is **opt-in** for now. Over time, we expect it to
become opt-out, and eventually the legacy deployment scripts will be removed entirely.

### Configuration: environment variables vs. config file

The old scripts picked up configuration from the shell environment -- feature
flags, image tags, storage settings, and more. This is convenient at first, but
becomes a maintenance burden: it's unclear which variables are active, what
their defaults are, and whether your colleague's deployment matches yours.

With roxie, **we intentionally do not inherit environment variables**. Instead,
all configuration lives in a YAML config file that you pass explicitly. This
makes deployments self-documenting and reproducible.

#### Example: enabling a feature flag

**Before (environment variables):**

```bash
ROX_NETWORK_GRAPH_EXTERNAL_IPS=true ./deploy/deploy.sh
```

**After (roxie CLI):**

```bash
roxie deploy --features ROX_NETWORK_GRAPH_EXTERNAL_IPS
```

The `--features` flag accepts a comma-separated list. Prefix with `-` to disable:

```bash
roxie deploy --features ROX_NETWORK_GRAPH_EXTERNAL_IPS,-ROX_CISA_KEV
```

**Via deploy.sh with roxie under the hood:**

```bash
./deploy/deploy.sh --features ROX_NETWORK_GRAPH_EXTERNAL_IPS
```

### The roxie config file

Instead of juggling `--set` flags, you can write a YAML config file. This is
the recommended approach for anything beyond the simplest deployments. The
config file has two main sections -- `central` and `securedCluster` -- whose
`spec` fields correspond directly to the
[Central](https://docs.openshift.com/acs/operating/manage-central.html) and
[SecuredCluster](https://docs.openshift.com/acs/operating/manage-secured-clusters.html)
CRD specifications. This means you can customize your deployment in any way
the CRDs support.

Here's an example config file (`my-deployment.yml`):

```yaml
roxie:
  # Image tag to deploy can also be set in the config file.
  version: "4.11.0"

central:
  namespace: stackrox
  spec:
    central:
      exposure:
        loadBalancer:
          # Expose Central via load balancer
          enabled: true
    # Feature flags and other env vars for Central:
    customize:
      envVars:
      - name: ROX_NETWORK_GRAPH_EXTERNAL_IPS
        value: "true"
      - name: ROX_BASELINE_GENERATION_DURATION
        value: "5m"
    # Scanner V4 configuration:
    scannerV4:
      scannerComponent: Enabled

securedCluster:
  namespace: stackrox
  spec:
    clusterName: my-cluster
    # Feature flags for Sensor:
    customize:
      envVars:
      - name: ROX_INIT_CONTAINER_SUPPORT
        value: "true"
    # Per-node configuration:
    perNode:
      fileActivityMonitoring:
        mode: Enabled
```

Deploy with it:

```bash
roxie deploy --config my-deployment.yml

# Or via deploy.sh:
./deploy/deploy.sh --config my-deployment.yml
```

The `spec` paths mirror the CRDs, so anything you can configure in a
`Central` or `SecuredCluster` custom resource, you can configure here. Check
the CRD documentation or the operator source in `/operator/` for the full
set of available fields.

### Image tag

The roxie flow in `deploy.sh` also supports `MAIN_IMAGE_TAG` for convenience:

```bash
MAIN_IMAGE_TAG=4.11.0 ./deploy/deploy.sh
```

But the recommended approach is the explicit CLI flag:

```bash
roxie deploy --tag 4.11.0
```

### Feedback

We're actively improving roxie and would love your feedback! If you run into
issues, have feature requests, or just want to share your experience, please
reach out to the **ACS Install team**.

---

## Legacy deployment flow

> **Note:** The sections below describe the legacy `kubectl`-based deployment
> flow. Consider switching to roxie (see above) for a better experience.

### Usage

```
# Deploy scripts should be used from the git root of this repo
# Deploy StackRox locally on Kubernetes
$ ./deploy/k8s/deploy-local.sh

# Deploy StackRox locally on OpenShift
$ ./deploy/openshift/deploy-local.sh

# Deploy StackRox on a remote OpenShift cluster with an exposed route
$ LOAD_BALANCER=route ./deploy/openshift/deploy.sh
```

### Env variables

Most environment variables can be found in [common/env.sh](common/env.sh).

| **Name**                             | **Values**            | **Description**                                                                                                                                                                                                                                                            |
|--------------------------------------|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `COLLECTION_METHOD`                  | `core_bpf`            | Set the collection method for collector.                                                                                                                                                                                                                                   |
| `ROX_HOTRELOAD`                      | `true`  \ `false`     | `HOTRELOAD` mounts Sensor and Central local binaries into locally running pods. Only works with docker-desktop.  Alternatively you can use ./dev-tools/enabled-hotreload.sh. Note however that this will break the linter: https://stack-rox.atlassian.net/browse/ROX-6562 |
| `LOAD_BALANCER`                      | `route` \ `lb`        | Configure how to expose Central, important if deployed on remote clusters. Use `route` for OpenShift, `lb` for Kubernetes.                                                                                                                                                 |
| `MAIN_IMAGE_TAG`                     | `string`              | Configure the image tag of the `stackrox/main` image to be deployed.                                                                                                                                                                                                       |
| `MONITORING_SUPPORT`                 | `true`  \ `false`     | Enable StackRox monitoring.                                                                                                                                                                                                                                                |
| `MONITORING_ENABLE_PSP`              | `true` \ `false`      | Generate PodSecurityPolicies for monitoring. Defaults to `false`, as PSPs were deprecated in k8s 1.25.                                                                                                                                                                     |
| `REGISTRY_USERNAME`                  | `string`              | Set docker registry username to pull the docker.io/stackrox/main image.                                                                                                                                                                                                    |
| `REGISTRY_PASSWORD`                  | `string`              | Set docker registry password to pull the docker.io/stackrox/main image.                                                                                                                                                                                                    |
| `STORAGE`                            | `none`  \ `pvc`       | Defines which storage to use for the Central database, to preserve data between Central restarts it is recommended to use `pvc`.                                                                                                                                           |
| `SENSOR_DEV_RESOURCES`               | `true`  \ `false`     | (defaults to `true`) When set to true, Sensor will be deployed with reduced memory/cpu requests. This should be used exclusively for testing and development environments.                                                                                                 |
| `ROX_LOCAL_SOURCE_PATH`              | `string`              | When `ROX_HOTRELOAD` is enabled this variable sets the path to the local binary. This is useful when the `hostPath` mount links into a VM or container, e.g. when using KIND.                                                                                              |
| `ROX_INIT_BUNDLE_PATH`               | `string`              | Sets a custom init-bundle file path for Sensor.                                                                                                                                                                                                                            |
| `ROX_CENTRAL_EXTRA_HELM_VALUES_FILE` | `string`              | Adds a custom value file path to the Central Helm chart.                                                                                                                                                                                                                   |
| `ROX_SENSOR_EXTRA_HELM_VALUES_FILE`  | `string`              | Adds a custom value file path to the Sensor Helm chart.                                                                                                                                                                                                                    |
