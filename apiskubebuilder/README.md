# Generate Kubernetes APIs with kubebuilder


## Initialize a project

```
$ kubebuilder init --domain central.stackrox.io --repo github.com/stackrox/stackrox/apis
$ kubebuilder create api --group authprovider --version v1 --kind AuthProvider --resource --controller
```

## Build CRD to be installed with Helm

```
$ mkdir -p $(git rev-parse --show-toplevel)/image/templates/helm/stackrox-central/crds
$ kubectl kustomize config/crd > $(git rev-parse --show-toplevel)/image/templates/helm/stackrox-central/crds/authproviderv1beta1_crd.yaml

# Generate helm chart
$ make cli
$ roxctl helm output central-services --debug --remove

# Install helm chart
$ helm upgrade --install stackrox-central-services ./stackrox-central-services-chart \
    --set imagePullSecrets.username=$QUAY_REGISTRY_USER \
     --set imagePullSecrets.password=$QUAY_REGISTRY_PASSWORD \
     --set central.image.tag=latest \
    -n stackrox \
    --create-namespace
```
