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

This will create `stackrox/prevent` with a tag defined by
`git describe --tags --abbrev=10 --dirty`. This is the only image required
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

#### Deploy for Development
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

#### Deploy for Customer

Note: you may need to run `unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY`
on a fresh terminal locally so that you don't try to run an interactive container remotely.
```
docker run -i --rm stackrox.io/prevent:<tag> interactive 1>swarm.zip
```

This will run you through an installer as follows and generates a swarm.zip file:
```$xslt
docker run -i --rm stackrox.io/prevent:1.2 interactive 1>swarm.zip
Enter orchestrator (dockeree, k8s, openshift, swarm): swarm
Enter image to use (default: 'stackrox.io/prevent:1.3'): stackrox.io/prevent:1.2
Enter public port to expose (default: '443'):
Enter volume (optional) (external, hostpath): hostpath
Enter path on the host (default: '/var/lib/prevent'):
Enter mount path inside the container (default: '/var/lib/prevent'):
Enter hostpath volume name (default: 'prevent-db'):
Enter node selector key (default: 'node.hostname'):
Enter node selector value: roxbase2
```

```$xslt
unzip swarm.zip -d swarm
```

Note: This should be run in an environment that does have the proper cert bundle
```$xslt
bash swarm/deploy.sh
```
Now central has been deployed and use the UI to deploy sensor

#### Monitoring
You can deploy Prometheus to monitor the services:

```bash
docker stack deploy -c prometheus/swarm.yaml prevent-health
```

### Kubernetes

#### Deploy for Development
Set your Docker image-pull credentials as `REGISTRY_USERNAME` and
`REGISTRY_PASSWORD`, then run:

```bash
./deploy/k8s/deploy.sh
```

#### Deploying for Customer

```
docker run -i --rm stackrox.io/prevent:<tag> interactive 1>k8s.zip
```

This will run you through an installer as follows and generates a k8s.zip file.
The below works on GKE and creates an external volume
```$xslt
docker run -i --rm stackrox.io/prevent:1.2 interactive 1>k8s.zip
Enter orchestrator (dockeree, k8s, openshift, swarm): k8s
Enter image to use (default: 'stackrox.io/prevent:1.3'): stackrox.io/prevent:1.2
Enter image pull secret (default: 'stackrox'):
Enter namespace (default: 'stackrox'):
Enter volume (optional) (external, hostpath): external
Enter mount path inside the container (default: '/var/lib/prevent'):
```

```$xslt
unzip k8s.zip -d k8s
```

```$xslt
bash k8s/deploy.sh
```
Now central has been deployed and use the UI to deploy sensor

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

### OpenShift

#### Deployment for Customer

Note: If using a host mount, you need to allow the container to access it by using
`sudo chcon -Rt svirt_sandbox_file_t <full volume path>`

Take the image-setup.sh script from this repo and run it to do the pull/push to local OpenShift registry
This is a prerequisite for every new cluster
```
bash image-setup.sh
```

```
docker run -i --rm stackrox.io/prevent:<tag> interactive 1>openshift.zip
```

This will run you through an installer as follows and generates a openshift.zip file.
```$xslt
docker run -i --rm stackrox.io/prevent:1.2 interactive 1>openshift.zip
Enter orchestrator (dockeree, k8s, openshift, swarm): openshift
Enter image to use (default: 'docker-registry.default.svc:5000/stackrox/prevent:1.3'): docker-registry.default.svc:5000/stackrox/prevent:1.2
Enter namespace (default: 'stackrox'):
Enter volume (optional) (external, hostpath):
```

```$xslt
unzip openshift.zip -d openshift
```

```$xslt
bash openshift/deploy.sh
```


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

When you push the tag to GitHub, CircleCI will start a build and will push
the image as `stackrox/prevent:[your-release-tag]`,
for example `stackrox/prevent:1.0` and `stackrox.io/prevent:1.0`.

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

Also, update the [current releases page](https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/591593496/Current+product+releases)
so that the team knows which versions to deploy to customers.
