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

## Nongroovy node-name monitoring

Set `NUM_NODES` to the expected positive node count to enable monitoring during
`NonGroovyE2e`. When it is unset or empty, tests run normally without node polls,
node artifacts, or a node-monitor JUnit report. Monitoring is independent of
`GKE_SPOT`; it works for regular and Spot nodes alike. Python's
`GKECluster(num_nodes=...)` exports the same `NUM_NODES` used for provisioning.
There is no separate monitoring flag or initial-count fallback.

The monitor samples nodes before tests, every five minutes while tests run, and
once at the end. It then compares the number of unique observed names with
`NUM_NODES`. A mismatch or polling error produces a warning and diagnostic JUnit failure,
but never changes the product exit code or prevents tests from starting.
Invalid configuration, failed initial polls, empty samples, and report-writing
errors are advisory too; unavailable artifacts are reported in logs.
Each `kubectl` invocation is limited to ten seconds.

Artifacts under `ARTIFACT_DIR` (default `junit-reports`):
- `node-names.txt`: sampled names, including repetitions.
- `node-snapshot-<timestamp>.json`: raw node JSON from each poll, retained even
  on passing runs. Failed calls may leave partial or empty snapshots.
- `node-monitor.log`: timestamped poll attempts and kubectl stderr. Poll errors
  and the final verdict are included in the JUnit report and job log.
- `node-monitor.xml`: testcase `ci.gke.node / NoNodeDisruption`. It passes for a
  matching unique-name count and error-free polling, independently of the product
  result; cancellation is skipped unless a monitor error was already recorded.
  The report contains only the outcome and failure explanation, without run links
  or extra output metadata. Job logs show the expected count, observed names, and
  polling errors.

This checks names, not readiness or per-snapshot counts. NotReady nodes with
unchanged names, disappearance without replacement, and transient nodes between
polls may go undetected. A replacement is not proof of Spot preemption.

GHA invokes `python3 .openshift-ci/node_disruption.py run -- <command>`; Prow
imports `run()` directly. Monitoring writes only logs and JUnit, without GitHub
step outputs or workflow annotations, and does not skip post-test steps. Both
paths retain product exit codes, timeout handling, and process-group cleanup on
cancellation.
