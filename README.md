# StackRox Mitigate

Mitigate is a new StackRox initiative to provide security in the
deployment phase of the container lifecycle.

## Build Tooling
The following build tools are required:

 * Make
 * [Bazel](https://docs.bazel.build/versions/master/install.html) 0.9 or higher.
 Install XCode before Bazel if you are building on a Mac.
 * [Yarn](https://yarnpkg.com/en/)
 * [Go](https://golang.org/dl/)
 * Various Go linters that can be installed using `make -C central dev`

## How to Build
```
make image
```

This will create `stackrox/mitigate:latest`. This is the only image required
to run Mitigate.

## How to Test
```
make test
```

Note: there are integration tests in some components, and we currently
run those manually. They will be re-enabled at some point.

## How to Deploy
Deployment configurations are under the `deploy/` directory, organized
per orchestrator.

**WARNING:** You are looking at the tip of the development tree.
If you need to create a customer demo, use the latest release version.

The deploy script will:

 1. Launch Central.
 1. Create a cluster configuration and a service identity, then
 deploy the cluster sensor using that configuration and those credentials.

### Docker Swarm

Set `LOCAL_API_ENDPOINT` to a `hostname:port` string appropriate for your
local host, VM, or cluster, then:

```
./deploy/swarm/deploy.sh
```

Currently, this script works on a Swarm worker that uses TLS certificate
bundles and TCP connections.

If you need to run in an environment without this configuration, you can remove
references to the following secrets from both the `secrets` and `volumes`
portions of the sensor-deploy YAML:

 * `docker_client_ca_pem`
 * `docker_client_cert_pem`
 * `docker_client_key_pem`
 * `registry_auth`

 Additionally, remove these environment variables from the sensor-deploy YAML:

 * `DOCKER_HOST`
 * `DOCKER_TLS_VERIFY`
 * `DOCKER_CERT_PATH`


### Kubernetes
Set your Docker image-pull credentials as `DOCKER_USER` and `DOCKER_PASS`, then run:

```
./deploy/k8s/deploy.sh
```

The script will access the UI using a local port-forward, but you can
optionally create a LoadBalancer service to access Central instead.

```
kubectl create -f deploy/k8s/lb.yaml
```
