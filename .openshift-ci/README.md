This folder supports OpenShift CI for this repo [config](https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox).

stackrox/stackrox jobs in openshift/release typically execute the following steps:

- [begin.sh](begin.sh)
- An automation-flavors create() depending on cluster type. [optional: e.g. with GKE create() is handled in dispatch.sh]
- [dispatch.sh](dispatch.sh) is the entrypoint for tests. Depending on the job, this will run one of the scripts in [scripts/ci/jobs](../scripts/ci/jobs/).
- An automation-flavors destroy() depending on cluster type.
- [end.sh](end.sh)

The `*.py` in this folder provides some semantics useful to e2e and system tests e.g.:

- create a cluster
- run test
- gather and examine state
- teardown

## Python Style & Lint

Python code is expected to be PEP8 and checked with
[pycodestyle](https://pypi.org/project/pycodestyle/). Linting is via
[pylint](https://pypi.org/project/pylint/).

For dev workflow there are make targets:

```
make style
make fix-style
make lint
```

## Debugging

### Links

Job Definition: https://github.com/openshift/release/tree/master/ci-operator/jobs/stackrox/stackrox
Config Definition: https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox

#### Access Job Cluster and Real Time Logs

- To access the OpenShift UI to view the pod logs directly in OpenShift UI search for something like `Using namespace https://console.build01.ci.openshift.org/k8s/cluster/projects/ci-op-yz5q9nlt`.
- Access OpenShift UI, open `Administrator` overview on the top left.
- View the `Environment`, copy the `KUBECONFIG` path, open the Pod's `Terminal` view in the UI and run `cat <KUBECONFIG_PATH>`
- Copy KUBECONFIG content and create the KUBECONFIG locally
- Run `export KUBECONFIG=/local/path/to/kubeconfig`

#### Check StackRox logs and build logs

Path in artifacts:
```
download build logs: artifacts/gke-qa-e2e-tests/stackrox-stackrox-e2e-test/artifacts/howto-locate-other-artifacts-summary.html
```

## Nongroovy node-disruption monitoring

Set `GKE_NODE_DISRUPTION_WATCH=true` to monitor nodes during `NonGroovyE2e`.
Unset or any other value preserves the existing execution path. GHA nongroovy
jobs explicitly enable it; Prow jobs remain opt-in. This setting is independent
of `GKE_SPOT` and does not change cluster provisioning.

The monitor collects node names immediately before `tests/e2e/run.sh`, every
five minutes while it runs, and immediately after it completes. Each `kubectl`
call has a ten-second timeout. Product tests continue even when a later poll
fails; the monitoring verdict is decided at the end, not fail-fast.

The number of unique names across all samples must equal `NUM_NODES`, when set
to a positive integer. Otherwise the initial sample's node count is used. An
invalid configuration, failed initial poll, or empty initial sample prevents
tests from starting. Later polling errors also fail the final monitoring result.
Readiness and per-snapshot node counts are not checked: disappearance without
replacement, NotReady nodes with unchanged names, and transient replacements
between polls may go undetected. Name changes do not establish Spot causality.

`ARTIFACT_DIR/node-names.txt` contains the sampled names (including repetitions).
`ARTIFACT_DIR/node-snapshots.jsonl` retains every poll's timestamp, raw JSON
stdout, stderr, exit code, and any polling error, even on passing runs. Failure
logs include the expected count, unique names, and evidence paths; detailed
snapshots remain in artifacts rather than successful job logs. Product timeout
and cancellation still terminate the process group with TERM, then KILL after
five seconds.

`ARTIFACT_DIR/node-monitor.xml` contains the stable testcase
`ci.gke.node / NoNodeDisruption`: pass for a matching unique-name count and
error-free polling regardless of the product result, failure for a count mismatch
or monitor errors, skipped for
cancellation unless a monitor failure has already been recorded.
The GHA adapter alone publishes the `failed` step output used to skip recovery;
ordinary product failures do not set that output to true.
