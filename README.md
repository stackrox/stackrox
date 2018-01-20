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

```
./deploy/swarm/deploy.sh
```

Currently, this script works on a Swarm worker that does not have TLS enabled.
Future changes will automate setup for clusters with such configurations, but
in the meantime you can manually edit sensor-remote-deploy.yaml to add the
following secrets:

```
secrets:
  rox_docker_client_ca_pem:
    external: true
  rox_docker_client_cert_pem:
    external: true
  rox_docker_client_key_pem:
    external: true
  rox_registry_auth:
    external: true
```

and mounts:

```
    secrets:
    - source: rox_docker_client_ca_pem
      target: ca.pem
      mode: 400
    - source: rox_docker_client_cert_pem
      target: cert.pem
      mode: 400
    - source: rox_docker_client_key_pem
      target: key.pem
      mode: 400
    - source: rox_registry_auth
      target: rox_registry_auth
      mode: 400
```

To create those secrets, use `roxc`. A typical invocation might look like:

```
roxc system setup --platform=swarm \
    --registry-username $(cat .buildUsername) \
    --registry-password $(cat .buildPassword) \
    --swarm-client-cert-path /tmp/certs
```

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
