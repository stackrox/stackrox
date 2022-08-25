# StackRox Operator

Central Services and Secured Cluster Services operator.

## Quickstart

All following commands should be run from this directory (`operator/`).

1. Build and run operator locally. Note that this starts the operator without deploying it as a container in the cluster.  
It does not install any webhooks either.
See [Advanced usage](#advanced-usage) for different ways of running operator.

```bash
make install run
```

2. Create `stackrox` image pull secret in `stackrox` namespace.  
Helm charts use it by default to configure pods. If it does not exist, you'll need to specify `ImagePullSecrets` in custom resources.

```bash
make stackrox-image-pull-secret
```

3. Create Central Custom Resource using ~~the provided sample~~ test sample.    

```bash
kubectl -n stackrox delete persistentvolumeclaims stackrox-db

# TODO: switch back to user-facing samples in `config/samples/platform_v1alpha1_*.yaml`.
kubectl apply -n stackrox -f tests/common/central-cr.yaml
```

4. Once Central services come online, create Secured Cluster using test sample.

```bash
# Get init-bundle secrets document from Central and save as secrets in the cluster
kubectl -n stackrox exec deploy/central -- \
  roxctl central init-bundles generate my-test-bundle --insecure-skip-tls-verify --password letmein --output-secrets - \
  | kubectl -n stackrox apply -f -

# Create Secured Cluster CR
kubectl apply -n stackrox -f tests/common/secured-cluster-cr.yaml
```

4. Check status of the custom resource.

```bash
# For Central
kubectl get -n stackrox centrals.platform.stackrox.io
kubectl get -n stackrox centrals.platform.stackrox.io stackrox-central-services --output=json

# For Secured Cluster
kubectl get -n stackrox securedclusters.platform.stackrox.io
kubectl get -n stackrox securedclusters.platform.stackrox.io stackrox-secured-cluster-services --output=json
```

5. Delete the custom resource.

```bash
# Central
kubectl delete centrals.platform.stackrox.io stackrox-central-services

# Secured Cluster
kubectl delete securedclusters.platform.stackrox.io stackrox-secured-cluster-services
```

## Automated testing

This runs unit and integration tests using a minimum k8s control plane (just apiserver and etcd).
Simply run:

```bash
$ make test
```

This runs end-to-end tests. Requires that your kubectl is configured to connect to a k8s cluster.
Simply run:

```bash
$ make test-e2e
```

### Secured Cluster Services

An example can be found in `config/samples/platform_v1alpha1_securedcluster.yaml`.

## List all available commands/targets

To see help on other available `Makefile` targets, run

```bash
$ make help
```

## Advanced usage

### Launch the operator on the (local) cluster

While `make install run` can launch the operator, the operator is running outside the cluster and this approach may not be sufficient to test all aspects of it.
An example are features that need to exercise any of the webhooks.

The recommended approach is the following.

0. Make sure you have [cert-manager installed](https://cert-manager.io/docs/installation/).
   It takes care of the TLS aspects of the connection from k8s API server to the webhook server
   embedded in the manager binary.

1. Build operator image
   ```bash
   $ make docker-build
   ```
2. Make the image available for the cluster, this depends on k8s distribution you use.  
   You don't need to do anything when using KIND.  
   For minikube it could be done like this
   ```bash
   $ docker save stackrox/stackrox-operator:$(make tag) | ssh -o StrictHostKeyChecking=no -i $(minikube ssh-key) docker@$(minikube ip) docker load
   ```
3. Install CRDs and deploy operator resources
   ```bash
   $ make deploy
   ```
4. Validate that the operator's pod has started successfully
   ```bash
   $ kubectl -n stackrox-operator-system describe pods
   ```
   Check logs
   ```bash
   $ kubectl -n stackrox-operator-system logs deploy/rhacs-operator-controller-manager manager -f
   ```
5. Create CRs and have fun testing.
   Make sure you delete the CRs before you undeploy the operator resources.
   Otherwise, you'll need to clean up the operands yourself.
6. When done
   ```bash
   $ make undeploy uninstall
   ```

### Bundling

```bash
# Refresh bundle metadata. Make sure to check the diff and commit it.
$ make bundle
# Make sure that the operator is built and pushed
$ make docker-build docker-push
# Build and push bundle image
$ make bundle-build docker-push-bundle
```

Build and push everything as **one-liner**

```bash
$ make everything
```

Testing bundle with Scorecard

```bash
# Test locally-built bundle files
$ make bundle-test
# Test bundle image; the image must be pushed beforehand.
$ make bundle-test-image
```

### Launch the operator on the cluster with OLM and the bundle

Note that unlike the `make deploy` route, deployment with OLM does not require cert-manager to be installed.

```bash
# 0. Get the operator-sdk program.
$ make operator-sdk

# 1. Install OLM.
$ make olm-install

# 2. Create a namespace for testing bundle.
$ kubectl create ns bundle-test

# 2. Create image pull secrets.
# If the inner magic does not work, just provide --docker-username and --docker-password with your DockerHub creds.
$ kubectl -n bundle-test create secret docker-registry my-opm-image-pull-secrets \
  --docker-server=https://quay.io/v2/ \
  --docker-email=ignored@email.com \
  $($(command -v docker-credential-osxkeychain || command -v docker-credential-secretservice) get <<<"quay.io" | jq -r '"--docker-username=\(.Username) --docker-password=\(.Secret)"')

# 3. Configure default service account to use these pull secrets.
$ kubectl -n bundle-test patch serviceaccount default -p '{"imagePullSecrets": [{"name": "my-opm-image-pull-secrets"}]}'

# 3. Build and push operator and bundle images.
# Use one-liner above.

# 4. Run bundle.
$ bin/operator-sdk-1.20.1 run bundle \
  quay.io/rhacs-eng/stackrox-operator-bundle:v$(make --quiet tag) \
  --pull-secret-name my-opm-image-pull-secrets \
  --service-account default \
  --namespace bundle-test

# 5. Add image pull secrets to operator's ServiceAccount.
# Run it while the previous command executes otherwise it will fail.
# Note that serviceaccount might not exist for a few moments.
# Rerun this command until it succeeds.
# We hope that in OpenShift world things will be different and we will not have to do this.
$ kubectl -n bundle-test patch serviceaccount rhacs-operator-controller-manager -p '{"imagePullSecrets": [{"name": "my-opm-image-pull-secrets"}]}'
# You may need to bounce operator pods after this if they can't pull images for a while.
$ kubectl -n bundle-test delete pod -l app=rhacs-operator

# 6. operator-sdk run bundle command should complete successfully.
# If it does not, watch pod statuses and check pod logs.
$ kubectl -n bundle-test get pods
# ... and dive deep from there into the ones that are not healthy.
```

Once done, cleanup with

```bash
kubectl -n bundle-test delete clusterserviceversions.operators.coreos.com -l operators.coreos.com/rhacs-operator.bundle-test

kubectl -n bundle-test delete subscriptions.operators.coreos.com -l operators.coreos.com/rhacs-operator.bundle-test

kubectl -n bundle-test delete catalogsources.operators.coreos.com rhacs-operator-catalog
```

Also, you can blow everything away with

```bash
$ make olm-uninstall
$ kubectl delete ns bundle-test
```

### Launch the Operator with OLM and Index

Note this assumes OLM is already in place which is the case for OpenShift clusters.  
If you're launching on non-OpenShift Kubernetes, first deploy OLM with `make olm-install`.

```bash
# Deploy
# TODO(ROX-11744): drop branding here once operator is available from quay.io/stackrox-io
ROX_PRODUCT_BRANDING=RHACS_BRANDING make deploy-via-olm TEST_NAMESPACE=index-test

# Undeploy
kubectl delete ns index-test
```

## Extending the StackRox Custom Resource Definitions

Instructions and best practices on how to extend the StackRox CRDs is contained in the separate file
[EXTENDING_CRDS.md](./EXTENDING_CRDS.md).

## Installing operator via OLM

The following command will install operator to the currently selected kubernetes cluster.

```bash
 make kuttl deploy-via-olm
```

If operator image has a `-dirty` suffix then the following command has to be used instead:

```bash
make kuttl deploy-dirty-tag-via-olm
```

For upgrading an existing operator:

```bash
make kuttl upgrade-via-olm

```
Note ерфе there is a specific command for upgrading `-dirty` suffixed tags `upgrade-dirty-tag-via-olm`
