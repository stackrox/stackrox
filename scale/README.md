## Manual performance testing with the workload scale simulator

This quickstart guide is for anyone looking to quickly test performance scenarios against a scaled cluster, e.g. manual testing of the ACS UI.

To start a scale simulation, run the following from the root of the stackrox repo:

**Note that this is a destructive operation on your current active `kubectx` cluster that cannot be easily undone.**

```sh
cd scale
./launch_workload.sh <workload_name>
# <workload_name> is the name of a yaml file in the `workloads` directory, without file extension
# e.g. $ ./launch_workload.sh xlarge
```

## Overview

Running this script does the following:
- Deletes the `admission-control` deployment
- Deletes the `collector` daemonset
- Creates a configmap from the yaml file specified in the command, and mounts it under `/var/scale/stackrox/workload.yaml` in the `sensor` container
- Sets some standard CPU/MEM resource limits on stackrox deployments (likely for reproducible results in actual automated tests)

When sensor restarts, it will be put into a "fake" mode when it detects the presense of the `/var/scale/stackrox/workload.yaml` file. This fake
mode will cause sensor to use a mocked k8s client instead of the real client, and it will start sending data to central based
on the values in the provided `workload.yaml` file. While in this fake mode, `sensor` no longer will listen for actual events happening in the cluster.

Each of the top level "workload" keys in this yaml represent a different resource that will be scaled up in some fashion. Some of the
properties in this file can be omitted, but it isn't currently documented which ones. A safer bet for things you don't care to
test is to just reduce the numbers so that the impact on your system is minimal.

Some of the items in the yaml, like `nodeWorkload: numNodes`, simply add a number of items to the database, while workloads
like `deploymentWorkload` have an effect that continues over time. Brief details of what each property does are noted in the commented `workloads/sample.yaml` file.

To tweak scale test values, modify your workload.yaml and re-run the `./launch_workload.sh` script. This will delete the old configmap
and recreate it with the new yaml. Then you can restart sensor with `kubectl -n stackrox rollout pause deploy sensor` which should cause the new config to take effect.
