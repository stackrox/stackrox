# Community StackRox Operator installation

## Introduction

Historically, Helm and the "manifest installation" methods were the only way to install the community, StackRox-branded build.
An operator was available only for the "Red Hat Advanced Cluster Security"-branded build.

This is changing. Due to significant maintenance burden of three installation methods,
we are planning to consolidate on just one: the operator.

As the first step, in the 4.10 release we proved the simplest possible, _temporary_ way to install the community StackRox-branded operator.
We hope this is useful to the community for getting to know the operator.

**See [the document in the `release-4.10` branch for instructions](https://github.com/stackrox/stackrox/blob/release-4.10/operator/install/README.md) for the above.**

---

**The following text describes the installation for the upcoming 4.11 release.**

In release 4.11, we plan to provide a more customizable, powerful and unified way to install the operator.

## How to use it?

Once 4.11 is released, installing the operator is simply a matter of:
```shell
helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/

helm install --wait --namespace stackrox-operator-system --create-namespace stackrox-operator stackrox/stackrox-operator
```

## Where to go from here?

Once the operator is running, to actually deploy StackRox you need to create a `Central` and/or a `SecuredCluster` custom resource.
Please have a look at the [samples](../config/samples) directory.

Before applying the `SecuredCluster` CR you need to retrieve from central and apply on the cluster a cluster registration secret.

[Documentation for the custom resource schema](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/latest/html/installing/installing-rhacs-on-red-hat-openshift#install-central-config-options-ocp) -
the way to customize your StackRox deployment - is currently only available
at the Red Hat documentation portal.

## Caveats

You may encounter a few references to RH ACS when using the operator in places such as:
- the descriptions of a few fields in the OpenAPI schema of the custom resources
- the `UserAgent` header used by the operator controller when talking to the kube API server
- central web UI when generating cluster registration secrets

These will be cleaned up in a future release.
