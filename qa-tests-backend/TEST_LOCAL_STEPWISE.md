Overview
--------

    RUN GROOVY TESTS LOCALLY ON MACOS (INTEL) AGAINST A REMOTE CLUSTER.

    1. Setup local dev environment for Groovy test invocation
    2. Install ACS on test cluster
    3. Setup test fixtures (nothing needed yet... revisit if needed for other tests investigation)
    4. Run a single Groovy test

Plan for running //rox/qa-tests-backend tests locally against a remote cluster:

1. Build a test runtime image (maybe use rox-ci-image)
  - java, gradle, groovy, ...                          <- dockerfile
  - env vars for quay.io                               <- pass
  - docker build time working directory                -> /build
  - rw bind mount of ~/data/run-qa-tests/              -> /data
  - rw bind mount of ~/go/src/github.com/stackrox/rox/ -> /rox
2. Bringup a test cluster (Infra or run automation-flavor image locally)
3. Use the test image to build test prerequisites and setup test harness
4. Use the test image to run //row/qa-tests-backend


qa-test-settings.properties
---------------------------

This file format is described in the `load` method of the
[java.util.properties][java_util_properties] documentation.

Example encoding GCP service account credentials:
```
jq -rc '.' < credentials.json | perl -pn -e 's/\\n/\\/g; chomp if eof; print;'
```

[java_util_properties]: https://docs.oracle.com/javase/9/docs/api/java/util/Properties.html

Setup Service Account
---------------------

Only needed once to enable CI runs. Adding this info since it was useful in troubleshoting
dotenv format issues for [RS-361](https://issues.redhat.com/browse/RS-361).

```
PROJECT_ID="stackrox-ci"
SERVICE_ACCOUNT_NAME="shane-rs361"
SERVICE_ACCOUNT_ID="${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Create the service account
gcloud --project "$PROJECT_ID" iam service-accounts create "$SERVICE_ACCOUNT_NAME"
gcloud --project "$PROJECT_ID" iam service-accounts keys create \
    "$HOME/creds/$SERVICE_ACCOUNT_NAME.json" --iam-account="$SERVICE_ACCOUNT_ID"

# Add role bindings
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$SERVICE_ACCOUNT_ID" --role="roles/containeranalysis.admin"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="serviceAccount:$SERVICE_ACCOUNT_ID" --role="roles/storage.objectViewer"

# List roles bound to this service account
gcloud projects get-iam-policy $PROJECT_ID  \
    --flatten="bindings[].members" \
    --format="table(bindings.role)" \
    --filter="bindings.members:$SERVICE_ACCOUNT_ID"

# List roles bound to existing `ci-gcr-scanner` service account
gcloud projects get-iam-policy $PROJECT_ID  \
    --flatten="bindings[].members" \
    --format="table(bindings.role)" \
    --filter="bindings.members:ci-gcr-scanner@stackrox-ci.iam.gserviceaccount.com"

# Delete the example service account
yes | gcloud --project "$PROJECT_ID" iam service-accounts delete "$SERVICE_ACCOUNT_ID"
```


References
----------

* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/842006562/Release+Checklists+-+QA+Signoff
* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/1558642983/QA+Release+Checklist+-+3.0.50.0
* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/1340015510/Upgrade+test
* https://help-internal.stackrox.com/docs/get-started/quick-start/
* https://cloud.google.com/docs/authentication
* https://cloud.google.com/sdk/gcloud/reference/auth/activate-service-account
* https://cloud.google.com/docs/authentication/production
