## Clair Scanner Testing

The files in this directory deploy a Clair scanner for test purposes. This is
intended to work with Clair v2.1.4 and vulns will be found for nginx:1.12.1.

To deploy the scanner you need to set `CLAIR_DB_PASSWORD`
and run the deploy script. e.g.:

```
CLAIR_DB_PASSWORD="something" ./scripts/ci/clair/deploy.sh qa-clair
```

To test you need to set `CLAIR_ENDPOINT` e.g.:

```
CLAIR_ENDPOINT="http://clairsvc.qa-clair:6060" ./gradlew test --tests=ImageScanningTest
```

### The Test Database

Data feeds take some time to load after Clair startup, so this setup uses a
prepopulated database. To reduce the size of the database and make it reasonable
for CI, all but the relevant vulns, features and namespaces were deleted prior
to dump.

Image layers were added using a modified `klar` that POSTs using sha256 image
layer names: https://github.com/gavin-stackrox/klar. As so:

```
kubectl -n qa-clair port-forward svc/clairsvc 6060:6060 &
CLAIR_ADDR=localhost:6060 CLAIR_OUTPUT=High klar nginx:1.12.1
```
