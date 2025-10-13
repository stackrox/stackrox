# Debug OpenShift CI jobs

## Download all artifacts

1. Visit the openshift-ci jobs page, visible in the github build details or the corresponding JIRA ticket.
2. Click on the `Artifacts` link in the top right corner.
3. Navigate to `artifacts/` > `<job name>` `stackrox-stackrox-e2e-test/artifacts/howto-locate-other-artifacts-summary.html` and open the HTML file.
4. Copy the `gsutil` command and run it locally to copy the job artifacts. Alternatively you can browse them in GCP.

### Working with the artifacts

```
# Import Central backup into existing Central installation
$ cd <build-artifacts>/central-data
$ roxctl central db restore postgres_db_2024_05_06_20_15_42.sql.zip

# Import database dump via postgres tooling
$ unzip postgres_db_2024_05_06_20_15_42.sql.zip
$ psql -h $DB_HOST_IP -U $DB_USER -p $DB_PORT -c 'create database central_dump;'
$ pg_restore -h $DB_HOST_IP -p $DB_PORT -U $DB_USER -d central_dump --no-owner --clean --if-exists --exit-on-error -Fc -vvv --single-transaction --schema=public postgres.dump
```
