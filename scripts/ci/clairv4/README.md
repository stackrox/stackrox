## Clair v4 Scanner Testing

The files in this directory deploy a Clair v4 scanner for test purposes. This is
intended to work with Clair v4.5.0 and vulns will be found for nginx:1.12.1.

To deploy the scanner you need to set `CLAIR_V4_DB_PASSWORD`
and run the deploy script. e.g.:

```
CLAIR_V4_DB_PASSWORD="something" ./scripts/ci/clairv4/deploy.sh qa-clairv4
```

To test you need to set `CLAIR_V4_ENDPOINT` e.g.:

```
CLAIR_V4_ENDPOINT="http://clairv4svc.qa-clairv4:6060" ./gradlew test --tests=ImageScanningTest
```

### The Test Database

Data feeds take some time to load after Clair v4 startup, so this setup uses a
prepopulated database. To reduce the size of the database and make it reasonable
for CI, all but the relevant vulns, features and namespaces were deleted prior
to dump.

Image layers were added using a modified `klar` that POSTs using sha256 image
layer names: https://github.com/gavin-stackrox/klar. As so:

###TODO

```
kubectl -n qa-clair port-forward svc/clairv4svc 6060:6060 &
CLAIR_ADDR=localhost:6060 CLAIR_OUTPUT=High klar nginx:1.12.1
```
