# ACS OpenShift Tests Extension (OTE)

This directory contains the OpenShift Tests Extension (OTE) binary for Advanced Cluster Security (ACS) testing.

## Overview

The `acs-tests` binary implements the OTE v1.1 protocol to enable ACS testing within the OpenShift test framework using the `openshift-tests` command with the `--extension-local-binaries` flag.

## Structure

```
openshift-tests-ext/
├── cmd/acs-tests/       # Binary entry point
├── pkg/adapter/         # Generic test execution adapter
├── test/acs/            # Test scenarios (JSON)
├── scripts/             # Test scripts
├── go.mod               # Go module definition
└── Makefile             # Build automation
```

## Building

```bash
make build
```

## Running Tests

### Standalone (for development)

```bash
# Run all ACS tests
./acs-tests run-suite acs/all

# Run smoke tests only
./acs-tests run-suite acs/smoke

# List available tests
./acs-tests list tests

# Show extension info
./acs-tests info
```

### With openshift-tests (production)

```bash
openshift-tests run-suite acs/all \
  --extension-local-binaries /usr/bin/acs-tests.gz \
  --junit-dir ./junit
```

## Test Suites

- `acs/all` - All ACS tests
- `acs/smoke` - ACS smoke tests (subset with SMOKE tag)

## Adding Tests

Tests are defined in `test/acs/scenarios.json`. To add a new test:

1. Create a test script in `scripts/`
2. Add a scenario entry to `scenarios.json`
3. Rebuild the binary with `make build`

## CI Integration

This binary is built as part of CI composite containers and invoked by step-registry refs in the openshift/release repository.

See: `ci-operator/step-registry/interop/acs/` in openshift/release

## Related

- OpenShift Tests Extension framework: https://github.com/openshift-eng/openshift-tests-extension
- OTE Protocol spec: https://github.com/openshift-eng/openshift-tests-extension/blob/main/docs/protocol.md
- Component Readiness: https://github.com/openshift/component-readiness
