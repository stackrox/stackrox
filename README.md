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
 * [Go](https://golang.org/dl/) 1.12.
 * Various Go linters that can be installed using `make dev`.
 * [Node.js](https://nodejs.org/en/) `8.12.0` or higher (it's highly recommended to use an LTS version)
 If you're managing multiple versions of Node.js on your machine, consider using [nvm](https://github.com/creationix/nvm))
 * [Yarn](https://yarnpkg.com/en/)

### How to Build
```bash
make main-image
```

This will create `stackrox/main` with a tag defined by `make tag`.

### Possible OS/X complications:
If you are on OS/X and get an error when building the golang x/tools,
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

### Test the base configuration
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
and that **all the builds for that commit have completed successfully**.

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

Then use `git cherry-pick -x ${commit_sha}` to cherry pick commits from `master`
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
the image as `stackrox/main:[your-release-tag]`,
for example `stackrox/main:1.0` and `stackrox.io/main:1.0`.

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
1. Go the [tags page on GitHub](https://github.com/stackrox/rox/tags).
1. Find the corresponding tag. Click the three-dots menu on the right and
click "Create release".
1. Write release notes based on JIRA issues that
went into the current release ([filter](https://stack-rox.atlassian.net/issues/?jql=project%20%3D%20ROX%20AND%20fixVersion%20%3D%20latestReleasedVersion()%20AND%20resolution%20not%20in%20(%22Won%27t%20Do%22%2C%20%22Won%27t%20Fix%22%2C%20%22Invalid%20Ticket%22%2C%20%22Not%20a%20Bug%22%2C%20Duplicate%2C%20%22Duplicate%20Ticket%22%2C%20%22Cannot%20Reproduce%22))).
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
