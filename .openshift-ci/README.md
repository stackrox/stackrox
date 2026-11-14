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

Prow's `NonGroovyE2e` monitors nodes when `NUM_NODES` is a positive integer
(exported by `GKECluster`). Unset or empty disables monitoring. It is independent
of `GKE_SPOT`; the nongroovy GHA workflow does not enable it.

Node names are collected before tests, every five minutes, and after tests unless
cancelled, with a ten-second limit per kubectl call. The unique-name count must
match `NUM_NODES`. Polling errors, invalid configuration, and an empty initial
sample also produce diagnostic failures; later empty samples are ignored.
Monitoring and artifact errors are advisory: they never change the product exit
code or prevent tests from starting.

Artifacts under `ARTIFACT_DIR` (default `junit-reports`):
- `node-names.txt`: sampled names, including repetitions.
- `node-snapshot-<timestamp>.json`: raw kubectl output, possibly partial.
- `node-monitor.log`: collection attempts and kubectl stderr.
- `node-monitor.xml`: one `ci.gke.node / NoNodeDisruption` success/failure/skipped
  testcase. Cancellation is skipped unless an error was recorded. Failure details
  go into JUnit; counts, names, and errors also appear in job logs.

This checks names, not readiness or per-snapshot counts. Disappearance without
replacement and transient nodes may go undetected. Replacement does not prove
Spot preemption.

Optional CLI: `python3 .openshift-ci/node_disruption.py run -- <command>`.
Prow calls `run(timeout=...)`; the CLI has no timeout option. Both preserve product
exit codes and clean up process groups on cancellation.

Cluster-free tests live in `tests/test_node_disruption.py`. From the repo root:

```sh
PYTHONPATH=.openshift-ci python3 -m unittest discover -s .openshift-ci/tests -p test_node_disruption.py
```

CI runs `make -C .openshift-ci test` in Unit Tests / `openshift-ci-unit-tests`,
except for UI-only changes.
