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

## E2E node disruption monitoring

Prow starts one monitor per GKE or OpenShift cluster runner after provisioning.
GitHub Actions starts a background monitor before the test and collects the
report in a separate step. Test commands run directly with their existing step
timeouts. QA Parts 1 and 2 use separate report directories. GKE validates the initial node count against the provisioning
count. OpenShift uses the initial node names as its baseline.

The monitor samples at startup, every five minutes, and at completion unless
cancelled. Each kubectl call has a 30-second timeout. The first usable snapshot establishes the baseline, even after a failed startup
probe. Changes in node membership and sampling errors produce skipped JUnit
cases with advisory details and preserve the suite outcome.
Changes between samples may go undetected.

The monitor writes `junit-node-monitor.xml`, `node-monitor.log`, sampled names,
and raw snapshots under `ARTIFACT_DIR` (default `junit-reports`). The JUnit
case is `ci.gke.node.NoNodeDisruption`. Abrupt termination may prevent a final
report. Advisory results do not suppress fallback job-failure reports.

Cluster-free tests live in `tests/test_node_disruption.py`. From the repo root:

```sh
PYTHONPATH=.openshift-ci python3 -m unittest discover -s .openshift-ci/tests -p test_node_disruption.py
```

CI runs `make -C .openshift-ci test` in Unit Tests / `openshift-ci-unit-tests`,
except for UI-only changes.
