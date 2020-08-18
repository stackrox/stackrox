## Anchore Scanner Testing

Ref: https://docs.anchore.com/current/docs/engine/engine_installation/helm/

The scripts in this directory add support to deploy an Anchore scanner for test purposes.
As relied on by e.g. qa-tests-backend/src/test/groovy/ImageScanningTest.groovy.

To deploy the scanner you need to set `ANCHORE_USERNAME` and `ANCHORE_PASSWORD`
and run the deploy script. e.g.:

```
ANCHORE_USERNAME=admin ANCHORE_PASSWORD="something" ./scripts/ci/anchore/deploy.sh qa-anchore qa
```

To test you need to set `ANCHORE_USERNAME`, `ANCHORE_PASSWORD` and `ANCHORE_ENDPOINT` e.g.:

```
ANCHORE_USERNAME=admin ANCHORE_PASSWORD="something" \
ANCHORE_ENDPOINT="http://qa-anchore-engine-api.qa-anchore:8228" \
gradle test --tests=ImageScanningTest
```

### The Test Database

The deployment uses a limited set of vulnerabilities for debian:8 and Anchore
scanner 1.6.9. The full data feed for Anchore can take multiple hours to
download making it unfeasible to rely on in CI. Similarily re-creating a DB with
the full feed takes too long for CI. The limited data set was gathered by
installing Anchore and waiting for the feeds to accumulate and filtering out all
but debian:8 (see `feed_setup.sh`).
