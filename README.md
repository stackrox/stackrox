# Misha presenting code changes

Lorem ipsum.


[![CircleCI][circleci-badge]][circleci-link]

# StackRox Kubernetes Security Platform

The StackRox Kubernetes Security Platform performs a risk analysis of the
container environment, delivers visibility and runtime alerts, and provides
recommendations to proactively improve security by hardening the environment.
StackRox integrates with every stage of container lifecycle: build, deploy and
runtime.

Note: the StackRox Kubernetes Security platform is built on the foundation of 
the product formerly known as Prevent, which itself was called Mitigate and
Apollo. You may find references to these previous names in code or
documentation.

## Community
You can reach out to us through [Slack](https://cloud-native.slack.com/archives/C01TDE3GK0E) (#stackrox).  
For alternative ways, stop by our Community Hub [stackrox.io](https://www.stackrox.io/).

## Table of contents

* [Quick Installation via Helm](#quick-installation-via-helm)
    + [First use](#first-use)
* [Manual Deployment](#manual-deployment)
* [Development](#development)
    + [Quickstart](#quickstart)
      - [Build Tooling](#build-tooling)
      - [Clone StackRox](#clone-stackrox)
      - [Local development](#local-development)
      - [Common Makefile Targets](#common-makefile-targets)
      - [Productivity](#productivity)
      - [GoLand configuration](#goland-configuration)
      - [Debugging](#debugging)
    + [How to Deploy](#how-to-deploy)
* [Generating portable installers](#generating-portable-installers)

## Quick Installation via Helm

StackRox offers quick installation via Helm Charts. Follow the [Helm Installation Guide](https://helm.sh/docs/intro/install/) to get `helm` CLI on your system.
First, add the [stackrox/helm-charts/opensource](https://github.com/stackrox/helm-charts/tree/main/opensource) repository to Helm.
```sh
helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/
```
To see all available Helm charts in the repo run (you may add the option `--devel` to show non-release builds as well)
```sh
helm search repo stackrox
```
In order to install stackrox-central-services you will need a secure password. This password will be needed later when creating an init bundle.
```sh
openssl rand -base64 20 | tr -d '/=+' > stackrox-admin-password.txt
```

From here you can install stackrox-central-services to get Central and Scanner components deployed on your cluster. Note that you need only one deployed instance of stackrox-central-services even if you plan to secure multiple clusters.
```sh
helm install -n stackrox --create-namespace stackrox-central-services stackrox/stackrox-central-services --set central.adminPassword.value="$(cat stackrox-admin-password.txt)"
```

If you're deploying StackRox on a small node like a local development cluster, run the following command to reduce StackRox resource requirements. Keep in mind that these reduced resource settings are not suited for a production setup.
```sh
helm upgrade -n stackrox stackrox-central-services stackrox/stackrox-central-services \
  --set central.resources.requests.memory=1Gi \
  --set central.resources.requests.cpu=1 \
  --set central.resources.limits.memory=4Gi \
  --set central.resources.limits.cpu=1 \
  --set scanner.autoscaling.disable=true \
  --set scanner.replicas=1 \
  --set scanner.resources.requests.memory=500Mi \
  --set scanner.resources.requests.cpu=500m \
  --set scanner.resources.limits.memory=2500Mi \
  --set scanner.resources.limits.cpu=2000m
```
To create a secured cluster, you first need to generate an init bundle containing initialization secrets. The init bundle will be saved in `stackrox-init-bundle.yaml`, and you will use it to provision secured clusters as shown below.
```sh
kubectl -n stackrox exec deploy/central -- roxctl --insecure-skip-tls-verify \
  --password "$(cat stackrox-admin-password.txt)" \
  central init-bundles generate stackrox-init-bundle --output - > stackrox-init-bundle.yaml
```
Set a meaningful cluster name for your secured cluster in the `CLUSTER_NAME` environment variable. The cluster will be identified by this name in the clusters list of the StackRox UI.
```sh
CLUSTER_NAME="my-secured-cluster"
```
Then install stackrox-secured-cluster-services (with the init bundle you generated earlier) using this command:
```sh
helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  -f stackrox-init-bundle.yaml \
  --set clusterName="$CLUSTER_NAME"
```
When deploying stackrox-secured-cluster-services on a different cluster than the one where stackrox-central-services are deployed, you will also need to specify the endpoint (address and port number) of Central via `--set centralEndpoint=<endpoint_of_central_service>` command-line argument.

When deploying StackRox on a small node, you can install with additional options. This should reduce stackrox-secured-cluster-services resource requirements. Keep in mind that these reduced resource settings are not recommended for a production setup.
```sh
helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  -f stackrox-init-bundle.yaml \
  --set clusterName="$CLUSTER_NAME" \
  --set sensor.resources.requests.memory=500Mi \
  --set sensor.resources.requests.cpu=500m \
  --set sensor.resources.limits.memory=500Mi \
  --set sensor.resources.limits.cpu=500m
```

To further customize your Helm installation consult these documents:
* <https://docs.openshift.com/acs/installing/installing_helm/install-helm-quick.html>
* <https://docs.openshift.com/acs/installing/installing_helm/install-helm-customization.html>

### First use

Follow these steps to get to StackRox UI

1.Setup port forward to central:
```
kubectl -n stackrox port-forward deploy/central 8000:8443
```

2.Open <https://localhost:8000> in your browser.

3.If a certificate warning is displayed, you can accept the warnings to install the default self-signed certificate. To configure custom certificates, see the section on "Adding Custom Certificates" in <https://docs.openshift.com/acs/configuration/add-custom-certificates.html>.

4.Log in as `admin` using the password in `stackrox-admin-password.txt`

## Manual Deployment

To manually deploy the latest development version of StackRox to your Kubernetes
cluster in the stackrox namespace:

```
git clone git@github.com:stackrox/stackrox.git
cd stackrox
MAIN_IMAGE_TAG=latest ./deploy/k8s/deploy.sh
```

If you are using docker for desktop or minikube use
`./deploy/k8s/deploy-local.sh`. And for openshift:
`./deploy/openshift/deploy.sh`.

When the deployment has completed a port-forward should exist so you can connect
to https://localhost:8000/. Credentials for the 'admin' user can be found in
`./deploy/k8s/central-deploy/password`
(`deploy/openshift/central-deploy/password` in the OpenShift case).

## Development

**UI Dev Docs**: Refer to [ui/README.md](./ui/README.md)

**E2E Dev Docs**: Refer to [qa-tests-backend/README.md](./qa-tests-backend/README.md)

### Quickstart

#### Build Tooling

The following tools are necessary to test code and build image(s):

* [Make](https://www.gnu.org/software/make/)
* [Go](https://golang.org/dl/)
  * Get the version specified in [EXPECTED_GO_VERSION](./EXPECTED_GO_VERSION).
* Various Go linters and RocksDB dependencies that can be installed using `make reinstall-dev-tools`.
* UI build tooling as specified in [ui/README.md](ui/README.md#Build-Tooling).
* Docker (make sure you `docker login` to your company [DockerHub account](https://hub.docker.com/settings/security))
* RocksDB (follow [Mac](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-darwin.md#install-rocksdb) or [Linux](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-linux.md#install-rocksdb) guide)
* Xcode command line tools (macOS only)
* [Bats](https://github.com/sstephenson/bats) is used to run certain shell tests.
  You can obtain it with `brew install bats` or `npm install -g bats`.
* [oc OpenShift](https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/stable/) cli tool

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

 ```bash
 xcode-select --install
 sudo xcode-select --switch /Library/Developer/CommandLineTools # Enable command line tools
 sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
 ```

 For more info, see <https://github.com/nodejs/node-gyp/issues/569>
 </details>

#### Clone StackRox

```bash
# Create a GOPATH: this is the location of your Go "workspace".
# (Note that it is not – and must not – be the same as the path Go is installed to.)
# The default is to have it in ~/go/, or ~/development, but anything you prefer goes.
# Whatever you decide, create the directory, set GOPATH, and update PATH:
export GOPATH=$HOME/go # Change this if you choose to use a different workspace.
export PATH=$PATH:$GOPATH/bin
# You probably want to permanently set these by adding the following commands to your shell
# configuration (e.g. ~/.bash_profile)

cd $GOPATH
mkdir -p bin pkg
mkdir -p src/github.com/stackrox
cd src/github.com/stackrox
git clone git@github.com:stackrox/stackrox.git
```

#### Local development

To sweeten your experience, install [the workflow scripts](#productivity) beforehand.

First install RocksDB. Follow [Mac](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-darwin.md#install-rocksdb) or [Linux](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-linux.md#install-rocksdb) guidelines
```bash
$ cd $GOPATH/src/github.com/stackrox/stackrox
$ make install-dev-tools
$ make image
```

Now, you need to bring up a Kubernetes cluster *yourself* before proceeding.
Development can either happen in GCP or locally with
[Docker Desktop](https://docs.docker.com/docker-for-mac/#kubernetes) or [Minikube](https://minikube.sigs.k8s.io/docs/start/).
Note that Docker Desktop is more suited for macOS development, because the cluster will have access to images built with `make image` locally without additional configuration. Also, the collector has better support for Docker Desktop than Minikube where drivers may not be available.

```bash
# To keep the StackRox central's rocksdb state between restarts, set:
$ export STORAGE=pvc

# To save time on rebuilds by skipping UI builds, set:
$ export SKIP_UI_BUILD=1

# When you deploy locally make sure your kube context points to the desired Kubernetes cluster,
# for example Docker Desktop.
# To check the current context you can call a workflow script:
$ roxkubectx

# To deploy locally, call:
$ ./deploy/k8s/deploy-local.sh

# Now you can access StackRox dashboard at https://localhost:8000
# or simply call another workflow script:
$ logmein
```

See the [deployment guide](#how-to-deploy) for further reading. To read more about the environment variables see the
[deploy/README.md](https://github.com/stackrox/stackrox/blob/master/deploy/README.md#env-variables).

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
```

#### Productivity

The [workflow repository](https://github.com/stackrox/workflow) contains some helper scripts
which support our development workflow. Explore more commands with `roxhelp --list-all`.

```bash
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

#### GoLand configuration

If you're using GoLand for development, the following can help improve the experience.

Make sure the `Protocol Buffers` plugin is installed. The plugin comes installed by default in GoLand.  
If it isn't, use `Help | Find Action...`, type `Plugins` and hit enter, then switch to `Marketplace`, type its name and install the plugin.  
This plugin does not know where to look for `.proto` imports by default in GoLand therefore you need to explicitly
configure paths for this plugin.

* Go to `File | Settings | Languages & Frameworks | Protocol Buffers`.
* Uncheck `Configure automatically`.
* Click on `+` button, navigate and select `./proto` directory in the root of the repo.
* Optionally, also add `$HOME/go/pkg/mod/github.com/gogo/googleapis@1.1.0`
  and `$HOME/go/pkg/mod/github.com/gogo/protobuf@v1.3.1/`.
* To verify: use menu `Navigate | File...` type any `.proto` file name, e.g. `alert_service.proto`, and check that all
  import strings are shown green, not red.

#### Debugging

With GoLand, you can naturally use breakpoints and debugger when running unit tests in IDE.  
If you would like to debug local or even remote deployment, follow the procedure below.

<details><summary>Kubernetes debugger setup</summary>

 1. Create debug build locally by exporting `DEBUG_BUILD=yes`:
    ```bash
    $ DEBUG_BUILD=yes make image
    ```
    Alternatively, debug build will also be created when the branch name contains `-debug` substring. This works locally with `make image` and in CircleCI.
 2. Deploy the image using instructions from this README file. Works both with `deploy-local.sh` and `deploy.sh`.
 3. Start the debugger (and port forwarding) in the target pod using `roxdebug` command from `workflow` repo.
    ```bash
    # For central
    $ roxdebug
    # For sensor
    $ roxdebug deploy/sensor
    # See usage help
    $ roxdebug --help
    ```
 4. Configure GoLand for remote debugging (should be done only once):
    1. Open `Run | Edit Configurations …`, click on the `+` icon to add new configuration, choose `Go Remote` template.
    2. Choose `Host:` `localhost` and `Port:` `40000`. Give this configuration some name.
    3. Select `On disconnect:` `Leave it running` (this prevents GoLand forgetting breakpoints on reconnect).
 5. Attach GoLand to debugging port: select `Run | Debug…` and choose configuration you've created.  
    If all done right, you should see `Connected` message in the `Debug | Debugger | Variables` window at the lower part
    of the screen.
 6. Set some code breakpoints, trigger corresponding actions and happy debugging!

See [Debugging go code running in Kubernetes](https://github.com/stackrox/dev-docs/blob/main/docs/knowledge-base/%5BBE%5D%20Debugging-go-code-running-in-Kubernetes.md) for
more info.
</details>

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

```bash
./deploy/k8s/deploy.sh
```

To avoid typing in docker registry username and password each time you deploy,
set credentials in `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` environment variables,
or, more securely, configure credentials in docker credentials store for your OS as
suggested [here](https://docs.docker.com/engine/reference/commandline/login/#credentials-store).
</details>

<details><summary>Openshift</summary>

Before deploying on Openshift, ensure that you have the [oc - OpenShift Command Line](https://github.com/openshift/oc)
installed.

On MacOS, you can install it via the following command:
```bash
brew install openshift-cli
```

Afterwards, deploy `rox`:
```bash
# Automatically creates a OS route for exposing all relevant services
export LOAD_BALANCER=route

./deploy/openshift/deploy.sh
```

To avoid typing in docker registry username and password each time you deploy,
set credentials in `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` environment variables,
or, more securely, configure credentials in docker credentials store for your OS as
suggested [here](https://docs.docker.com/engine/reference/commandline/login/#credentials-store).
</details>

## Generating portable installers

<details><summary>Kubernetes</summary>

```bash
docker run -i --rm stackrox.io/main:<tag> interactive > k8s.zip
```

This will run you through an installer and generate a `k8s.zip` file.

```bash
unzip k8s.zip -d k8s
```

```bash
bash k8s/central.sh
```

Now Central has been deployed. Use the UI to deploy Sensor.

</details>

<details><summary>OpenShift</summary>

Note: If using a host mount, you need to allow the container to access it by using  
`sudo chcon -Rt svirt_sandbox_file_t <full volume path>`

Take the image-setup.sh script from this repo and run it to do the pull/push to
local OpenShift registry. This is a prerequisite for every new cluster.

```bash
bash image-setup.sh
```

```bash
docker run -i --rm stackrox.io/main:<tag> interactive > openshift.zip
```

This will run you through an installer and generate a `openshift.zip` file.

```bash
unzip openshift.zip -d openshift
```

```bash
bash openshift/central.sh
```
</details>


[circleci-badge]: https://circleci.com/gh/stackrox/stackrox.svg?&style=shield&circle-token=eb5a0b87a6253b4c060d011bbfed4a2f1f516746
[circleci-link]:  https://circleci.com/gh/stackrox/workflows/stackrox/tree/master
