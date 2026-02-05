#!/usr/bin/env bash
set -eou pipefail

# First get examples of machine sets from worker machine sets. It is important that we have different zones and to have correct configuration for disk, etc.
oc get machineset.machine.openshift.io --namespace openshift-machine-api $(oc get machineset.machine.openshift.io --namespace openshift-machine-api | grep 'worker' | sort | tail -n 1 | cut -f 1 -d ' ') -o yaml > /tmp/prom-perf-machineset-1.yaml
oc get machineset.machine.openshift.io --namespace openshift-machine-api $(oc get machineset.machine.openshift.io --namespace openshift-machine-api | grep 'worker' | sort -r | tail -n 1 | cut -f 1 -d ' ') -o yaml > /tmp/prom-perf-machineset-2.yaml

# Cleanup config and update some params. Important is replicas=1, to remove annotations, and to set "perf-scale-node-role" for spec.metadata.labels (you need yq for that)
cat /tmp/prom-perf-machineset-1.yaml \
    | yq 'del(.metadata|.uid)' \
    | yq 'del(.status)' \
    | yq 'del(.metadata.annotations)' \
    | yq 'del(.metadata.creationTimestamp)' \
    | yq 'del(.metadata.resourceVersion)' \
    | yq '.spec.template.spec.providerSpec.value.machineType = "n2-standard-32"' \
    | yq '.spec.replicas = 1' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machine-role" = "infra"' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machine-type" = "infra"' \
    | yq '.spec.template.spec.metadata.labels."perf-scale-node-role" = "prometheus"' \
    | yq '.metadata.name += "-prom-1"' \
    | yq '.spec.selector.matchLabels."machine.openshift.io/cluster-api-machineset" += "-prom-1"' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machineset" += "-prom-1"' \
> /tmp/prom-perf-machineset-1-updated.yaml

cat /tmp/prom-perf-machineset-2.yaml \
    | yq 'del(.metadata|.uid)' \
    | yq 'del(.status)' \
    | yq 'del(.metadata.annotations)' \
    | yq 'del(.metadata.creationTimestamp)' \
    | yq 'del(.metadata.resourceVersion)' \
    | yq '.spec.template.spec.providerSpec.value.machineType = "n2-standard-32"' \
    | yq '.spec.replicas = 1' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machine-role" = "infra"' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machine-type" = "infra"' \
    | yq '.spec.template.spec.metadata.labels."perf-scale-node-role" = "prometheus"' \
    | yq '.metadata.name += "-prom-2"' \
    | yq '.spec.selector.matchLabels."machine.openshift.io/cluster-api-machineset" += "-prom-2"' \
    | yq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machineset" += "-prom-2"' \
> /tmp/prom-perf-machineset-2-updated.yaml

# Apply config to cluster (after this we should see 2 additional machinesets)
oc create --filename=/tmp/prom-perf-machineset-1-updated.yaml
oc create --filename=/tmp/prom-perf-machineset-2-updated.yaml

# !!! WAIT for nodes to be ready - it should be 2 additional nodes !!!
oc get machineset.machine.openshift.io --namespace openshift-machine-api

# After nodes are up and ready -> we want to move prometheus pods to dedicated nodes for it
oc apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    prometheusK8s:
      nodeSelector:
        perf-scale-node-role: "prometheus"
EOF
