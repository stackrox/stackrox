[![CircleCI][circleci-badge]][circleci-link]
[![Coverage Status][coveralls-badge]][coveralls-link]

# StackRox Kubernetes Security Platform

The StackRox Kubernetes Security Platform performs a risk analysis of the
container environment, delivers visibility and runtime alerts, and provides
recommendations to proactively improve security by hardening the environment.
StackRox integrates with every stage of container lifecycle: build, deploy and
runtime.

Note: the StackRox Kubernetes Security platform is built on the foundation of 
the product formerly known as Prevent, which itself was called Mitigate and
Apollo.  You may find references to these previous names in code or
documentation.

## Table of contents

* [Development](#development)
    + [Quickstart](#quickstart)
      - [Build Tooling](#build-tooling)
      - [Clone StackRox](#clone-stackrox)
      - [Local development](#local-development)
      - [Common Makefile Targets](#common-makefile-targets)
      - [Productivity](#productivity)
    + [How to Deploy](#how-to-deploy)
* [Deploying for Customer](#deploying-for-customer)
* [How to Release a New Version](#how-to-release-a-new-version)

## Development
**UI Dev Docs**: please refer to [ui/README.md](./ui/README.md)

**E2E Dev Docs**: please refer to [qa-tests-backend/README.md](./qa-tests-backend/README.md)

### Quickstart

#### Build Tooling

The following tools are necessary to build image(s):

 * [Make](https://www.gnu.org/software/make/)
 * [Go](https://golang.org/dl/)
   * Get the version specified in [EXPECTED_GO_VERSION](./EXPECTED_GO_VERSION).
 * Various Go linters and RocksDB dependencies that can be installed using `make reinstall-dev-tools`.
 * UI build tooling as specified in [ui/README.md](ui/README.md#Build-Tooling).
 * Docker
 * rocksdb
 * xcode command line tools (macOS only)

##### xcode - macOS only
<details><summary>Click to expand</summary>
 Usually you would have these already installed by brew.
 However if you get an error when building the golang x/tools,
 try first making sure the EULA is agreed by:
 
 1. starting XCode
 2. building a new blank app project
 3. starting the blank project app in the emulator
 4. close both the emulator and the XCode, then
 5. run the following commands:
 
 ```
 xcode-select --install
 sudo xcode-select --switch /Library/Developer/CommandLineTools # Enable command line tools
 sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
 ```
 
 For more info, see https://github.com/nodejs/node-gyp/issues/569
 </details>
 
#### Clone StackRox

```
# Create a GOPATH: this is the location of your Go "workspace".
# (Note that it is not – and must not – be the same as the path Go is installed to.)
# The default is to have it in ~/go/, or ~/development, but anything you prefer goes.
# Whatever you decide, create the directory, and add a line in your ~/.bash_profile
# exporting the env variable:
export GOPATH=$HOME/go # Change this if you choose to use a different workspace.
export PATH=$PATH:$GOPATH/bin
source ~/.bash_profile

$ cd $GOPATH
$ mkdir bin pkg src

# Replace https git-urls with ssh, required to fetch go dependencies.
$ git config --global --add url.git@github.com:.insteadof https://github.com/

# Instruct Go to bypass the Go package proxy for our private dependencies
$ go env -w GOPRIVATE=github.com/stackrox

$ cd $GOPATH
$ mkdir -p src/github.com/stackrox
$ cd src/github.com/stackrox
$ git clone git@github.com:stackrox/rox.git
```

#### Local development

Development can either happen in GCP or locally with
[Docker Desktop](https://docs.docker.com/docker-for-mac/#kubernetes) or [Minikube](https://minikube.sigs.k8s.io/docs/start/).  
To sweeten your experience, install [the workflow scripts](#productivity) beforehand.

```
# Install rocksdb, central's main database
# It is necessary because of several CGO bindings
$ brew install rocksdb

$ cd $GOPATH/src/github.com/stackrox
$ make install-dev-tools
$ make image

# To mount local StackRox binaries into your pods, enable hotreload:
$ export HOTRELOAD=true

# To keep the StackRox central's rocksdb state between restarts, set:
$ export STORAGE=pvc

# When you deploy locally make sure your kube context points to the desired kubernetes cluster,
# for example Docker Desktop.
# To check the current context you can call a workflow script:
$ roxkubectx

# To deploy locally, call:
$ ./deploy/k8s/deploy-local.sh

# Now you can access StackRox dashboard at https://localhost:8000
# or simply call another workflow script:
$ logmein
```

See the [deployment guide](#how-to-deploy) for further reading.


#### Common Makefile Targets

```bash
# Build image, this will create `stackrox/main` with a tag defined by `make tag`.
$ make image

# Compile all binaries
$ make main-build-dockerized

# Displays the docker image tag which would be generated
$ make tag

# Note: there are integration tests in some components, and we currently 
# run those manually. They will be re-enabled at some point.
$ make test

# Apply and check style standards in Go and JavaScript
$ make style

# enable pre-commit hooks for style checks
$ make init-githooks

# Compile and restart only central
$ make fast-central

# Compile only sensor
$ make fast-sensor

# Only compile protobuf
$ make proto-generated-srcs

# Update files embedded in binaries, useful when working on the Helm charts, for instance.
$ make go-packr-srcs
```

#### Productivity

The [workflow repository](https://github.com/stackrox/workflow) contains some helper scripts 
which support our development workflow. Explore more commands with `roxhelp --list-all`.

```
# Change directory to rox root
$ cdrox

# Handy curl shortcut for your StackRox central instance
# Uses https://localhost:8000 by default or ROX_BASE_URL env variable
# Also uses the admin credentials from your last deployment via deploy.sh
$ roxcurl /v1/metadata

# Run quickstyle checks, faster than roxs' "make style"
$ quickstyle

# The workflow repository includes some tools for supporting 
# working with multiple inter-dependent branches.
# Examples:
$ smart-branch <branch-name>    # create new branch
    ... work on branch...
$ smart-rebase                  # rebase from parent branch
    ... continue working on branch...
$ smart-diff                    # check diff relative to parent branch
    ... git push, etc.
```

---

### How to Deploy
Deployment configurations are under the `deploy/` directory, organized
per orchestrator.

The deploy script will:

 1. Launch Central.
 1. Create a cluster configuration and a service identity
 1. Deploy the cluster sensor using that configuration and those credentials

You can set the environment variable `MAIN_IMAGE_TAG` in your shell to
ensure that you get the version you want.
If you check out a commit, the scripts will launch the image corresponding to
that commit by default. The image will be pulled if needed.

Further steps are orchestrator specific.

<details><summary>Kubernetes</summary>

Set your Docker image-pull credentials as `REGISTRY_USERNAME` and
`REGISTRY_PASSWORD`, then run:

```bash
./deploy/k8s/deploy.sh
```
</details>

## Deploying for Customer

<details><summary>Kubernetes</summary>

```
docker run -i --rm stackrox.io/main:<tag> interactive > k8s.zip
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
docker run -i --rm stackrox.io/main:<tag> interactive > openshift.zip
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

Replace the value with the version number you want to release from:
```bash
export RELEASE_BRANCH=2.4.22.x
export MASTER_VERSION=${RELEASE_BRANCH}
export RELEASE_VERSION=2.4.22.0
```

The release branch naming convention should follow <major_version>.<generic_minor_version>.
This is because this branch will become the base for all patch releases of the generic
minor version defined. Above, the branch has a major version of `2.4` and a generic minor
version of `22.x`, which will be used as the basis for `22.0`, `22.1`, `22.2`, etc...

The release version should be the specific version you plan to release. This will be used
when creating the tag later in the process. With each release, we should create at least
1 release candidate to use for testing prior to releasing to customers (Release
Candidate versions should be a combination of the full version number with `-rc.x`
appended to the end: i.e., `2.4.22.0-rc.1`).

By convention, we do not currently use a `v` prefix for release tags (that is,
we push tags like `0.5`, not `v0.5`).

### Prep the release
Proceed with the steps that under the section of the release type you're making:
non-patch or patch.

#### Create a release branch from master
These steps assume that the tip of `origin/master` is what you plan to release
and that **all the builds for that commit have completed successfully**. We will
checkout `origin/master` and create a new release branch from it. We make an
empty commit to `release/${RELEASE_BRANCH}` to diverge from master. This allows
us to start tracking the point of divergence for the release. We will push the
branch to github for use in future builds for that release version. We also will
tag the master branch commit with `${MASTER_VERSION}` (N.B. not `${RELEASE_VERSION}` --
master tag should now look like `2.4.22.x`.

```bash
git checkout master
git fetch
git pull
## Tag master branch to indicate current release 
git tag -a -m "v${MASTER_VERSION}" "${MASTER_VERSION}"
git push origin "${MASTER_VERSION}"
## Create Release Branch
git checkout -b release/${RELEASE_BRANCH}
git commit --allow-empty -m "${RELEASE_BRANCH}"
git push origin release/${RELEASE_BRANCH}
```

#### Patching the Release
```bash
git fetch
git checkout release/${RELEASE_BRANCH}
```

### Pull Fixes into the Release
Then use `git cherry-pick -x ${commit_sha}` to cherry pick commits from `master`
that are going into this patch release. If release requires special changes
(besides cherry picking from `master`), push the release branch and create
(and merge after code review) PR(s) targeting it.

```bash
export RELEASE_COMMIT="$(git rev-parse HEAD)"
echo -e "Preparing to release:\n$(git log -n 1 ${RELEASE_COMMIT})"
```

### Create a Release Candidate

In order to test the release branch in CI you will need to apply a `-rc.x` tag
on the release branch. For example, for `-rc.1`:

```bash
export RC_VERSION=1
git checkout release/${RELEASE_BRANCH}
git tag -a -m "v${RELEASE_VERSION}-rc.${RC_VERSION}" "${RELEASE_VERSION}-rc.${RC_VERSION}"
git tag -ln "${RELEASE_VERSION}-rc.${RC_VERSION}"
git push origin "${RELEASE_VERSION}-rc.${RC_VERSION}"
git push origin release/${RELEASE_BRANCH}
```

When you push the tag to GitHub, CircleCI will start a build and will push
the image to docker.io as `stackrox/main:[your-rc-tag]`,
for example `stackrox/main:2.4.22.0-rc.1`.

### Create a Release

```bash
git checkout release/${RELEASE_BRANCH}
git tag -a -m "v${RELEASE_VERSION}" "${RELEASE_VERSION}"
git tag -ln "${RELEASE_VERSION}"
git push origin "${RELEASE_VERSION}"
git push origin release/${RELEASE_BRANCH}
```

When you push the tag to GitHub, CircleCI will start a build and will push
the image as `stackrox/main:[your-release-tag]`,
for example `stackrox/main:2.4.22.0` and `stackrox.io/main:2.4.22.0`.

### Update JIRA release
*Note: Jira [doesn't have](https://community.atlassian.com/t5/Jira-questions/How-do-I-assign-the-permission-to-create-Versions-to-a/qaq-p/677499)
version / release specific permissions, therefore request Jira admins to assign
to you a "Release Manager" project role (at least temporaly) to perform some of
the Jira actions below.*

<details><summary>Steps to update Jira</summary>

**Important Note**: When doing bulk operations review the lists, that's your
best chance to catch mistakes from the past release cycle or find out that
something unexpected landed in the upcoming release.

  1. Add the version being released to "Fix Version(s)" for completed items that
don't have it ([filter](https://stack-rox.atlassian.net/issues/?filter=15720)).
  1. Add the version being released to "Affected Version(s)" for bugs that have
  this field empty ([filter](https://stack-rox.atlassian.net/issues/?filter=15719)).
  1. Add the version being released to "Affected Version(s)" for all the bugs
  that affect previous release and are still not fixed ([filter](https://stack-rox.atlassian.net/issues/?filter=15728)).
  1. Find the version that is being released [here](https://stack-rox.atlassian.net/projects/ROX?orderField=RANK&selectedItem=com.atlassian.jira.jira-projects-plugin%3Arelease-page&status=released-unreleased),
  review that there are no issues under this version w/o code being merged
  (otherwise it may mean that the release is being blocked, or that you need
  to remove the version being released from their "Fix Version(s)" field, you
  may need to update "Affected Version(s)" as well). Finally mark the version as
released.
  1. Create next version in Jira if it doesn't exist (for non-patch releases
  only), order it properly among other versions.

</details>

### Create Release Notes
Once the GA version of the release has been created, we need to mark the tag as a release
in GitHub.
1. Go the [tags page on GitHub](https://github.com/stackrox/rox/tags).
1. Find the corresponding tag. Click the three-dots menu on the right and
click "Create release".
1. Write release notes based on JIRA issues that
went into the current release ([filter](https://stack-rox.atlassian.net/issues/?jql=project%20%3D%20ROX%20AND%20fixVersion%20%3D%20latestReleasedVersion()%20AND%20resolution%20not%20in%20(%22Won%27t%20Do%22%2C%20%22Won%27t%20Fix%22%2C%20%22Invalid%20Ticket%22%2C%20%22Not%20a%20Bug%22%2C%20Duplicate%2C%20%22Duplicate%20Ticket%22%2C%20%22Cannot%20Reproduce%22))).

### Update solutions offline scripts
* update image tags for main, collector, and monitoring in the [solutions offline scripts](https://github.com/stackrox/solutions/blob/master/offline/create-archive.sh)
* run the `create-archive.sh` script to generate an image bundle
* upload the generated image bundle to the released version directory in [Google storage bucket](https://console.cloud.google.com/storage/browser/sr-roxc/?project=stackrox-hub)

[circleci-badge]: https://circleci.com/gh/stackrox/rox.svg?&style=shield&circle-token=140f88ea9dfd594ff68b71eaf1d4407c4331833d
[circleci-link]:  https://circleci.com/gh/stackrox/workflows/rox/tree/master
[coveralls-badge]: https://coveralls.io/repos/github/stackrox/rox/badge.svg?t=uFuaaq
[coveralls-link]: https://coveralls.io/github/stackrox/rox
