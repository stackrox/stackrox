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

## Single-node GKE pilot (ROX-37373)

The QA, non-Groovy, and UI job entrypoints explicitly provision one
`e2-standard-8` node instead of three `e2-standard-4` nodes. Oldest/latest
variants reuse these entrypoints. Other GKE jobs and OCP topology are unchanged.
These three test paths already select roxie's `ci` resource profile; this pilot
does not change component requests or shared deployment defaults.

Only these jobs enable `GKECluster(sample_resources=True)`. Temporary pilot
measurements are uploaded as `${ARTIFACT_DIR}/resource-usage/samples.jsonl`:

- `node-usage` and `pod-usage`: timestamped `kubectl top` output every 30 seconds.
- `nodes`, `pods`, and `events`: scheduling snapshots every 60 seconds. Pod
  snapshots exclude container environments and commands to avoid publishing credentials.
- Failed commands record an `error`; missing measurements are not evidence of
  low utilization. Sampling is best effort and cannot replace a test failure.

Each command is limited to five seconds. Sampling stops before cluster teardown,
including the SIGINT cancellation path.

Before accepting the sizing change, collect three complete passing runs for each
ordinary family and passing oldest/latest variants. UI and non-Groovy presubmits
run automatically; explicitly request QA and oldest/latest variants with Prow's
`/test <job>` commands. Compare against matching multi-node baseline versions:

- Requests must fit the actual node's allocatable CPU and memory, including
  platform pods, init containers, per-node agents, and transient test workloads.
- No capacity-related Pending pods, OOMs, evictions, or node pressure.
- No sustained five-minute CPU or memory utilization above 90% of allocatable
  capacity; incomplete measurements leave capacity unverified.
- Median test execution time must remain within 20% of the matched baseline.

Restore the prior sizing for a family that fails these gates rather than reducing
requests or weakening its tests. Remove the temporary sampler and its opt-ins
after validation. Three standard-4 nodes to one standard-8 reduces provisioned
CPU/RAM by one third and removes two boot disks, but report cost per completed
run as well as cost per hour.

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
