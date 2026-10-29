# StackRox Platform Integration Tests

This framework is designed to test integration and functional flows through APIs on a running k8s or openshift cluster.

# Prerequisites
Gradle is used to build and run tests written in Groovy using the Spock test framework.

- If you would like to use the recommended IDE:
  - Download and install [IntelliJ IDEA](https://www.jetbrains.com/idea/download/)
    and create a new project from the `qa-tests-backend` directory.

# Running Tests
- If protos have been changed or not generated: `make proto-generated-srcs`
- If you plan to run tests pulling from quay.io (currently every test derived
  from `BaseSpecification`), set `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` env
  vars. Read-only credentials are available in bitwarden's "ACS general engineering secrets"
  collection under `Quay.io readonly user`.

## Test access to the StackRox instance under test
After deployment, source the roxie-generated envrc file to set `ROX_ADMIN_PASSWORD` and
`API_ENDPOINT`. The QA client uses `API_HOSTNAME` and `API_PORT`, so set them to the host
and port from `API_ENDPOINT` before running tests. Roxie's interactive sub-shell requires
the same variables when the endpoint is not `localhost:8000`.
For OpenShift clusters, also export `CLUSTER=OPENSHIFT`; Roxie's environment does not set `CLUSTER`.

If you need to override connection settings (e.g. when testing against an existing cluster),
the following environment variables are available:
- `API_HOSTNAME`: defaults to `localhost`
- `API_PORT`: defaults to `8000`
- `CLUSTER`: either `OPENSHIFT` or `K8S`
- `ROX_ADMIN_PASSWORD`: admin password for the Central instance

## Using IntelliJ
If you have deployed StackRox into a cluster with roxie and roxies post-deployment environment
has been loaded, the tests in `src/test/groovy/` can be run directly.

### Tests dependent on integration credentials
- If your tests depend on an integration password or token in an environment variable such as:
  `GOOGLE_CREDENTIALS_GCR_SCANNER_V2`, `EMAIL_NOTIFIER_PASSWORD`,
  `MAILGUN_PASSWORD`, `JIRA_TOKEN`, `DTR_REGISTRY_PASSWORD`, `QUAY_PASSWORD`
  - Create a `qa-tests-backend/qa-test-settings.properties` file that contains environment variable assignments.
  - Copy environment variable settings from [BitWarden](https://vault.bitwarden.com/#/vault?itemId=da41ea10-15fa-44e2-988e-af260101b26e)

### Custom configuration
- If you have deployed the cluster differently or need to use a custom environment variable configuration:
- Go to `Run > Edit Configurations`
- Select Gradle, add a new configuration
  - Script path : `github.com/stackrox/stackrox/qa-tests-backend/src/test/groovy/<Groovy class name>.groovy`
  - Working Directory : `github.com/stackrox/stackrox/qa-tests-backend`
  - Environment Variables:
    - `CLUSTER`: Either `OPENSHIFT` or `K8S`
    - `API_HOSTNAME`: hostname central is running; default `localhost`
    - `API_PORT`: central port; default `8000`
    - `ROX_USERNAME`: default `admin`
    - `ROX_ADMIN_PASSWORD`: password for `admin` for accessing Central
    - `KUBECONFIG`: kubeconfig file to use
    - `POD_SECURITY_POLICIES`: set to `false` if the underlying kubernetes cluster does not support pod security policies

  - module : `qa-test-backend.test`
- Save the configuration and run the test.

## Using command-line

If you have deployed StackRox into a cluster with roxie and roxie's post-deployment environment
has been loaded, the tests in `src/test/groovy/` can be run directly from the command-line without
setting any additional environment variables.

To run tests, from within `qa-tests-backend` directory:

- A single test: `./gradlew test --tests=TestName`, where `TestName` is the name of the test, e.g, `TestSummary`
- A single test with filtering: `./gradlew test --tests=TestName.*filter*`, where `filter` is something to match in
  the test def string, e.g, `ComplianceTest.*CVE*` matches all feature tests that include `CVE`.
- A test group: `./gradlew test -Dgroups=GroupName`, where `GroupName` is the name of the test group, e.g, `BAT`
- A makefile target: `make -C qa-backend-tests smoke-test`

### Custom configuration
If you have deployed the cluster differently or need to use a custom configuration, set `CLUSTER`, `API_HOSTNAME`,
`PORT`,`ROX_USERNAME`, `ROX_ADMIN_PASSWORD` and other relevant integration credential environment variables.

## Running a single test multiple times

To test for flakiness, you can run a single test multiple times while emulating a CI environment. This is
achieved by running the following commands:

```sh
./tests/e2e/run-e2e-tests.sh -t "$MAIN_IMAGE_TAG" -y --config-only qa
./tests/e2e/run-e2e-tests.sh -d -t "$MAIN_IMAGE_TAG" --spin-cycle=100 -y qa DiagnosticBundleTest
```

Note that access to the
[CI vault instance](https://vault.ci.openshift.org/ui/vault/secrets/kv/show/selfservice/stackrox-stackrox-e2e-tests/credentials)
is required to set up credentials as they are used in CI.

# Adding Tests
## Annotations
New tests are added with a `@Tag` annotation to indicate which to which
group the test belongs. The default test group that runs in CI is the `BAT`
group.

# Running Bits of Groovy

Developing groovy code in a test specification context has a lot of
overhead and can often be painful. For more details see [sampleScripts](src/main/groovy/sampleScripts/README.md).

# Common Failure Patterns

## When the proxy to central dies
`Connection refused: localhost/0:0:0:0:0:0:0:1:8000`

You will need to start another proxy:

`nohup oc port-forward -n stackrox svc/central 8000:443 &`

 Or use `kubectl`/`oc`:

`kubectl -n stackrox port-forward svc/central 8000:443`

## When image pulls from docker.io get throttled

You shouldn't use images from DockerHub in tests.
We don't use a paid account there and so image pulls get throttled, and
tests that use such images fail.

If you need a specific image from DockerHub, pull it, retag as
`quay.io/rhacs-eng/qa:<your-tag-here>` and push.
Then consume the new image from `quay.io/rhacs-eng/qa:<your-tag-here>`
in tests. Such pulls shouldn't get throttled.
