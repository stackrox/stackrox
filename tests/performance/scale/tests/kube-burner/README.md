# Perf&Scale testing

## Setup cluster

You can use `infractl` to create a testing cluster. You will need different instance types for different workloads.

To create a cluster with CPU-optimized instances, you can run the following command:
```
export INFRA_NAME="perf-123"
export ARTIFACTS_DIR="/tmp/artifacts-${INFRA_NAME}"

infractl create openshift-4-perf-scale "${INFRA_NAME}" --description "Perf Test"
```

After the cluster is up and running, you can download artifacts and set Kubernetes context:
```
infractl artifacts "${INFRA_NAME}" --download-dir "${ARTIFACTS_DIR}"

export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"
```

## Deploy Central + Sensor

And run the following commands:
```
./utilities/start-central-and-scanner.sh "${ARTIFACTS_DIR}"
./utilities/wait-for-pods.sh "${ARTIFACTS_DIR}"
./utilities/get-bundle.sh "${ARTIFACTS_DIR}"
./utilities/start-secured-cluster.sh "${ARTIFACTS_DIR}"
./utilities/turn-on-monitoring.sh "${ARTIFACTS_DIR}"
./utilities/wait-for-pods.sh "${ARTIFACTS_DIR}"
```

## Download kube-burner

You can download kube-burner from https://github.com/cloud-bulldozer/kube-burner/releases.

An example:
```
export KUBE_BURNER_VERSION=1.4.3

mkdir -p ./kube-burner

curl --silent --location "https://github.com/cloud-bulldozer/kube-burner/releases/download/v${KUBE_BURNER_VERSION}/kube-burner-${KUBE_BURNER_VERSION}-$(uname -s)-$(uname -m).tar.gz" --output "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz"

tar -zxvf "./kube-burner/kube-burner-${KUBE_BURNER_VERSION}.tar.gz" --directory ./kube-burner

export KUBE_BURNER_PATH="$(pwd)/kube-burner/kube-burner"
alias kube-burner="${KUBE_BURNER_PATH}"
```

## Configure Indexer

The indexer is configured in the kube-burner config yaml global section. For run-workload.sh, this is set in cluster-density-template.yml with a condition to use "elastic" if ELASTICSEARCH_URL is defined in the environment:
```
export ELASTICSEARCH_URL=https://user:password@elasticserver
./run-workload.sh
```
This configuration can be added to other templates:
```yaml
global:
  gc: true
  indexerConfig:
    enabled: true
    {{ if env "ELASTICSEARCH_URL" -}}
    type: elastic  # "opensearch" can be used in kube-burner >=v1.6
    esServers: [ {{ env "ELASTICSEARCH_URL" }} ]
    defaultIndex: kube-burner
    {{ else -}}
    type: local
    metricsDirectory: collected-metrics
    createTarball: true
    tarballName: collected-metrics.tar.gz
    {{ end }}
```

## Run kube-burner workload

Workloads are defined in `tests/kube-burner` directory. You can use for example `cluster-density` workload.

Go to `tests/kube-burner/cluster-density`:
```
cd tests/kube-burner/cluster-density
```

Before running kube-burner command, you have to get Prometheus URL and token to access it. You can do it with the following commands:
```
export PROMETHEUS_URL="https://$(oc get route --namespace openshift-monitoring prometheus-k8s --output jsonpath='{.spec.host}' | xargs)"
export PROMETHEUS_TOKEN="$(oc serviceaccounts new-token --namespace openshift-monitoring prometheus-k8s)"
```

After that you can run the following command to run workload. It will create in total 100 namespaces, 500 deployments and 1000 pods.
```
kube-burner init --uuid=node-09--c2d-highcpu-8--dep-5--pod-2--workload-100--run-1 --config=cluster-density.yml --metrics-profile=metrics.yml --alert-profile=alerts.yml --skip-tls-verify --timeout=2h --prometheus-url="${PROMETHEUS_URL}" --token="${PROMETHEUS_TOKEN}"
```

This run will create a directory named `collected-metrics` and a tar file `collected-metrics.tar.gz` with the content of that directory.

To run different workflow sizes you can use provided script within `cluster-density` workload directory.

You can run the following command to create workload with 1000 namespaces, 10000 deployments and 10000 pods.
```
./run-workload.sh --kube-burner-path "${KUBE_BURNER_PATH}" --num-namespaces 1000 --num-deployments 10 --num-pods 1
```

Results will be stored in a tar file named by using the following pattern:

`node-<num-nodes>--<worker-instance-type>--dep-<num-deployments>--pod-<num-pods>--workload-<num-namespaces>--run-0.tar.gz`

## Analyse metrics

There is Google Colab Template that can be used to analyze results. It has helper functions to load results and visualize them.
It also contains several examples to analyze results.

To start with your analysis. Clone [ACS-Perf-Scale-Test-Results-Analysis-Template](https://colab.research.google.com/drive/1h_xgkCTubqjd_6hPQnp9iV_L0atUnE-0) and set name to match the goal of your analysis. Do all necessary work in your cloned Colab.

If you develop useful helper functions, consider including them in the base template with some examples by providing comments with code on the base template.

## Increase cluster size

To icrease cluster size you can scale machinesets. You can get current machinesets with the following command:
```
oc get machinesets.machine.openshift.io --namespace openshift-machine-api
```

After that you can scale one with the following command:
```
oc scale --replicas=5 machineset --namespace openshift-machine-api <machineset name>
```

You can monitor with the following command until all machines are ready and available:
```
oc get machinesets.machine.openshift.io --namespace openshift-machine-api
```

## Increase pod limit per node for worker nodes

By default, the Openshift node can run 250 pods. This limit is problematic for some workloads where many pods and deployments are created with little CPU/memory utilization. To increase the maximum number of pods per node to 500, you can use the configuration provided in the example: `utilities/examples/set-max-pods.yml`.

If you are in `tests/kube-burner/cluster-density` directory, you can run the following command:
```
oc create --filename=../../../utilities/examples/set-max-pods.yml
```

And after that, you can monitor the update status for the worker machine-pool with the following command:
```
oc get machineconfigpools worker
```

After `UPDATED` is back to `True` and the state for `UPDATING` is `False`, that means Openshift has finished updating all nodes in that machine-pool, and you can run your workload.

## Cleanup

Don't forget to delete artifacts after you destroy the cluster.

```
rm -rf "/tmp/artifacts-${INFRA_NAME}"
```
