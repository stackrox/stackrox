# StackRox Operators

* Central Services Operator is in `central/`
* Secured Cluster Services Operator is in  `securedcluster/`

## Quickstart

Run the following commands while being in a directory of the operator (e.g. `central/`).

1. Build and run operator locally. Note that this starts the operator without deploying it as a container in the cluster.

```bash
$ make install run
```

2. Create Custom Resource using the provided sample.

```bash
$ kubectl apply -f config/samples/platform_v1alpha1_central.yaml
```

3. Check status of the custom resource.

```bash
$ kubectl get centrals.platform.stackrox.io
$ kubectl get centrals.platform.stackrox.io central-sample --output=json
```

4. Delete the custom resource.

```bash
$ kubectl delete centrals.platform.stackrox.io central-sample
```

To see help on other `Makefile` targets, run

```bash
$ make help
```
