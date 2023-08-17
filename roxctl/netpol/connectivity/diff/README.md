# Static Network Policy Connectivity Diff

## Technology Preview Notice

The static network policy connectivity diff feature is offered as a developer preview feature.
While we are open to receiving feedback about this feature, our technical support will not be able to assist and answer questions about it.

## About

The static network policy connectivity diff is a tool that analyzes two sets of Kubernetes manifests, including network policies.
Based on two given folders containing deployment and network policy YAMLs, it analyzes the permitted cluster connectivity for each input folder.
It produces a list of a **differences in terms of allowed connections**, based on the workloads and network policies defined.
It is based on [NP-Guard's Network Policy Analyzer component](https://github.com/np-guard/netpol-analyzer). For more details, refer to the [NP-Guard webpage](https://np-guard.github.io/).

## Command Objective

Generate a file that allows users to visualize the **connectivity diff** between two versions of workloads and network policy manifests.

## Usage

### Producing connectivity diff from YAML manifests (network policies and workload resources)

To produce a connectivity-diff report, the command `roxctl netpol connectivity diff --dir1=<folder1> --dir2=<folder2>` requires two folders, `dir1` and `dir2`, each containing Kubernetes manifests, including network policies.
The manifests must not be templated (e.g., Helm charts) to be considered. All YAML files that could be accepted by `kubectl apply -f` will be accepted as a valid input and searched by `roxctl netpol connectivity diff`.

#### Syntactic vs semantic diff:

The example shown below has two versions, where `dir1` is `netpol-analysis-example-minimal/` , and `dir2` is  `netpol-diff-example-minimal/`.
The difference between the dirs consists of a small change in network policy `backend-netpol`.

The policy from `dir1`:

```
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
status: {}

```

The change on `dir2` is an added `-` before the `ports` attribute. The syntactic diff output:
```
$ diff netpol-diff-example-minimal/netpols.yaml netpol-analysis-example-minimal/netpols.yaml
12c12
<   - ports:
---
>     ports:
```

Semantic diff output (plain text format):

```
$ roxctl netpol connectivity diff --dir1=roxctl/netpol/connectivity/diff/testdata/netpol-analysis-example-minimal/ --dir2=roxctl/netpol/connectivity/diff/testdata/netpol-diff-example-minimal
Connectivity diff:
diff-type: changed, source: default/frontend[Deployment], destination: default/backend[Deployment], dir1:  TCP 9090, dir2: TCP 9090,UDP 53
diff-type: added, source: 0.0.0.0-255.255.255.255, destination: default/backend[Deployment], dir1:  No Connections, dir2: TCP 9090
```

Semantic diff output in `md` format:

| diff-type | source | destination | dir1 | dir2 | workloads-diff-info |
|-----------|--------|-------------|------|------|---------------------|
| changed | default/frontend[Deployment] | default/backend[Deployment] | TCP 9090 | TCP 9090,UDP 53 |  |
| added | 0.0.0.0-255.255.255.255 | default/backend[Deployment] | No Connections | TCP 9090 |  |

Connectivity report from `dir1`:
```
$ roxctl connectivity-map netpols-analysis-example-minimal/
0.0.0.0-255.255.255.255 => default/frontend[Deployment] : TCP 8080
default/frontend[Deployment] => 0.0.0.0-255.255.255.255 : UDP 53
default/frontend[Deployment] => default/backend[Deployment] : TCP 9090
```

Connectivity report from `dir2`:
```
$ roxctl connectivity-map netpol-diff-example-minimal/
0.0.0.0-255.255.255.255 => default/backend[Deployment] : TCP 9090
0.0.0.0-255.255.255.255 => default/frontend[Deployment] : TCP 8080
default/frontend[Deployment] => 0.0.0.0-255.255.255.255 : UDP 53
default/frontend[Deployment] => default/backend[Deployment] : TCP 9090,UDP 53
```

The semantic-diff report provides a summary of changed/added/removed connections from `dir2` with respect to allowed connections from `dir1`.

### Understanding the output
Each line in the output represents an allowed connection that has been added/removed/changed on `dir2` with respect to `dir1`.
The `workloads-diff-info` adds information about added/removed workload related to the added/removed connection, if relevant.

## Parameters

The output can be redirected to a file by using `--output-file` parameter.

The output format can be set by using the `--output-format` parameter. Supported output formats: `txt, md, csv`.

When running in a CI pipeline, `roxctl netpol connectivity diff` may benefit from the `--fail` option that stops the processing on the first encountered error.

Using the `--strict` parameter produces an error "there were errors during execution" if any warnings appeared during the processing. Note that the combination of `--strict` and `--fail` will not stop on the first warning, as the interpretation of warnings as errors happens at the end of execution.
