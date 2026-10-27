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

## Prow resource baseline (ROX-37373)

All GKE and `ocp_*` Prow job entrypoints opt into resource sampling, including
oldest/latest and compatibility variants that reuse them. Existing node counts,
machine types, resource profiles, and test workloads are preserved. Other
platforms and GHA deployments are unchanged.

Measurements are uploaded as `${ARTIFACT_DIR}/resource-usage/samples.jsonl`:

- `node-usage` and `pod-usage`: timestamped `kubectl top` output every 30 seconds.
- `nodes`, `pods`, and `events`: scheduling snapshots every 60 seconds. Pod
  snapshots exclude container environments and commands to avoid publishing credentials.
- Failed commands record an `error`; missing measurements are not evidence of
  low utilization. Sampling is best effort and cannot replace a test failure.

Each command is limited to five seconds. Sampling stops before GKE teardown
and when OCP's test runner finishes; OCP cluster destruction remains owned by
its external automation-flavors step. SIGINT cancellation stops the sampler.

Explicitly trigger optional presubmits, including oldest/latest, Scanner install,
Operator, scale, race-condition, BYODB, and compatibility families. Collect full
runs at existing sizing before making a separate sizing change. Operator
workloads must be measured during execution, not only after namespace cleanup.

For OCP, use the node snapshot role labels and pod `nodeName` to report masters
and workers separately. Keep platform workloads in worker budgets and inspect
replica/placement requirements, including ingress anti-affinity. A CPU/RAM fit
alone does not establish that a single worker is a valid OCP topology.

For each future sizing pilot, include effective pod/init-container requests and
per-node agents, verify all requests fit allocatable capacity, and check Pending
pods, OOMs, evictions, pressure, peak/sustained usage, and runtime. Compare cost
per completed run against matching multi-node baselines. Do not lower requests
or weaken tests just to make a candidate fit. Incomplete telemetry leaves
capacity unverified; retain existing sizing until it can be validated.

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
