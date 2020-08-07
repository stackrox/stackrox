# Helm charts for the StackRox Kubernetes Security Platform 

After you [install StackRox Central](https://help.stackrox.com/docs/get-started/quick-start/#install-stackrox-central),
you can use helm charts to install Sensor, Collector, and Admission Controller.

- [Prerequisites](#prerequisites)
- [Install Sensor using Helm chart](#install-sensor-using-helm-chart)
- [Uninstall Sensor using Helm chart](#uninstall-sensor-using-helm-chart)
- [Upgrade Sensor using Helm chart](#upgrade-sensor-using-helm-chart)
- [Configuration](#configuration)

## Prerequisites
To install the StackRox Kubernetes Security Platform's Sensor, Collector, and
Admission Controller by using Helm charts, you must:

- Use Helm command-line interface (CLI) v2 or v3
- Use the StackRox Kubernetes Security Platform version 3.0.41 or newer.
- [Integrate the StackRox Kubernetes Security Platform with the image registry](https://help.stackrox.com/docs/integrate-with-other-tools/integrate-with-image-registries/)
you use.
- Have credentials for the `stackrox.io` registry or the other image registry
  you use.

> **IMPORTANT**
>
> We publish new Helm charts with every new release of the StackRox
> Kubernetes Security Platform. Make sure to use a version that matches the
> version of the StackRox Kubernetes Security Platform you've installed.

## Install Sensor using Helm chart

1. Generate an authentication Token with the `Sensor creator` role. See the
   [Authentication](https://help.stackrox.com/docs/use-the-api/#authentication)
   section of [Use the API](https://help.stackrox.com/docs/use-the-api/) topic.
1. After you have generated the authentication token, export it as
   `ROX_API_TOKEN` variable:
   ```bash
   export ROX_API_TOKEN=<api-token>
   ```
1. Navigate to the [stackrox/helm-charts](https://github.com/stackrox/helm-charts)
   repository, and download the helm charts folder that corresponds to the
   StackRox Kubernetes Security Platform version you are using. For example, if
   you are using the StackRox Kubernetes Security Platform version 3.0.42.0,
   download the folder named `3.0.42.0`.
1. From the downloaded folder, modify the `values.yaml` file based on your
   environment. See the [Configuration](#configuration) section to understand the
   available parameters.
1. Run the following command to generate secrets, certificates, and other
   required configurations:
   ```bash
   ./scripts/setup.sh -f <path-to-the-values-yaml-file> -e <central endpoint>
   ```
   - You must specify the path to the `values.yaml` file with `-f` parameter when
     you run the setup script. If you don't specify `values.yaml` file, the
     setup script uses the `values.yaml` file from the chart directory.
1. Run the Helm chart by using the following command:
   
   - For Helm v2:
     ```
     helm install --name sensor <path-to-the-directory-containing-the-charts> --namespace stackrox
     ```
   - For Helm v3:
     ```
     helm install sensor <path-to-the-directory-containing-the-charts> --namespace stackrox
     ```
   > **NOTE**
   > - You can't use the `-f` parameter to specify the `values.yaml` file when you run the install command.
   > - You must specify the namespace as `stackrox` by using the `--namespace stackrox` option with each helm command.

## Uninstall Sensor using Helm chart

1. Run the Helm chart by using the following command:
   
   - For Helm v2:
     ```
     helm delete --name sensor --namespace stackrox
     ```
   - For Helm v3:
     ```
     helm delete sensor --namespace stackrox
     ```
1. To verify if the Sensor is uninstalled, view the output of the following
   command:
   ```
   helm list --namespace stackrox
   ```

## Upgrade Sensor using Helm chart

> **NOTE**
>
> - You can only upgrade those Sensor installations that you've [installed using Helm charts](#install-sensor-using-helm-chart).
> - You don't have to run the setup script when you upgrade.

1. Navigate to the [stackrox/helm-charts](https://github.com/stackrox/helm-charts)
   repository, and download the helm charts folder that corresponds to the
   StackRox Kubernetes Security Platform version you are using. For example, if
   you are using the StackRox Kubernetes Security Platform version 3.0.41,
   download the folder named `3.0.41`.
1. In the downloaded folder, create a new directory called `secrets`. 
1. In the new `secrets` directory, copy the contents from the `secrets`
   directory of the release from which you are upgrading. For example, if you
   are upgrading from version 3.0.41.0 to 3.0.42.0, copy the contents of the
   directory `3.0.41.0/secrets` into `3.0.42.0/secrets`.   
1. Run the Helm upgrade command.
   - For Helm v2:
     ```
     helm upgrade --namespace stackrox --name sensor .
     ```
   - For Helm v3:
     ```
     helm upgrade --namespace stackrox sensor .
     ```

## Configuration

The following table lists the configurable parameters of the `values.yaml` file
and their default values.

|Parameter |Description | Default value |
|:---------|:-----------|:--------------|
|`cluster.name`| Name of your cluster. | |
|`cluster.type`| Either Kubernetes (`KUBERNETES_CLUSTER`) or OpenShift (`OPENSHIFT_CLUSTER`) cluster. |`KUBERNETES_CLUSTER` |
|`endpoint.central`| Address of the Central endpoint, including the port number (without a trailing slash). If you are using a non-gRPC capable LoadBalancer, use the WebSocket protocol by prefixing the endpoint address with `wss://`. |`central.stackrox:443` |
|`endpoint.advertised`| Address of the Sensor endpoint including port number.No trailing slash.|`sensor.stackrox:443` |
|`image.repository.main`|Repository from which to download the main image. |`main` |
|`image.repository.collector`|Repository from which to download the collector image.  |`collector` |
|`image.registry.main`| Address of the registry you are using for main image.|`stackrox.io` |
|`image.registry.collector`| Address of the registry you are using for collector image.|`collector.stackrox.io` |
|`config.collectionMethod`|Either `EBPF`, `KERNEL_MODULE`, or `NO_COLLECTION`. |`KERNEL_MODULE` |
|`config.admissionControl.createService`|This setting controls whether Kubernetes is configured to contact the StackRox Kubernetes Security Platform with `AdmissionReview` requests. |`false` |
|`config.admissionControl.listenOnUpdates`|When you keep it as `false`, the StackRox Kubernetes Security Platform creates the `ValidatingWebhookConfiguration` in a way that causes the Kubernetes API server not to send object update events. Since the volume of object updates is usually higher than the object creates, leaving this as `false` limits the load on the admission control service and decreases the chances of a malfunctioning admission control service.|`false` |
|`config.admissionControl.enableService`|It controls whether the StackRox Kubernetes Security Platform evaluates policies; if it’s disabled, all AdmissionReview requests are automatically accepted.  |`false` |
|`config.admissionControl.enforceOnUpdates`|This controls the behavior of the admission control service. You must specify `listenOnUpdates` as `true` for this to work. |`false`|
|`config.admissionControl.scanInline`| |`false` |
|`config.admissionControl.disableBypass`|Set it to `true` to disable [bypassing the admission controller](https://help.stackrox.com/docs/manage-security-policies/use-admission-controller-enforcement/). |`false` |
|`config.admissionControl.timeout`|The maximum time in seconds, the StackRox Kubernetes Security Platform should wait while evaluating admission review requests. Use it to set request timeouts when you enable image scanning. If the image scan runs longer than the specified time, the StackRox Kubernetes Security Platform accepts the request. Other enforcement options, such as scaling the deployment to zero replicas, are still applied later if the image violates applicable policies.|`3` |
|`config.registryOverride`|Use this parameter to override the default `docker.io` registry. Specify the name of your registry if you are using some other registry.| |
|`config.disableTaintTolerations`|If you specify `false`, tolerations are applied to collector, and the collector pods can schedule onto all nodes with taints. If you specify it as `true`, no tolerations are applied, and the collector pods won't scheduled onto nodes with taints. |`false` |
|`config.createUpgraderServiceAccount`| Specify `true` to create the `sensor-upgrader` account. By default, the StackRox Kubernetes Security Platform creates a service account called `sensor-upgrader` in each secured cluster. This account is highly privileged but is only used during upgrades. If you don’t create this account, you will have to complete future upgrades manually if the Sensor doesn’t have enough permissions. See [Enable automatic upgrades for secured clusters](https://help.stackrox.com/docs/configure-stackrox/enable-automatic-upgrades/) for more information.|`false` |
|`config.createSecrets`| Specify `false` to skip the orchestrator secret creation for the sensor, collector, and admission controller. | `true` |
|`config.offlineMode`| Specify `true` if you are installing sensor in offline mode so that StackRox Kubernetes Security Platform does not try to reach internet. By default, the StackRox Kubernetes Security Platform tries to reach internet.|`false` |
|`config.slimCollector`| Specify `true` if you want to use a slim Collector image for deploying Collector. Using slim Collector images requires Central to provide the matching kernel module or eBPF probe. If you are running the StackRox Kubernetes Security Platform in offline mode, you must download a kernel support package from [stackrox.io](https://install.stackrox.io/collector/support-packages/index.html) and upload it to Central for slim Collectors to function. Otherwise, you must ensure that Central can access the online probe repository hosted at https://collector-modules.stackrox.io/.|`false` |
|`envVars`| Specify environment variables for sensor and admission controller. Each environment variable will have a `Name` and a `Value`|`[]` |
