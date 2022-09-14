# Static Network Policy Generator

## Developer Preview Notice

The static network policy generation feature is offered as a _developer preview_ feature.
While we are open to receiving feedback about this feature, our technical support will not be able to
assist and answer questions about it.

## About

The static network policy generator is a tool that analyzes k8s manifests and generates a set of suggested network policies in form of yaml documents that may be directly applied to a k8s cluster. It is integrated with [NP-Guard's Cluster Topology Analyzer](https://github.com/np-guard/cluster-topology-analyzer) component, which discovers the network connections and generates the network policies. For more details, refer to the [NP-Guard webpage](https://np-guard.github.io/).

## Usage

### Compiling

The feature `roxctl generate netpol` is currently not available in the officially released images of `roxctl` and must be compiled locally after fetching the stackrox repository.

Refer to [the build tooling section in the Readme](https://github.com/stackrox/stackrox#build-tooling) for details about build prerequisites.

```shell
git clone https://github.com/stackrox/stackrox.git && cd stackrox
# Compile roxctl for a given OS with empty GOTAGS
make cli-{darwin,linux} GOTAGS=''
# Set feature-flag
export ROX_ROXCTL_NETPOL_GENERATE="true"
# Confirm feature availability

$ bin/darwin_amd64/roxctl generate netpol -h
Based on a given folder containing deployment YAMLs, will generate a list of recommended Network Policies. Will write to stdout if no output flags are provided.

Usage:
  bin/darwin_amd64/roxctl generate netpol <folder-path> [flags]

Flags:
      --fail                 fail on the first encountered error (default false)
  -h, --help                 help for netpol
  -d, --output-dir string    save generated policies into target folder - one file per policy
  -f, --output-file string   save and merge generated policies into a single yaml file
      --remove               remove the output path if it already exists (default false)
      --strict               treat warnings as errors (default false)
(...)
```

### Generating Network Policies from yaml manifests

To generate network policies, `roxctl generate netpol` requires a folder containing K8s manifests.
The manifests must not be templated (e.g., Helm) to be considered.
All yaml files that could be accepted by `kubectl apply -f` will be accepted as as valid input and searched by `roxctl generate netpol`.

Example run with the output generated to `stdout`:

```shell
$ git clone --depth=1 https://github.com/stackrox/stackrox.git && cd stackrox
$ bin/darwin_amd64/roxctl generate netpol tests/roxctl/bats-tests/test-data/np-guard/scenario-minimal-service
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: backend-netpol
spec:
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - port: 9090
      protocol: TCP
  podSelector:
    matchLabels:
      app: backendservice
  policyTypes:
  - Ingress
  - Egress

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: frontend-netpol
spec:
  egress:
  - ports:
    - port: 9090
      protocol: TCP
    to:
    - podSelector:
        matchLabels:
          app: backendservice
  - ports:
    - port: 53
      protocol: UDP
  ingress:
  - ports:
    - port: 8080
      protocol: TCP
  podSelector:
    matchLabels:
      app: frontend
  policyTypes:
  - Ingress
  - Egress
```

### Parameters

The output can be redirected to a single file by using `--output-file=out.yaml` parameter.

When expecting multiple network policies to be generated on the output, the user can choose the `--output-dir=<name>` option to generate the policies into a folder where each network policy will be output to a separate file.

When running in a CI pipeline, `roxctl generate netpol` may benefit from the `--fail` option that stops the processing on the first encountered error.

Using the `--strict` parameter produces an error "_there were errors during execution_" if any warnings appeared during the processing. Note that the combination of `--strict` and `--fail` will not stop on the first warning, as the interpretation of warnings as errors happens at the end of execution.
