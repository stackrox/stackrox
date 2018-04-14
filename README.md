# StackRox Prevent

Prevent is a new StackRox initiative to provide security in the
deployment phase of the container lifecycle.

## Build Tooling
**Note**: if you want to develop only Prevent UI, please refer to [ui/README.md](./ui/README.md) for dev env setup instructions.

Prevent is distributed as a single image. The following build tools are
required to completely build that image and run tests:

 * Make
 * [Bazel](https://docs.bazel.build/versions/master/install.html) 0.9 or higher.
 Install XCode before Bazel if you are building on a Mac.
 * [Yarn](https://yarnpkg.com/en/)
 * [Go](https://golang.org/dl/)
 * Various Go linters that can be installed using `make -C central dev`

## How to Build
```bash
make image
```

This will create `stackrox/prevent:latest`. This is the only image required
to run StackRox Prevent.

## How to Test
```bash
make test
```

Note: there are integration tests in some components, and we currently
run those manually. They will be re-enabled at some point.

## How to Apply or Check Style Standards
```bash
make style
```

This will rewrite Go code to conform to standard style guidelines.
JavaScript code is only checked, not rewritten.

## How to Deploy
Deployment configurations are under the `deploy/` directory, organized
per orchestrator.

_**WARNING:** You are looking at the tip of the development tree.
If you need to create a customer demo, use the instructions for the
[latest stable version](https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/233242976/StackRox+Prevent)._

The deploy script will:

 1. Launch Central.
 1. Create a cluster configuration and a service identity, then
 deploy the cluster sensor using that configuration and those credentials.

You can (and likely should) set the environment variable `PREVENT_IMAGE_TAG`
in your shell to ensure that you get the version you want.

### Docker Swarm

#### Deploy
Set `LOCAL_API_ENDPOINT` to a `hostname:port` string appropriate for your
local host, VM, or cluster, then:

```bash
./deploy/swarm/deploy.sh
```

When prompted, enter the credentials for whatever image registry you are
downloading StackRox Prevent from. Usually, this is [Docker Hub](https://hub.docker.com).
They are necessary so that Sensor can properly deploy the Benchmark Bootstrap
service on all cluster nodes when requested.
You may set these as `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` in your
environment to avoid typing them repeatedly.

If `DOCKER_CERT_PATH` is empty in the script's environment, the script will
request that Central generate a Sensor config that excludes Docker TLS
credentials. Otherwise, the credentials currently in use in your shell
will be sent to the cluster and created as secrets for the Sensor to use.

If you are running on a local VM and do not want Swarm to pull a new image when
you submit the StackRox Prevent stack (e.g., to use a locally built `:latest` tag),
use this variant instead:

```bash
./deploy/swarm/deploy-local.sh
```

#### Monitoring
You can deploy Prometheus to monitor the services:

```bash
docker stack deploy -c prometheus/swarm.yaml prevent-health
```

### Kubernetes

#### Deploy
Set your Docker image-pull credentials as `REGISTRY_USERNAME` and
`REGISTRY_PASSWORD`, then run:

```bash
./deploy/k8s/deploy.sh
```

#### Exposing the UI
The script will provide access the UI using a local port-forward, but you can
optionally create a LoadBalancer service to access Central instead.

```bash
kubectl create -f deploy/k8s/lb.yaml
```

#### RBAC
If you are deploying Sensor into a cluster with RBAC enabled (generally,
this applies to Kubernetes >=1.8), you need to create RBAC bindings.

In some environments, you may need to elevate privileges to execute this;
for instance, in GKE, you need to pass kubectl `--username` and `--password`
from `gcloud container clusters describe --format=json [NAME] | jq .masterAuth`.)

```bash
kubectl create -f deploy/k8s/rbac.yaml
```

#### Monitoring
You can deploy Prometheus to monitor the services:

```bash
kubectl create -f prometheus/k8s.yaml
```
Create a port forward to the pod on port 9090 to access the UI.

## How to Release a New Version
Releasing a new version of StackRox Prevent requires only a few steps.

These steps assume that the tip of `origin/master` is what you plan to release,
and that the build for that commit has completed.

### Get Ready
```bash
git checkout master
git pull
export RELEASE_COMMIT="$(git rev-parse HEAD)"
echo "Preparing to release ${RELEASE_COMMIT}"
```

Decide the release version and export it into your shell for convenience,
for example:

```bash
export RELEASE_VERSION=0.999
```

By convention, we do not currently use a `v` prefix for release branches and
tags (that is, we push branches like `release/0.5` and tags like `0.5`,
not `release/v0.5` and `v0.5`).

### Create a Release Branch (for non-patch releases)
```bash
git checkout -b release/${RELEASE_VERSION}
git push -u origin release/${RELEASE_VERSION}
```

### Create a Tag
```bash
git tag -a -m "v${RELEASE_VERSION}" "${RELEASE_VERSION}"
git tag -ln "${RELEASE_VERSION}"
git push origin "${RELEASE_VERSION}"
```

### Push to Docker Hub
```bash
export FROM="stackrox/prevent:${RELEASE_COMMIT}"
export TO="stackrox/prevent:${RELEASE_VERSION}"
docker pull "${FROM}"
docker tag "${FROM}" "${TO}"
docker push "${TO}"
```

### Push to stackrox.io
```bash
export FROM="stackrox/prevent:${RELEASE_VERSION}"
export TO="stackrox.io/prevent:${RELEASE_VERSION}"
docker tag "${FROM}" "${TO}"
docker push "${TO}"
```

### Modify Demo Instructions
The StackRox Prevent demo instructions live in a [Google Drive folder](https://drive.google.com/drive/folders/1gem9vG0Z0hzokF7S_r4WGwXDCCXi6fbT).

1. Copy the current latest version of the instructions to a new Google Doc.
1. Update the instructions at the top of the document to reference the new
release version git and Docker image tags.
1. Run through the entire document and make sure that everything works.
1. If there are new features to showcase, consider modifying the demo
instructions to demonstrate them.

### Update JIRA release
Mark this version "Released" in JIRA. Create the next one if it does not exist.

Find all bugs that are still open and affect a previous release.
Add this release to the "Affects Version(s)" list for those bugs.

### Publish a Confluence Page for the Version
Copy the "Latest Stable Version" page, update it, and replace the link on
[Prevent wiki homepage](https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/233242976/StackRox+Prevent).

## POC with Clair

### Launch Clair
Clair requires persistence.
If everything is running on one host, then use the below
```bash
docker run -d -p 6060:6060 \
    -e  "LANG=en_US.utf8" \
    -e  "PGDATA=/var/lib/postgresql/data" \
    -e "CLAIR_MINIMUM_SEVERITY=Low" \
    stackrox/scanner:latest
```

### Upload to Clair

This will launch the clair integration from the Prevent image.
It pushes all active images in Prevent to Clair.
This could take quite a while.
```bash
docker run -it --entrypoint=/prevent/clair \
    --net=host \
    -v $HOME/.docker:/config \
    stackrox/prevent:latest \
    --clair http://localhost:6060 \
    -m ${LOCAL_API_ENDPOINT}
```

### Manual steps

* Add registry integrations through UI
* Add Clair integration through UI
* If nothing is showing up, you may want to hit "Reassess" just to reprocess all of the data

### Notes

To send an individual image to Clair, run:
```bash
docker run -it \
    --entrypoint=/prevent/clair \
    --net=host -v $HOME/.docker:/config \
    stackrox/prevent:latest \
    --clair http://localhost:6060 \
    --image docker.io/stackrox/prevent:latest
```

Overrides for registries can be done using "PREVENT_REGISTRY_OVERRIDE"
```bash
PREVENT_REGISTRY_OVERRIDE=example.io=http://example.io,docker.io=registry-1.docker.io
```
Overrides are necessary for http registries

Password overrides for Mac where we can't read the docker config
e.g. `PREVENT_REGISTRY_AUTH=docker.io=<base64 password>`
```bash
echo "Enter image registry (e.g. docker.io)" && \
read registry && \
echo "Enter username and password separated by a colon" && \
read -s auth && \
echo ${registry}=$(echo $auth | base64 | tr -- '+=/' '-_~')
```
