# StackRox Platform Integration Tests

This framework is designed to test integration and functional flows through APIs on a running k8s or openshift cluster.

# Prerequisites
Gradle is used to build and run tests written in Groovy using the Spock test framework.

- If you would like to use the recommended IDE:
  - Download and install [IntelliJ IDEA](https://www.jetbrains.com/idea/download/)
    and create a new project from the `qa-tests-backend` directory.
- If you plan to run tests pulling from quay.io (currently every test derived
  from `BaseSpecification`), set `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` env
  vars. Read-only credentials are available in 1Password's "Everyone at StackRox"
  vault under `quay.io RO User robot`. 

# Running Tests
If protos have been changed or not generated: `make proto-generated-srcs`

## Using IntelliJ
If you have deployed StackRox into a cluster with the `deploy.sh` script,
the tests in `src/test/groovy/` can be run directly. Cluster type and login data
are inferred from the `central-deploy` directory.

### Tests dependent on integration credentials
- If your tests depend on an integration password or token in an environment variable such as:
  `GOOGLE_CREDENTIALS_GCR_SCANNER`, `EMAIL_NOTIFIER_PASSWORD`,
  `MAILGUN_PASSWORD`, `JIRA_TOKEN`, `DTR_REGISTRY_PASSWORD`, `QUAY_PASSWORD`
  - Create a `qa-tests-backend/qa-test-settings.properties` file that contains environment variable assignments.
  - Copy environment variable settings from the [StackRox 1Password Vault](https://stackrox.1password.com)

### Custom configuration
- If you have deployed the cluster differently or need to use a custom environment variable configuration:
- Go to `Run > Edit Configurations`
- Select Gradle, add a new configuration
  - Script path : `github.com/stackrox/stackrox/qa-tests-backend/src/test/groovy/<Groovy class name>.groovy`
  - Working Directory : `github.com/stackrox/stackrox/qa-tests-backend`
  - Environment Variables:
    - `CLUSTER`: Either `OPENSHIFT` or `K8S`
    - `API_HOSTNAME`: hostname central is running; default `localhost`
    - `PORT`: central port; default `8000`
    - `ROX_USERNAME`: default `admin`
    - `ROX_PASSWORD`: default read from deploy dir based on specified `CLUSTER`
    - `KUBECONFIG`: kubeconfig file to use

  - module : `qa-test-backend.test`
- Save the configuration and run the test.

## Using command-line

If you have deployed StackRox into a cluster with the `deploy.sh` script,
the tests in `src/test/groovy/` can be run directly from the command-line without
setting any environment variables. Cluster type and login data
are inferred from the `central-deploy` directory.

To run tests, from within `qa-tests-backend` directory:

- A single test: `./gradlew test --tests=TestName`, where `TestName` is the name of the test, e.g, `TestSummary`
- A single test with filtering: `./gradlew test --tests=TestName.*filter*`, where `filter` is something to match in the test def string, e.g, `ComplianceTest.*CVE*` matches all feature tests that include `CVE`.
- A test group: `./gradlew test -Dgroups=GroupName`, where `GroupName` is the name of the test group, e.g, `BAT`
- A makefile target: `make -C qa-backend-tests smoke-test`

### Custom configuration
If you have deployed the cluster differently or need to use a custom configuration, set `CLUSTER`, `API_HOSTNAME`, `PORT`,`ROX_USERNAME`, `ROX_PASSWORD` and other relevant integration credential environment variables.

## CircleCI
### Labels
Tests runs in CircleCI are controlled by CircleCI labels. Here are the labels relevant to QA tests:
  - `ci-all-qa-tests` : run ALL QA tests, not just BAT
  - `ci-no-qa-tests` : skip QA tests
  - `ci-openshift-tests` : Run tests on Openshift. This label can be combined with the previous two labels

### Spock Reports
Test outputs are integrated with spock-reports plugin.
All the reports are added under build/spock-reports folder.
The report is generated with all the tests executed with asserts for the failed and the steps executed.

# Adding Tests
## Annotations
New tests are added with a `@Category` annotation to indicate which to which
group the test belongs. The default test group that runs in CI is the `BAT`
group.

# Running Bits of Groovy

Developing groovy code in a test specification context has a lot of 
overhead and can often be a pain. Sometimes you may want to run a bit 
of kubernetes code for example. One way to do that is to use a 
standalone gradle task.

See for example: 
[build.gradle](https://github.com/stackrox/stackrox/blob/391a08e18b7386cb6ce2ec205b6489a20df0e61c/qa-tests-backend/build.gradle#L145)
the task `checkSensorCollectorHealth` corresponds to [checkSensorCollectorHealth.groovy](https://github.com/stackrox/stackrox/blob/391a08e18b7386cb6ce2ec205b6489a20df0e61c/qa-tests-backend/src/main/groovy/checkSensorCollectorHealth.groovy). This can be executed with:
```
./gradlew checkSensorCollectorHealth
```

