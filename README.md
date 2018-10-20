[![CircleCI][circleci-badge]][circleci-link]
[![Coverage Status][coveralls-badge]][coveralls-link]

# StackRox Container Security Platform

The StackRox Container Security Platform performs a risk analysis of the container
environment, delivers visibility and runtime alerts, and provides recommendations
to proactively improve security by hardening the environment. StackRox
integrates with every stage of container lifecycle: build, deploy and runtime.

Note: the StackRox platform is built on the foundation of the product formerly
known as Prevent, which itself was called Mitigate and Apollo. You may find
references to these previous names in code or documentation.

## Development
**Note**: if you want to develop only StackRox UI, please refer to [ui/README.md](./ui/README.md).

### Build Tooling
The following tools are necessary to build image(s):

 * [Make](https://www.gnu.org/software/make/)
 * [Bazel](https://docs.bazel.build/versions/master/install.html) 0.17.2 or higher.
 Install XCode before Bazel if you are building on a Mac.
 * [Go](https://golang.org/dl/) 1.11.
 * Various Go linters that can be installed using `make dev`.
 * [Node.js](https://nodejs.org/en/) `8.12.0` or higher (it's highly recommended to use an LTS version)
 If you're managing multiple versions of Node.js on your machine, consider using [nvm](https://github.com/creationix/nvm))
 * [Yarn](https://yarnpkg.com/en/)

### How to Build
```bash
make image
```

This will create `stackrox/prevent` with a tag defined by `make tag`.
This is the only image required to run the base configuration of StackRox.
Runtime collection and system monitoring require additional images.

### How to Test
```bash
make test
```

Note: there are integration tests in some components, and we currently
run those manually. They will be re-enabled at some point.

### How to Apply or Check Style Standards
```bash
make style
```

This will check Go and Javascript code for conformance with standard style
guidelines, and rewrite the relevant code if possible.

### How to Deploy
Deployment configurations are under the `deploy/` directory, organized
per orchestrator.

The deploy script will:

 1. Launch Central.
 1. Create a cluster configuration and a service identity, then
 deploy the cluster sensor using that configuration and those credentials.

You can set the environment variable `PREVENT_IMAGE_TAG` in your shell to
ensure that you get the version you want.
If you check out a commit, the scripts will launch the image corresponding to
that commit by default. The image will be pulled if needed.

Further steps are orchestrator specific.

<details><summary>Docker Swarm</summary>

Set `LOCAL_API_ENDPOINT` to a `hostname:port` string appropriate for your
local host, VM, or cluster, then:
```bash
./deploy/swarm/deploy.sh
```

When prompted, enter the credentials for whatever image registry you are
downloading StackRox Platform from. Usually, this is [Docker Hub](https://hub.docker.com).
They are necessary so that Sensor can properly deploy the Benchmark Bootstrap
service on all cluster nodes when requested.
You may set these as `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` in your
environment to avoid typing them repeatedly.

If `DOCKER_CERT_PATH` is empty in the script's environment, the script will
request that Central generate a Sensor config that excludes Docker TLS
credentials. Otherwise, the credentials currently in use in your shell
will be sent to the cluster and created as secrets for the Sensor to use.

If you are running on a local VM and do not want Swarm to pull a new image when
you submit the StackRox Platform stack (e.g., to use a locally built `:latest` tag),
use this variant instead:

```bash
./deploy/swarm/deploy-local.sh
```
</details>

<details><summary>Kubernetes</summary>

Set your Docker image-pull credentials as `REGISTRY_USERNAME` and
`REGISTRY_PASSWORD`, then run:

```bash
./deploy/k8s/deploy.sh
```
</details>

## Deploying for Customer

<details><summary>Docker Swarm (not officially supported)</summary>

Note: you may need to run `unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY`
on a fresh terminal locally so that you don't try to run an interactive container
remotely.

```
docker run -i --rm stackrox.io/prevent:<tag> interactive > swarm.zip
```

This will run you through an installer and generate a `swarm.zip` file:

```$xslt
unzip swarm.zip -d swarm
```

Note: This should be run in an environment that does have the proper cert bundle
```$xslt
bash swarm/central.sh
```

Now Central has been deployed. Use the UI to deploy Sensor.

</details>

<details><summary>Kubernetes</summary>

```
docker run -i --rm stackrox.io/prevent:<tag> interactive > k8s.zip
```

This will run you through an installer and generate a `k8s.zip` file.

```$xslt
unzip k8s.zip -d k8s
```

```$xslt
bash k8s/central.sh
```
Now Central has been deployed. Use the UI to deploy Sensor.

</details>

<details><summary>OpenShift</summary>

Note: If using a host mount, you need to allow the container to access it by using
`sudo chcon -Rt svirt_sandbox_file_t <full volume path>`

Take the image-setup.sh script from this repo and run it to do the pull/push to
local OpenShift registry. This is a prerequisite for every new cluster.
```
bash image-setup.sh
```

```
docker run -i --rm stackrox.io/prevent:<tag> interactive > openshift.zip
```

This will run you through an installer and generate a `openshift.zip` file.

```$xslt
unzip openshift.zip -d openshift
```

```$xslt
bash openshift/central.sh
```
</details>

## How to Release a New Version

Replace the value with a version number you're about to release:
```bash
export RELEASE_VERSION=0.999
```

By convention, we do not currently use a `v` prefix for release tags (that is,
we push tags like `0.5`, not `v0.5`).

### Get Ready
Proceed with the steps that under the section of the release type you're making:
non-patch or patch.

#### Non-patch Release
These steps assume that the tip of `origin/master` is what you plan to release
and that all the builds for that commit have completed successfully.

```bash
git checkout master
git pull
export RELEASE_COMMIT="$(git rev-parse HEAD)"
echo -e "Preparing to release:\n$(git log -n 1 ${RELEASE_COMMIT})"
```

#### Patch Release
Identify the release version / tag that will be patched (patch or non-patch):
```bash
export RELEASE_TO_PATCH=0.998
git fetch --tags
git checkout -b release/${RELEASE_VERSION} ${RELEASE_TO_PATCH}
```

Then use `get cherry-pick -x ${commit_sha}` to cherry pick commits from `master`
that are going into this patch release. If release requires special changes
(besides cherry picking from `master`), push the release branch and create
(and merge after code review) PR(s) targeting it.

```bash
export RELEASE_COMMIT="$(git rev-parse HEAD)"
echo -e "Preparing to release:\n$(git log -n 1 ${RELEASE_COMMIT})"
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

### Update JIRA release
Mark this version "Released" in [JIRA](https://stack-rox.atlassian.net/projects/ROX?orderField=RANK&selectedItem=com.atlassian.jira.jira-projects-plugin%3Arelease-page&status=unreleased).
Create the next one if it does not exist.

Find all bugs that are still open and affect a previous release.
Add this release to the "Affects Version(s)" list for those bugs.

### Create Release Notes
1. Go the [releases page on GitHub](https://github.com/stackrox/rox/releases).
1. Edit the corresponding tag and write release notes based on JIRA issues that
went into the current release.
1. Mark the release as "Pre-release" if QA verification is pending.

### Promote the Release for Demos / POCs
If QA and the team has approved the promotion of the release to SEs for customer
demos and POCs, then
* update the [current releases page](https://stack-rox.atlassian.net/wiki/spaces/StackRox/pages/591593496/Current+product+releases)
* remove "Pre-release" mark from [GitHub releases](https://github.com/stackrox/rox/releases)

[circleci-badge]: https://circleci.com/gh/stackrox/rox.svg?&style=shield&circle-token=140f88ea9dfd594ff68b71eaf1d4407c4331833d
[circleci-link]:  https://circleci.com/gh/stackrox/workflows/rox/tree/master
[coveralls-badge]: https://coveralls.io/repos/github/stackrox/rox/badge.svg?t=uFuaaq
[coveralls-link]: https://coveralls.io/github/stackrox/rox
