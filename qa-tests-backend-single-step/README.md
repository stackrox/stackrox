Overview
--------

    RUN GROOVY TESTS LOCALLY ON MACOS (INTEL) AGAINST A REMOTE CLUSTER.

    1. Setup local dev environment for Groovy test invocation
    2. Install ACS on test cluster
    3. Setup test fixtures
    4. Run a single Groovy test

Plan for running //rox/qa-tests-backend tests locally against a remote cluster:

1. Build a test runtime image (use rox-ci-image)
  - java, gradle, groovy, ...                          <- dockerfile
  - env vars for quay.io                               <- pass
  - docker build time working directory                -> /build
  - rw bind mount of ~/data/run-qa-tests/              -> /data
  - rw bind mount of ~/go/src/github.com/stackrox/rox/ -> /rox
2. Bringup a test cluster (Infra or run automation-flavor image locally)
3. Use the test image to build test prerequisites and setup test harness
4. Use the test image to run //row/qa-tests-backend

2021-11-18 Thursday
-------------------

https://app.circleci.com/pipelines/github/stackrox/rox/64605/workflows/3014576f-6092-48a4-ae32-14ba372d7195

### ROSA

https://console.redhat.com/acs-1637278575

### OSD-on-AWS

https://qaprodauth.console.redhat.com/acs-1637278573

### OSD-on-GCP

https://qaprodauth.console.redhat.com/acs-1637278584


References
----------

* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/842006562/Release+Checklists+-+QA+Signoff
* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/1558642983/QA+Release+Checklist+-+3.0.50.0
* https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/1340015510/Upgrade+test
* https://help-internal.stackrox.com/docs/get-started/quick-start/
