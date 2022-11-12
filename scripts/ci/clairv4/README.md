## Clair v4 Scanner Testing

The files in this directory deploy a Clair v4 scanner for test purposes. This is
intended to work with Clair v4.4.4 and vulns will be found for nginx:1.12.1.

To deploy the scanner you need to set `CLAIR_V4_DB_PASSWORD`
and run the deploy script. e.g.:

```
CLAIR_V4_DB_PASSWORD="something" ./scripts/ci/clairv4/deploy.sh qa-clairv4
```

To test you need to set `CLAIR_V4_ENDPOINT` e.g.:

```
CLAIR_V4_ENDPOINT="http://clairv4.qa-clairv4:8080" ./gradlew test --tests=ImageScanningTest
```

### The Test Database

Data feeds take some time to load after Clair v4 startup, so this setup uses a
prepopulated database. To reduce the size of the database and make it reasonable
for CI, all but the relevant vulns, features and namespaces were deleted prior
to dump.
