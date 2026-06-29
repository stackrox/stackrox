# Deploying StackRox Dev Versions

The legacy deploy scripts have served us well, but over time their configuration
grew into a maze of environment variables scattered across many files, each
subtly influencing deployment behavior. This makes them hard to maintain,
hard to debug, and hard to reproduce deployment configurations easily.

**[roxie](https://github.com/stackrox/roxie)** is the ACS/StackRox deployment
tool intended to replace the deployment shell scripts entirely.

Its core idea is simple: It uses **one self-contained configuration
file for reproducible deployments**, instead of implicitly picking up dozens of
environment variables.

## Deploying with roxie (recommended)

### Installation

Installation of roxie is straightforward:
1. there are executables for released versions which can be fetched from https://github.com/stackrox/roxie/releases
1. with a properly set up Go dev environment, the roxie repo can be checked out and roxie can be built
   with `make build` or `make install`.
1. there is the script wrapper `scripts/roxie.sh`, which handles the downloading of roxie automatically under
   the hood.
1. there is also a containerized version of roxie living at quay.io/rhacs-eng/roxie, which can be used ad-hoc.

Furthermore, the `deploy/deploy.sh` shell script supports a roxie backend: set `USE_ROXIE_DEPLOY=true` in the
environment and it will automatically download and run the roxie version specified in the `ROXIE_VERSION` file.

Any extra arguments you pass to `deploy.sh` are forwarded directly to roxie,
so you can use roxie CLI flags even with `deploy.sh`, e.g.:

```bash
./deploy/deploy.sh --tag 4.11.0
```

The roxie backend in `deploy.sh` is **opt-in** for now. Over time, we expect it to
become opt-out, and eventually the legacy deployment scripts will be removed entirely.

If you would like to use roxie directly, but roxie is not installed on your system, all `roxie`
commands below can also be run via `scripts/roxie.sh` instead:

```
❯ ./scripts/roxie.sh --help

roxie is a fast, developer-friendly CLI to deploy and manage
Red Hat Advanced Cluster Security (ACS) on any Kubernetes/OpenShift cluster.

Usage:
  roxie [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  deploy      Deploy ACS components
  help        Help about any command
  logs        View logs for ACS components
  shell       Open a subshell for an existing ACS Central deployment
  teardown    Teardown ACS components
  version     Print version information

Flags:
      --dry-run   Do not actually modify cluster
  -h, --help      help for roxie
  -v, --verbose   Enable verbose output (show CRs)

Use "roxie [command] --help" for more information about a command.
```

### Usage

roxie is designed as a standard CLI tool built around the notion of sub-commands.
The following focuses on the most important commands and flags, for a complete description
of roxies command line interface, please use `roxie --help`.

#### Configuration: environment variables vs. config file

The old scripts picked up configuration from the shell environment -- feature
flags, image tags, storage settings, and more. This is convenient at first, but
becomes a maintenance burden: it's unclear which variables are active, what
their defaults are, and whether your colleague's deployment matches yours.

With roxie, **we intentionally do not inherit environment variables** (apart from one: `MAIN_IMAGE_TAG`). Instead,
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

### Deployment

One of the most important commands is `roxie deploy <component>`, where component can be one of
1. central
1. securedcluster
1. both

Note that specifying "both" is optional -- leaving it out does the same thing.

For example, to deploy the whole stack to your current cluster context use
```
roxie deploy [ <roxie args> ... ] --tag <main tag>
```
or alternatively
```
export USE_ROXIE_DEPLOY=true
MAIN_IMAGE_TAG=<main tag> ./deploy/deploy.sh [ <roxie args> ... ]
```

There exist several flags for influencing the deployment behavior. With the exception of very few
special flags, there exist corresponding fields in a "roxie config YAML file", which can be passed
with `--config`. The config file has this structure:

```yaml
roxie:
  version: "<a main image tag>" # The same as `MAIN_IMAGE_TAG` used in other places in this repo.
  konfluxImages: <bool> # Only use this if you need to deploy downstream images
                        # (Konflux building pipelines must have passed for this to work).
  featureFlags:
    ROX_FEATURE_FLAG_A: "true"
    ROX_FEATURE_FLAG_B: "false"
  clusterType: "<string representation of the cluster type>" # Usually not needed, since this is auto-detected.

operator:
  skipDeployment: <bool> # Useful when running the operator using `make -C operator run` locally
                         # and roxie shouldn't interfere with that.
  deployViaOlm: <bool> # By default roxie deploys without olm.
  version: "<string representation of the operator tag>" # This is usually derived from roxie.version.
  envVars:
    ENV_VAR_A: "Value for ENV_VAR_A"
    ENV_VAR_B: "Value for ENV_VAR_B"
    # Prominent use-case: Setting the RELATED_IMAGE_... variables within the operator to
    # overwrite specific image references. Can be used, for example, to test an updated main image while
    # using the other images corresponding to some other tag.

central:
  namespace: "<namespace>" # By default this is "acs-central". A different popular choice is "stackrox".
  resourceProfile: "<profile name>" # Leave empty for using ACS default settings for resources.
                                    # Often a good idea to use "auto".
                                    # Important to keep in mind that even on some standard infra cluster flavors
                                    # (e.g. openshift-4), the "medium" resourceProfile might lead to some
                                    # resource shortage.
  pauseReconciliation: <bool> # By default this is false. In case the operand resources shall be modifiable without
                              # the operator's reconciler kicking in, set this to true.
  exposure: "<exposure type>" # Usually this can be unset, letting roxie pick a reasonable default.
                              # Only "lb" is currently supported.
  deployTimeout: "<duration string>" # Unless earlyReadiness is disabled, this can usually be left unset.
  portForwarding: <bool> # Can be usually left unset, a sensible default will be selected.
  earlyReadiness: <bool> # Defaults to true, meaning that roxie only waits for the central deployment.
                         # If disabled, roxie will wait for all operand workloads to be ready.
                         # Usually requires a longer `deployTimeout` to be configured, depending on several
                         # environment properties (e.g. speed of the node disk type, influencing the DB
                         # initialization for scanner).
  spec: <map> # This is a verbatim overlay for the Central CR spec and allows arbitrary customization of the CR.

securedCluster:
  namespace: "<namespace>" # By default this is "acs-sensor". A different popular choice is "stackrox".
  resourceProfile: "<profile name>" # Leave empty for using ACS default settings for resources.
                                    # Often a good idea to use "auto".
                                    # Important to keep in mind that even on some standard infra cluster flavors
                                    # (e.g. openshift-4), the "medium" resourceProfile might lead to some
                                    # resource shortage.
  pauseReconciliation: <bool> # By default this is false. In case the operand resources shall be modifiable without
                              # the operator's reconciler kicking in, set this to true.
  deployTimeout: "<duration string>" # Unless earlyReadiness is disabled, this can usually be left unset.
  earlyReadiness: <bool> # Defaults to true, meaning that roxie only waits for the sensor deployment.
                         # If disabled, roxie will wait for all operand workloads to be ready.
                         # Usually requires a longer `deployTimeout` to be configured, depending on several
                         # environment properties (e.g. speed of the node disk type, influencing the DB
                         # initialization for scanner).
  spec: <map> # This is a verbatim overlay for the SecuredCluster CR spec and allows arbitrary customization of the CR.
```

Note that roxie also supports "user config", which is automatically loaded by the deploy command
and contains overwritable defaults. On Linux systems the path of this user config file is usually
`~/.config/roxie/config.yaml` (or `$XDG_CONFIG_HOME/roxie/config.yaml`, if that environment variable is set).
On darwin the path of the file is `~/Library/Application Support/roxie/config.yaml`.

roxie supports two distinct modes of operation:
1. Interactive mode: after deployment a sub-shell is spawned in which the environment is set up automatically
   for interacting with central (most importantly endpoint and authentication information).
1. Non-interactive mode: after the deployment the environmental information for communicating with central
   is written into an envrc-style file which can then be used by the user as desired -- sourced, propagated, etc.

Most CLI flags of roxie implement short-cuts for certain patches of the in-memory representation of this deployment config.
For example, for the `roxie deploy` command:

1. `--tag/-t` corresponds to `roxie.version`
1. `--central-wait/--secured-cluster-wait` corresponds to `central.deployTimeout`/`securedCluster.deployTimeout`.
1. `--deploy-operator` corresponds to `operator.skipDeployment` (negated).
1. `--early-readiness` corresponds to `central.earlyReadiness`/`securedCluster.earlyReadiness`.
1. `--exposure` corresponds to `central.exposure`.
1. `--features` corresponds to `roxie.featureFlags`.
1. `--konflux` corresponds to `roxie.konfluxImages`.
1. `--olm` corresponds to `operator.deployViaOlm`.
1. `--pause-reconciliation` corresponds to `central.pauseReconciliation`/`securedCluster.pauseReconciliation`.
1. `--port-forwarding` corresponds to `central.portForwarding`.
1. `--resources` corresponds to `central.resourceProfile`/`securedCluster.resourceProfile`.
1. `--single-namespace` sets `central.namespace` and `securedCluster.namespace` to `stackrox`.
1. `--operator-env` corresponds to `operator.envVars`.

These CLI flags are special in the sense that they do not correspond to a specific fields in the config file:
1. `--set <YAML path>=<val>` can be used for arbitrary patches to the in-memory deployment config.
1. `--config/-c <file path>` can be used for specifying a roxie config file for the deployment.
1. `--envrc <path>` enables non-interactive mode, writing post-deployment information into the specified envrc file
   instead of spawning a sub-shell.
1. `--skip-user-config` disables automatic loading of the user config file.
1. `--image-preload-command <command>` can be used for configuring an "image preloading command" for deploying only
   locally existing images onto a local cluster. Preloading support for kind and minikube clusters is already built in.
   For other local cluster solutions it might be necessary to specify a preloading command. The provided command
   string will be executed in a shell where `$IMAGE` is bound to the value of the image which needs to be sent to the
   local cluster.

For debugging purposes these flags might be helpful:
1. `--dry-run` do not actually deploy.
1. `--verbose` dump verbose information during operation, including final custom resources.

### Tear down

Another important roxie command is `roxie teardown <component>`, where component is one of
1. central
1. securedcluster
1. both (meaning central + securedcluster)
1. all (meaning both + operator)

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
