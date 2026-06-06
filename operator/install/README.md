# Community StackRox Operator installation

## Introduction

Historically, Helm and the "manifest installation" methods were the only way to install the community, StackRox-branded build.
An operator was available only for the "Red Hat Advanced Cluster Security"-branded build.

This is changing. Due to significant maintenance burden of three installation methods,
we are planning to consolidate on just one: the operator.

**The following text describes the installation for release 4.11 and later.**

## How to use it?

Once 4.11 is released, installing the operator is simply a matter of:
```shell
helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/

helm install --wait --namespace stackrox-operator-system --create-namespace stackrox-operator stackrox/stackrox-operator
```

> [!WARNING]
> If you are upgrading from a 4.10.x operator manifest-based installation, include `--take-ownership` in the `helm` command line.
> You'll want at least helm 3.18 (released May 19, 2025) for this to work correctly with CRDs.

## Where to go from here?

Once the operator is running, to actually deploy StackRox you need to create a `Central` and/or a `SecuredCluster` custom resource.

### Migrations

For help replacing an existing deployment (done with manifests or the `central-services`/`secured-cluster-services` charts)
you can use the new `roxctl central migrate-to-operator` and `roxctl sensor migrate-to-operator` commands.
These generate custom resources, which - if applied in the same namespace as an existing legacy installation -
will cause the operator to seamlessly take over the existing resources.

> [!WARNING]
> The `migrate-to-operator` commands are not currently aware of every way that a legacy StackRox deployment
> could have been customized. We plan to improve on this in the following release, but nevertheless we advise
> caution.
> 
> For example for central, you can use a command such as `helm get values -n stackrox stackrox-central-services` to find what Helm values
> were used when installing, and then check the [documentation for the custom resource schema](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/latest/html/installing/installing-rhacs-on-red-hat-openshift#install-central-config-options-ocp)
> to find out which CR fields need to be set to achieve the same effect.

Example:

```shell
$ roxctl central migrate-to-operator --namespace stackrox > cr-central.yaml
$ cat cr-central.yaml
$ kubectl apply --namespace stackrox -f cr-central.yaml
```

### New installations

Please have a look at the [samples in this directory](.).

First, create a namespace and apply a `Central` CR.

```shell
kubectl create namespace stackrox
kubectl apply -f https://raw.githubusercontent.com/stackrox/stackrox/refs/heads/master/operator/install/platform_v1alpha1_central.yaml
```

Then, once central is up, you need to retrieve from central and apply on the cluster a Cluster Registration Secret (CRS).

```shell
kubectl -n stackrox rollout status deployment central
kubectl -n stackrox get secrets central-htpasswd --template='{{.data.password | base64decode}}' | \
kubectl -n stackrox exec -i deploy/central -- bash -c \
  'ROX_ADMIN_PASSWORD=$(cat) roxctl --insecure-skip-tls-verify central crs generate crs1 -o -' |
kubectl -n stackrox apply -f -
```

Finally, apply a `SecuredCluster` CR. You may want to adjust `spec.clusterName` to your preference.
Also, you'll need to set `spec.centralEndpoint` when applying `SecuredCluster` on a different cluster than `Central`.

```shell
kubectl apply -f https://raw.githubusercontent.com/stackrox/stackrox/refs/heads/master/operator/install/platform_v1alpha1_securedcluster.yaml
```

[Documentation for the custom resource schema](https://docs.redhat.com/en/documentation/red_hat_advanced_cluster_security_for_kubernetes/latest/html/installing/installing-rhacs-on-red-hat-openshift#install-central-config-options-ocp) -
the way to customize your StackRox deployment - is currently only available
at the Red Hat documentation portal.

## Caveats

You may encounter a few references to RH ACS when using the operator in places such as:
- the descriptions of a few fields in the OpenAPI schema of the custom resources
- the `UserAgent` header used by the operator controller when talking to the kube API server
- central web UI when generating cluster registration secrets

These will be cleaned up in a future release.

## Note about release 4.10

As the first step, in the 4.10 release we proved the simplest possible, _temporary_ way to install the community StackRox-branded operator.
See [this document in the `release-4.10` branch for instructions](https://github.com/stackrox/stackrox/blob/release-4.10/operator/install/README.md) for that release.
