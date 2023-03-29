# Deploy scripts

## Usage

```
# Deploy sripts should be used from the git root of this repo
# Deploy StackRox locally on Kubernetes
$ ./deploy/k8s/deploy-local.sh

# Deploy StackRox locally on OpenShift
$ ./deploy/openshift/deploy-local.sh

# Deploy StackRox on a remote OpenShift cluster with an exposed route
$ LOAD_BALANCER=route ./deploy/openshift/deploy.sh
```

## Env variables

Most environment variables can be found in [common/env.sh](common/env.sh).

| **Name**                | **Values**            | **Description**                                                                                                                                                            |
|-------------------------|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `COLLECTION_METHOD`     | `ebpf`  \ `kernel-module` | Set the collection method for collector.                                                                                                                                   |
| `HOTRELOAD`             | `true`  \ `false`         | `HOTRELOAD` mounts Sensor and Central local binaries into locally running pods. Only works with docker-desktop.  Alternatively you can use ./dev-tools/enabled-hotreload.sh. Note however that this will break the linter: https://stack-rox.atlassian.net/browse/ROX-6562 |
| `LOAD_BALANCER`         | `route` \ `lb`            | Configure how to expose Central, important if deployed on remote clusters. Use `route` for OpenShift, `lb` for Kubernetes.                                                 |
| `MAIN_IMAGE_TAG`        | `string`                  | Configure the image tag of the `stackrox/main` image to be deployed.                                                                                                       |
| `MONITORING_SUPPORT`    | `true`  \ `false`         | Enable StackRox monitoring.                                                                                                                                                |
| `MONITORING_ENABLE_PSP` | `true` \ `false`          | Generate PodSecurityPolicies for monitoring. Defaults to `false`, as PSPs were deprecated in k8s 1.25. |
| `REGISTRY_USERNAME`     | `string`                  | Set docker registry username to pull the docker.io/stackrox/main image. |
| `REGISTRY_PASSWORD`     | `string`                  | Set docker registry password to pull the docker.io/stackrox/main image.  |
| `STORAGE`               | `none`  \ `pvc`           | Defines which storage to use for the Central database, to preserve data between Central restarts it is recommended to use `pvc`.                                                |
| `SENSOR_DEV_RESOURCES`  | `true`  \ `false`         | (defaults to `true`) When set to true, Sensor will be deployed with reduced memory/cpu requests. This should be used exclusively for testing and development environments.      |
