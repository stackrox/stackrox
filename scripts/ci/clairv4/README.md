## Clair v4 Scanner Testing

The files in this directory deploy a Clair v4 scanner for test purposes. This is
intended to work with Clair v4.5.1.

To deploy the scanner, you need to run the deploy script. e.g.:

```
./scripts/ci/clairv4/deploy.sh qa-clairv4
```

To test you need to set `CLAIR_V4_ENDPOINT` e.g.:

```
CLAIR_V4_ENDPOINT="http://clairv4.qa-clairv4:8080" ./gradlew test --tests=ImageScanningTest
```

To tear it down, run:

```
./scripts/ci/clairv4/teardown.sh qa-clairv4
```

Clair runs in offline/air-gapped mode, meaning it will not update its
vulnerability database. The PostgreSQL database is pre-populated with
RHEL vulnerabilities obtained from a past run of Clair.

It was done as follows:

1. Run Clair in online-mode.
1. Wait an arbitrarily long time for vulns to be populated
1. Stop the Clair pod, so no more updates occur
1. Exec into the PostgreSQL deployment and run the following:
   `pg_dump -U postgres -d clair > /tmp/dump.sql`
1. Run `kubectl -n qa-clairv4 exec <pod-name> -- tar cf - /tmp/dump.sql | tar xf - -C $(pwd)/dump`
1. Run `gzip --best dump.sql`
1. Create the image using the Dockerfile in this directory.
