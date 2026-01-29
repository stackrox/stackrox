# Community StackRox Operator installation

## Introduction

Historically, Helm and the "manifest installation" methods were the only way to install the community, StackRox-branded build.
An operator was available only for the "Red Hat Advanced Cluster Security"-branded build.

This is changing. Due to significant maintenance burden of three installation methods,
we are planning to consolidate on just one: the operator.

As the first step, in the 4.10 release we are providing the simplest possible, _temporary_ way to install the community StackRox-branded operator.
We hope this is useful to the community for getting to know the operator before we provide a more customizable, powerful and unified way to install it in a subsequent release.

## How to use it?

Once 4.10 is released, installing the operator is simply a matter of:
```shell
kubectl apply -f https://github.com/stackrox/stackrox/raw/refs/tags/4.10.0/operator/install/manifest.yaml
kubectl rollout status deployment -n stackrox-operator-system stackrox-operator-controller-manager
```

## Where to go from here?

Once the operator is running, to actually deploy StackRox you need to create a `Central` and/or a `SecuredCluster` custom resource.
Please have a look at the [samples](../config/samples) directory.

Before applying the `SecuredCluster` CR you need to retrieve from central and apply on the cluster an init bundle or cluster registration secret.
**Note** that currently the page where you can generate an init bundle requires you to select OpenShift as the platform
Otherwise it is only possible to download an init bundle formatted for Helm installations.

[Documentation for the custom resource schema](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/latest/html/installing/installing-rhacs-on-red-hat-openshift#install-central-config-options-ocp) -
the way to customize your StackRox deployment - is currently only available
at the Red Hat documentation portal.

## Caveats

You may encounter a few references to RH ACS when using the operator in places such as:
- the descriptions of a few fields in the OpenAPI schema of the custom resources
- the `UserAgent` header used by the operator controller when talking to the kube API server
- central web UI when generating init bundles or cluster registration secrets

These will be cleaned up in a future release.

## How was this manifest created?

```shell
BUILD_TAG=4.10.0 ROX_PRODUCT_BRANDING=STACKROX_BRANDING make -C operator/ build-installer
cp operator/dist/install.yaml operator/install/manifest.yaml
```
