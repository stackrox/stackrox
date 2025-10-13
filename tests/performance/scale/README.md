# Tooling for working with RH Performance &amp; Scale systems

The [RH Openshift Performance & Scale team](https://source.redhat.com/communitiesatredhat/communitiesofpractice/crosscuttingco/product-performance-scale-community-of-practice/performance__scale_community_of_practice_wiki/openshift_performance_and_scale_knowledge_base) provide a number of tools and
frameworks to regularly test the performance of OpenShift. This repo provides
some doumentation and tooling for using these with ACS.

## Background

- [airflow](https://github.com/cloud-bulldozer/airflow-kubernetes)
- [e2e-benchmarking](https://github.com/cloud-bulldozer/e2e-benchmarking)
- [benchmark-operator](https://github.com/cloud-bulldozer/benchmark-operator)
- [kube-burner](https://github.com/cloud-bulldozer/kube-burner)
- [scale-ci-deploy](https://github.com/cloud-bulldozer/scale-ci-deploy)

## ACS running in perfScale airflow

- [airflow](http://airflow.apps.sailplane.perf.lab.eng.rdu2.redhat.com/home) (see DAGs with `acs` in their title).
- Metrics from [kube-burner](https://github.com/cloud-bulldozer/e2e-benchmarking/blob/dc5b31b5119605579ccbd1c6eaf6bf4a2f81dc2c/workloads/kube-burner/metrics-profiles/acs-metrics.yaml#L208) runs are indexed in Elastic Search http://perf-results-elastic.apps.observability.perfscale.devcluster.openshift.com / ripsaw-kube-burner
- Some sample visualisations are in the ACS folder in http://marquez.perf.lab.eng.rdu2.redhat.com:3000/dashboards/f/EMJr7Spnz/acs

## ACS deployment integrated with OpenShift monitoring

Using helm installs it is possible to have openshift-monitoring scrape ACS
metrics endpoints:

If you are running for the first time, add the helm repo:

```
helm repo add rhacs https://mirror.openshift.com/pub/rhacs/charts
helm repo update
```

To start central and sensor execute the following:

```
helm install -n stackrox stackrox-central-services --create-namespace rhacs/central-services \
  --set central.exposure.route.enabled=true \
  --set central.adminPassword.value=<a password> \
  --set central.image.tag=<probably a recent dev tag> \
  --set imagePullSecrets.username="something with pull rights from docker.io" \
  --set imagePullSecrets.password="something with pull rights from docker.io" \
  --set enableOpenShiftMonitoring=true \
  --set central.exposeMonitoring=true
```

When central is running grab a bundle:

```
roxctl -e https://`oc -n stackrox get routes central -o json | jq -r '.spec.host'`:443 \
   -p <the central password> central init-bundles generate perf-test \
   --output perf-bundle.yml
```

Then install the secured cluster in a similar manner to central.

```
helm install -n stackrox stackrox-secured-cluster-services rhacs/secured-cluster-services \
  -f perf-bundle.yml \
  --set image.main.tag=<probably a recent dev tag> \
  --set imagePullSecrets.username="something with pull rights from docker.io" \
  --set imagePullSecrets.password="something with pull rights from docker.io" \
   --set clusterName=perf-test \
   --set enableOpenShiftMonitoring=true \
   --set exposeMonitoring=true
```

Let openshift-monitoring know to include ACS metrics:

```
oc label namespace/stackrox openshift.io/cluster-monitoring="true"
```

## Measurements

### CPU and MEM via Prometheus

At present I'm interested in what sensor and central are up to regarding CPU and memory usage. From the openshift console -> monitoring -> metrics -> open prometheus UI. Then for memory:
```
node_namespace_pod_container:container_memory_working_set_bytes{container=~"sensor|central"}
```
Add a panel for CPU:
```
node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{container=~"sensor|central"}
```

Or use this URL with the appropriate prometheus route (`oc get routes -A | grep prometheus-k8s`):
```
https://__prometheus__/graph?g0.expr=node_namespace_pod_container%3Acontainer_memory_working_set_bytes%7Bcontainer%3D~%22sensor%7Ccentral%22%7D&g0.tab=0&g0.stacked=0&g0.range_input=1h&g1.expr=node_namespace_pod_container%3Acontainer_cpu_usage_seconds_total%3Asum_rate%7Bcontainer%3D~%22sensor%7Ccentral%22%7D&g1.tab=0&g1.stacked=0&g1.range_input=1h
```

### Top

As a comparison, running top on the node for the component of interest is a good idea:tm:.

```
# find sensor node:
$ oc -n stackrox get pods -o wide
...
# sensor + node should be in ^^

# get a shell on the node
$ oc adm debug node/_sensor_node_

# find sensor process e.g.:
$ ps axw | grep sensor
121903 ?        Ssl    6:35 /stackrox/bin/kubernetes-sensor

# run a batch top just looking at sensor:
$ top -w 132 -b -p 121903
```
