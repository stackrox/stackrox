# StackRox Operator

Central Services and Secured Cluster Services operator.

## Requirements

 - operator-sdk 1.5.x

## Quickstart

All following commands should be ran from this directory (`operator/`).

1. Build and run operator locally. Note that this starts the operator without deploying it as a container in the cluster.

```bash
$ make install run
```

2. Create Custom Resource using the provided sample.

```bash
$ kubectl apply -f config/samples/platform_v1alpha1_*.yaml
```

3. Check status of the custom resource.

```bash
$ kubectl get -n stackrox centrals.platform.stackrox.io
$ kubectl get -n stackrox centrals.platform.stackrox.io stackrox-central-services --output=json
```

or

```bash
$ kubectl get -n stackrox securedclusters.platform.stackrox.io
$ kubectl get -n stackrox securedclusters.platform.stackrox.io stackrox-secured-cluster-services --output=json
```

4. Delete the custom resource.

```bash
$ kubectl delete centrals.platform.stackrox.io stackrox-central-services
```

or

```bash
$ kubectl delete securedclusters.platform.stackrox.io stackrox-secured-cluster-services
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

While `make install run` can launch the operator, the operator is running outside of the cluster and this approach may not be sufficient to test all aspects of it.

The recommended approach is the following.

1. Build operator image
   ```bash
   $ make docker-build
   ```
2. Make the image available for the cluster, this depends on k8s distribution you use.  
   You don't need to do anything when using KIND.  
   For minikube it could be done like this
   ```bash
   $ docker save stackrox.io/stackrox-operator:$(make tag) | ssh -o StrictHostKeyChecking=no -i $(minikube ssh-key) docker@$(minikube ip) docker load
   ```
3. Install CRDs and deploy operator resources
   ```bash
   $ make install deploy
   ```
4. Validate that the operator's pod has started successfully
   ```bash
   $ kubectl -n stackrox-operator-system describe pods
   ```
   Check logs
   ```bash
   $ kubectl -n stackrox-operator-system logs deploy/stackrox-operator-controller-manager manager -f
   ```
5. Create CRs and have fun testing.
6. When done
   ```bash
   $ make undeploy
   ```

### Bundling

```bash
# Refresh bundle metadata. Make sure to check the diff and commit it.
$ make bundle
# Make sure that the operator is built and pushed
$ make docker-build docker-push
# Build and push bundle image
$ make bundle-build docker-push-bundle
# Run scorecard tests for the bundle
$ make bundle-test
```

Build and push as one-liner

```bash
$ make bundle docker-build docker-push bundle-build docker-push-bundle
```

### Launch the operator on the cluster with OLM and the bundle

TODO
