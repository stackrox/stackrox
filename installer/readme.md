# Go-based Stackrox Installer

This is the beginning of a go-based replacement for our Helm charts.
Currently just a PoC, but it could potentially become what is used by the operator to create k8s resources.

If you're interseted in kicking the tires on the PoC:

* Check out the `klape/unified-build` branch.
* `make bin/installer` (run this from the root dir of the repo)
* Create an `installer.yaml` with contents something like:

```
namespace: stackrox
scannerV4: true
images:
  scannerDb: "localhost:5001/stackrox/scanner-db:latest"
```

* run `bin/installer export {central,crs,securedcluster}` to print the resources to stdout or `bin/installer apply {central,crs,securedcluster}` to have the CLI apply them to the active k8s cluster
* The `crs` command will port-forward to central to create a CRS and then apply it to the same namespace in prep for secured cluster to be applied. 
