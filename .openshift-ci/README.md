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

### Single-node OpenShift experiment (ROX-37373)

OCP 4.22 Operator, non-Groovy, Scanner-install, QA and UI jobs use one
`e2-standard-16` control-plane node and zero workers. There is no opt-in flag.
`begin.sh` writes these settings to `shared_env` before automation-flavors
provisions the cluster. Other versions and specialized suites keep their
existing topology.

This is an experiment awaiting PR CI results, not validated sizing. Inspect the
rendered installer configuration, actual node count, ingress/operator health,
scheduling, CPU/memory peaks, runtime and teardown before promotion. Include
bootstrap, failed attempts and retries when comparing cost per successful run.
Keep application resources and test concurrency unchanged while sizing the node.

Before merging, establish passing multi-node coverage for each affected
suite/stream and verify lower cost without new topology failures or more than
10% worse p95 end-to-end runtime. Keep suites requiring multiple nodes on their
existing topology. Roll back by removing the SNO selection in `ocp-topology.sh`.
Expand to other streams only after verifying their provisioning compatibility.

Local checks:

```sh
python3 -m unittest discover -s .openshift-ci/tests -p test_ocp_topology.py
bash -n .openshift-ci/begin.sh .openshift-ci/ocp-topology.sh
```

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
