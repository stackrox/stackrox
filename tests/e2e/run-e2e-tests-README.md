# A local test runner for various E2E tests

`run-e2e-tests.sh` aims to provide a secure and accessible method for running
E2E tests that typically are only supported in CI. It does this by importing
test credentials from hashicorp vault and using the same test container that CI
uses. This should facilitate a test development workflow where the overhead to
make changes to test code and verify those changes is minimal.

## The Tested Image Version

By default this test runner emulates CI and relies on the output of `make tag`
to determine which images are deployed via `deploy/` scripts. When local changes
are made this tag goes `-dirty` and that suffix is dropped to facilitate local
changes to tests without the need to build and push `-dirty` images. In order to
satisfy the tight coupling between the `deploy/` scripts and roxctl version, a
matching `roxctl` is pulled from the public image and supplied to the test
container as `/usr/local/bin/roxctl`.

A `-t` flag can be used to set a particular version and override the default
behaviour of `make tag`.

## Vault Access

There are a number of required steps to get access to vault:

1. Log in to the secrets collection manager at
https://selfservice.vault.ci.openshift.org/secretcollection?ui=true (This is a
Red Hat-ism and will require SSO)
2. Ask in #epic-ci-improvement to be added to the collections required for this test:
stackrox-stackrox-initial and stackrox-stackrox-e2e-tests.
3. Login to the vault at: https://vault.ci.openshift.org/ui/vault/secrets (Use
*OIDC*) You should see these secret collections under kv/
4. Copy a 'token' from that UI and enter it at the prompt when running the
`run-e2e-tests.sh` script. (You can skip the prompt by setting this to a
VAULT_TOKEN environment variable)

The 'token' will expire hourly and you will need to renew it through the vault UI.

## Usage

For basic usage see `run-e2e-tests.sh -h`

### qa-tests-backend/ tests - 'qa' flavor

Configure only:
```
# Just configure the cluster in the current environment so it can run these tests:
run-e2e-tests.sh -c qa
```

Run a single suite (assumes a prior config step was executed):
```
# Run DeploymentTest.groovy suite:
run-e2e-tests.sh -d qa DeploymentTest
```

The `-d` option collects debug for failing tests under `/tmp/qa-tests-backend-logs/`
similar to how CI tests operate.

Run basic acceptance tests (BAT):
```
run-e2e-tests.sh qa
```

Only run tests (assumes a prior config step was executed):
```
run-e2e-tests.sh --test-only qa
```

Run tests repeatedly:
```
# Hammer on the IntegrationsSplunkViolationsTest
run-e2e-tests.sh --spin-cycle=100 qa IntegrationsSplunkViolationsTest
```

Run a gradle task (assumes a prior config step was executed):
```
# Run all @Tag("Parallel") tests
run-e2e-tests.sh qa testParallel
```

### Non groovy tests - 'e2e' flavor

Run everything just like CI:
```
run-e2e-tests.sh -d e2e
```

TBD - split configuration from test. separate the various test facets (roxctl,
integration, destructive, etc)
