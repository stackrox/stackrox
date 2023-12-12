## Table of Contents

- [StackRox Kubernetes Security Platform](#stackrox-kubernetes-security-platform)
    - [Table of Contents](#table-of-contents)
    - [Community](#community)
    - [Deploying StackRox](#deploying-stackrox)
        - [Quick Installation using Helm](#quick-installation-using-helm)
        - [Manual Installation using Helm](#manual-installation-using-helm)
        - [Installation via Scripts](#installation-via-scripts)
            - [Kubernetes Distributions (EKS, AKS, GKE)](#kubernetes-distributions-eks-aks-gke)
            - [OpenShift](#openshift)
            - [Docker Desktop, Colima, or minikube](#docker-desktop-colima-or-minikube)
        - [Accessing the StackRox User Interface (UI)](#accessing-the-stackrox-user-interface-ui)
    - [Development](#development)
        - [Quickstart](#quickstart)
            - [Build Tooling](#build-tooling)
            - [Clone StackRox](#clone-stackrox)
            - [Local Development](#local-development)
            - [Common Makefile Targets](#common-makefile-targets)
            - [Productivity](#productivity)
            - [GoLand Configuration](#goland-configuration)
            - [Running sql_integration tests](#running-sql_integration-tests)
            - [Debugging](#debugging)
    - [Generating Portable Installers](#generating-portable-installers)
    - [Dependencies and Recommendations for Running StackRox](#dependencies-and-recommendations-for-running-stackrox)

---

# StackRox Kubernetes Security Platform

The StackRox Kubernetes Security Platform performs a risk analysis of the
container environment, delivers visibility and runtime alerts, and provides
recommendations to proactively improve security by hardening the environment.
StackRox integrates with every stage of container lifecycle: build, deploy and
runtime.


The StackRox Kubernetes Security platform is built on the foundation of
the product formerly known as Prevent, which itself was called Mitigate and
Apollo. You may find references to these previous names in code or
documentation.

---

## Community

You can reach out to us through [Slack](https://cloud-native.slack.com/archives/C01TDE3GK0E) (#stackrox).
For alternative ways, stop by our Community Hub [stackrox.io](https://www.stackrox.io/).

For event updates, blogs and other resources follow the StackRox community site at [stackrox.io](https://www.stackrox.io/).

For the StackRox [Code of Conduct](https://www.stackrox.io/code-conduct/).

To [report a vulnerability or bug](https://github.com/stackrox/stackrox/security/policy).

---

## Deploying StackRox

### Quick Installation using Helm

StackRox offers quick installation via Helm Charts. Follow the [Helm Installation Guide](https://helm.sh/docs/intro/install/) to get `helm` CLI on your system.
Then run the helm quick installation script or proceed to section [Manual Installation using Helm](#manual-installation-using-helm) for configuration options.

<details><summary>Install StackRox via Helm Installation Script</summary>

```sh
/bin/bash <(curl -fsSL https://raw.githubusercontent.com/stackrox/stackrox/master/scripts/quick-helm-install.sh)
```
A default deployment of StackRox has certain CPU and memory requests and may fail on small (e.g. development) clusters if sufficient resources are not available. You may use the `--small` command-line option in order to install StackRox on smaller clusters with limited resources. Using this option is not recommended for production deployments.
```sh
/bin/bash <(curl -fsSL https://raw.githubusercontent.com/stackrox/stackrox/master/scripts/quick-helm-install.sh) --small
```
The script adds the StackRox helm repository, generates an admin password, installs stackrox-central-services, creates an init bundle for provisioning stackrox-secured-cluster-services, and finally installs stackrox-secured-cluster-services on the same cluster.

Finally, the script will automatically open the browser and log you into StackRox. A certificate warning may be displayed since the certificate is self-signed. See the [Accessing the StackRox User Interface (UI)](#accessing-the-stackrox-user-interface-ui) section to read more about the warnings. After authenticating you can access the dashboard using <https://localhost:8000/main/dashboard>.

</details>

### Manual Installation using Helm

StackRox offers quick installation via Helm Charts. Follow the [Helm Installation Guide](https://helm.sh/docs/intro/install/) to get the `helm` CLI on your system.

Deploying using Helm consists of 4 steps

1. Add the StackRox repository to Helm
2. Launch **StackRox Central Services** using helm
3. Create a cluster configuration and a service identity (init bundle)
4. Deploy the **StackRox Secured Cluster Services** using that configuration and those credentials (this step can be done multiple times to add more clusters to the StackRox Central Service)

<details><summary>Install StackRox Central Services</summary>

#### Default Central Installation
First, the StackRox Central Services will be added to your Kubernetes cluster. This includes the UI and Scanner. To start, add the [stackrox/helm-charts/opensource](https://github.com/stackrox/helm-charts/tree/main/opensource) repository to Helm.

```sh
helm repo add stackrox https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/
```
To see all available Helm charts in the repo run (you may add the option `--devel` to show non-release builds as well)
```sh
helm search repo stackrox
```
To install stackrox-central-services, you will need a secure password. This password will be needed later for UI login and when creating an init bundle.
```sh
STACKROX_ADMIN_PASSWORD="$(openssl rand -base64 20 | tr -d '/=+')"
```
From here, you can install stackrox-central-services to get Central and Scanner components deployed on your cluster. Note that you need only one deployed instance of stackrox-central-services even if you plan to secure multiple clusters.
```sh
helm upgrade --install -n stackrox --create-namespace stackrox-central-services \
  stackrox/stackrox-central-services \
  --set central.adminPassword.value="${STACKROX_ADMIN_PASSWORD}"
```

#### Install Central in Clusters With Limited Resources

If you're deploying StackRox on nodes with limited resources such as a local development cluster, run the following command to reduce StackRox resource requirements. Keep in mind that these reduced resource settings are not suited for a production setup.

```sh
helm upgrade -n stackrox stackrox-central-services stackrox/stackrox-central-services \
  --set central.resources.requests.memory=1Gi \
  --set central.resources.requests.cpu=1 \
  --set central.resources.limits.memory=4Gi \
  --set central.resources.limits.cpu=1 \
  --set central.db.resources.requests.memory=1Gi \
  --set central.db.resources.requests.cpu=500m \
  --set central.db.resources.limits.memory=4Gi \
  --set central.db.resources.limits.cpu=1 \
  --set scanner.autoscaling.disable=true \
  --set scanner.replicas=1 \
  --set scanner.resources.requests.memory=500Mi \
  --set scanner.resources.requests.cpu=500m \
  --set scanner.resources.limits.memory=2500Mi \
  --set scanner.resources.limits.cpu=2000m
```

</details>

<details><summary>Install StackRox Secured Cluster Services</summary>

#### Default Secured Cluster Installation
Next, the secured cluster component will need to be deployed to collect information on from the Kubernetes nodes.

Generate an init bundle containing initialization secrets. The init bundle will be saved in `stackrox-init-bundle.yaml`, and you will use it to provision secured clusters as shown below.
```sh
kubectl -n stackrox exec deploy/central -- roxctl --insecure-skip-tls-verify \
  --password "${STACKROX_ADMIN_PASSWORD}" \
  central init-bundles generate stackrox-init-bundle --output - > stackrox-init-bundle.yaml
```
Set a meaningful cluster name for your secured cluster in the `CLUSTER_NAME` shell variable. The cluster will be identified by this name in the clusters list of the StackRox UI.
```sh
CLUSTER_NAME="my-secured-cluster"
```
Then install stackrox-secured-cluster-services (with the init bundle you generated earlier) using this command:
```sh
helm upgrade --install --create-namespace -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  -f simon-test-cluster-init-bundle.yaml \
  --set clusterName="$CLUSTER_NAME" \
  --set centralEndpoint="central.stackrox.svc:443"
```
When deploying stackrox-secured-cluster-services on a different cluster than the one where stackrox-central-services is deployed, you will also need to specify the endpoint (address and port number) of Central via `--set centralEndpoint=<endpoint_of_central_service>` command-line argument.

#### Install Secured Cluster with Limited Resources
When deploying StackRox Secured Cluster Services on a small node, you can install with additional options. This should reduce stackrox-secured-cluster-services resource requirements. Keep in mind that these reduced resource settings are not recommended for a production setup.

```sh
helm install -n stackrox stackrox-secured-cluster-services stackrox/stackrox-secured-cluster-services \
  -f stackrox-init-bundle.yaml \
  --set clusterName="$CLUSTER_NAME" \
  --set centralEndpoint="central.stackrox.svc:443" \
  --set sensor.resources.requests.memory=500Mi \
  --set sensor.resources.requests.cpu=500m \
  --set sensor.resources.limits.memory=500Mi \
  --set sensor.resources.limits.cpu=500m
```
</details>

<details>
<summary>Additional information about Helm charts</summary>

To further customize your Helm installation consult these documents:

* <https://docs.openshift.com/acs/installing/installing_other/install-central-other.html#install-using-helm-customizations-other>
* <https://docs.openshift.com/acs/installing/installing_other/install-secured-cluster-other.html#configure-secured-cluster-services-helm-chart-customizations-other>

</details>

### Installation via Scripts

The `deploy` script will:

1. Launch **StackRox Central Services**
2. Create a cluster configuration and a service identity
3. Deploy the **StackRox Secured Cluster Services** using that configuration and those credentials

You can set the environment variable `MAIN_IMAGE_TAG` in your shell to
ensure that you get the version you want.

If you check out a commit, the scripts will launch the image corresponding to that commit by default. The image will be pulled if needed.

Further steps are orchestrator specific.

#### Kubernetes Distributions (EKS, AKS, GKE)

<details><summary>Click to expand</summary>

Follow the guide below to quickly deploy a specific version of StackRox to your Kubernetes cluster in the `stackrox` namespace. If you want to install a specific version, make sure to define/set it in `MAIN_IMAGE_TAG`, otherwise it will install the latest nightly build.

Run the following in your working directory of choice:

```
git clone git@github.com:stackrox/stackrox.git
cd stackrox
MAIN_IMAGE_TAG=VERSION_TO_USE ./deploy/deploy.sh
```

After a few minutes, all resources should be deployed.

**Credentials for the 'admin' user can be found in the `./deploy/k8s/central-deploy/password` file.**

**Note:** While the password file is stored in plaintext on your local filesystem, the Kubernetes Secret StackRox uses is encrypted, and you will not be able to alter the secret at runtime. If you lose the password, you will have to redeploy central.

</details>

#### OpenShift

<details><summary>Click to Expand</summary>

Before deploying on OpenShift, ensure that you have the [oc - OpenShift Command Line](https://github.com/openshift/oc) installed.

Follow the guide below to quickly deploy a specific version of StackRox to your OpenShift cluster in the `stackrox` namespace. Make sure to add the most recent tag to the `MAIN_IMAGE_TAG` variable.

Run the following in your working directory of choice:

```
git clone git@github.com:stackrox/stackrox.git
cd stackrox
MAIN_IMAGE_TAG=VERSION_TO_USE ./deploy/deploy.sh
```

After a few minutes, all resources should be deployed. The process will complete with this message.

**Credentials for the 'admin' user can be found in the `./deploy/openshift/central-deploy/password` file.**

**Note:** While the password file is stored in plaintext on your local filesystem, the Kubernetes Secret StackRox uses is encrypted, and you will not be able to alter the secret at runtime. If you loose the password, you will have to redeploy central.

</details>

#### Docker Desktop, Colima, or minikube

<details><summary>Click to Expand</summary>

Run the following in your working directory of choice:

```
git clone git@github.com:stackrox/stackrox.git
cd stackrox
MAIN_IMAGE_TAG=latest ./deploy/deploy-local.sh
```

After a few minutes, all resources should be deployed.

**Credentials for the 'admin' user can be found in the `./deploy/k8s/deploy-local/password` file.**

</details>

### Accessing the StackRox User Interface (UI)

<details><summary>Click to expand</summary>

After the deployment has completed (Helm or script install) a port-forward should exist, so you can connect to https://localhost:8000/. Run the following

```sh
kubectl port-forward -n 'stackrox' svc/central "8000:443"
```

Then go to https://localhost:8000/ in your web browser.

**Username** = The default user is `admin`

**Password (Helm)**   = The password is in `$STACKROX_ADMIN_PASSWORD` after a manual installation, or printed at the end of the quick installation script.

**Password (Script)** = The password will be located in the `/deploy/<orchestrator>/central-deploy/password.txt` folder for the script install.

</details>

---
## Development

- **UI Dev Docs**: Refer to [ui/README.md](./ui/README.md)

- **E2E Dev Docs**: Refer to [qa-tests-backend/README.md](./qa-tests-backend/README.md)

- **Pull request guidelines**: [contributing.md](./github/contributing.md)

- **Go coding style guide**: [go-coding-style.md](./github/go-coding-style.md) 

### Quickstart

#### Build Tooling

The following tools are necessary to test code and build image(s):

<details><summary>Click to expand</summary>

* [Make](https://www.gnu.org/software/make/)
* [Go](https://golang.org/dl/)
  * Get the version specified in [EXPECTED_GO_VERSION](./EXPECTED_GO_VERSION).
* Various Go linters and RocksDB dependencies that can be installed using `make reinstall-dev-tools`.
* UI build tooling as specified in [ui/README.md](ui/README.md#Build-Tooling).
* [Docker](https://docs.docker.com/get-docker/)
  * Note: Docker Desktop now requires a paid subscription for larger, enterprise companies.
  * Some StackRox devs recommend [Colima](https://github.com/abiosoft/colima)
* [RocksDB](https://rocksdb.org/)
  * Follow [Mac](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-darwin.md#install-rocksdb) or [Linux](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-linux.md#install-rocksdb) guide)
* [Xcode](https://developer.apple.com/xcode/) command line tools (macOS only)
* [Bats](https://github.com/sstephenson/bats) is used to run certain shell tests.
  You can obtain it with `brew install bats` or `npm install -g bats`.
* [oc](https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/stable/) OpenShift cli tool
* [shellcheck](https://github.com/koalaman/shellcheck#installing) for shell scripts linting.

**Xcode - macOS Only**

Usually you would have these already installed by brew.
However, if you get an error when building the golang x/tools,
try first making sure the EULA is agreed by:

1. starting Xcode
2. building a new blank app project
3. starting the blank project app in the emulator
4. close both the emulator and the Xcode, then
5. run the following commands:

 ```bash
 xcode-select --install
 sudo xcode-select --switch /Library/Developer/CommandLineTools # Enable command line tools
 sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
 ```

For more info, see <https://github.com/nodejs/node-gyp/issues/569>

</details>

#### Clone StackRox
<details><summary>Click to expand</summary>

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
</details>

#### Local Development

<details><summary>Click to expand</summary>

To sweeten your experience, install [the workflow scripts](#productivity) beforehand.

First install RocksDB. Follow [Mac](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-darwin.md#install-rocksdb) or [Linux](https://github.com/stackrox/dev-docs/blob/main/docs/getting-started/getting-started-linux.md#install-rocksdb) guidelines
```bash
$ cd $GOPATH/src/github.com/stackrox/stackrox
$ make install-dev-tools
$ make image
```

Now, you need to bring up a Kubernetes cluster *yourself* before proceeding.
Development can either happen in GCP or locally with
[Docker Desktop](https://docs.docker.com/desktop/kubernetes/), [Colima](https://github.com/abiosoft/colima#kubernetes), [minikube](https://minikube.sigs.k8s.io/docs/start/).
Note that Docker Desktop and Colima are more suited for macOS development, because the cluster will have access to images built with `make image` locally without additional configuration. Also, Collector has better support for these than minikube where drivers may not be available.

```bash
# To keep the StackRox Central's RocksDB state between restarts, set:
$ export STORAGE=pvc

# To save time on rebuilds by skipping UI builds, set:
$ export SKIP_UI_BUILD=1

# To save time on rebuilds by skipping CLI builds, set:
$ export SKIP_CLI_BUILD=1

# When you deploy locally make sure your kube context points to the desired kubernetes cluster,
# for example Docker Desktop.
# To check the current context you can call a workflow script:
$ roxkubectx

# To deploy locally, call:
$ ./deploy/deploy-local.sh

# Now you can access StackRox dashboard at https://localhost:8000
# or simply call another workflow script:
$ logmein
```

See [Installation via Scripts](#installation-via-scripts) for further reading. To read more about the environment variables, consult
[deploy/README.md](https://github.com/stackrox/stackrox/blob/master/deploy/README.md#env-variables).

</details>

#### Common Makefile Targets

<details><summary>Click to expand</summary>

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
</details>

#### Productivity

<details><summary>Click to expand</summary>

The [workflow repository](https://github.com/stackrox/workflow) contains some helper scripts
which support our development workflow. Explore more commands with `roxhelp --list-all`.

```bash
# Change directory to rox root
$ cdrox

# Handy curl shortcut for your StackRox central instance
# Uses https://localhost:8000 by default or ROX_BASE_URL env variable
# Also uses the admin credentials from your last deployment via deploy.sh
$ roxcurl /v1/metadata

# Run quickstyle checks, faster than stackrox's "make style"
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

</details>

#### GoLand Configuration

<details><summary>Click to expand</summary>

If you're using GoLand for development, the following can help improve the experience.


Make sure the `Protocol Buffers` plugin is installed. The plugin comes installed by default in GoLand.
If it isn't, use `Help | Find Action...`, type `Plugins` and hit enter, then switch to `Marketplace`, type its name and install the plugin.

This plugin does not know where to look for `.proto` imports by default in GoLand therefore you need to explicitly
configure paths for this plugin. See <https://github.com/jvolkman/intellij-protobuf-editor#path-settings>.

* Go to `GoLand | Preferences | Languages & Frameworks | Protocol Buffers`.
* Uncheck `Configure automatically`.
* Click on `+` button, navigate and select `./proto` directory in the root of the repo.
* Optionally, also add `$HOME/go/pkg/mod/github.com/gogo/googleapis@1.1.0`
  and `$HOME/go/pkg/mod/github.com/gogo/protobuf@v1.3.1/`.
* To verify: use menu `Navigate | File...` type any `.proto` file name, e.g. `alert_service.proto`, and check that all
  import strings are shown green, not red.

</details>

#### Running sql_integration tests

<details><summary>Click to expand</summary>

Go tests annotated with `//go:build sql_integration` require a PostgreSQL server listening on port 5432.
Due to how authentication is set up in code, it is the easiest to start Postgres in a container like this:

```bash
$ docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:13
```

With that running in the background, `sql_integration` tests can be triggered from IDE or command-line.

</details>

#### Debugging

<details><summary>Click to expand</summary>

**Kubernetes debugger setup**

With GoLand, you can naturally use breakpoints and the debugger when running unit tests in IDE.

If you would like to debug local or even remote deployment, follow the procedure below.

1. Create debug build locally by exporting `DEBUG_BUILD=yes`:
   ```bash
   $ DEBUG_BUILD=yes make image
   ```
   Alternatively, debug build will also be created when the branch name contains `-debug` substring. This works locally with `make image` and in CI.
2. Deploy the image using instructions from this README file. Works both with `deploy-local.sh` and `deploy.sh`.
3. Start the debugger (and port forwarding) in the target pod using `roxdebug` command from `workflow` repo.
   ```bash
   # For Central
   $ roxdebug
   # For Sensor
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

## Generating Portable Installers

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


## Dependencies and Recommendations for Running StackRox

<details><summary>Click to Expand</summary>

The following information has been gathered to help with the installation and operation of the open source StackRox project. These recommendations were developed for the [Red Hat Advanced Cluster Security for Kubernetes](https://www.redhat.com/en/resources/advanced-cluster-security-for-kubernetes-datasheet) product and have not been tested with the upstream StackRox project.

**Recommended Kubernetes Distributions**

The Kubernetes Platforms that StackRox has been deployed onto with minimal issues are listed below.

- Red Hat OpenShift Dedicated (OSD)
- Azure Red Hat OpenShift (ARO)
- Red Hat OpenShift Service on AWS (ROSA)
- Amazon Elastic Kubernetes Service (EKS)
- Google Kubernetes Engine (GKE)
- Microsoft Azure Kubernetes Service (AKS)

If you deploy into a Kubernetes distribution other than the ones listed above you may encounter issues.

**Recommended Operating Systems**

StackRox is known to work on the recent versions of the following operating systems.

- Ubuntu
- Debian
- Red Hat Enterprise Linux (RHEL)
- CentOS
- Fedora CoreOS
- Flatcar Container Linux
- Google COS
- Amazon Linux
- Garden Linux

**Recommended Web Browsers**

The following table lists the browsers that can view the StackRox web user interface.

- Google Chrome 88.0 (64-bit)
- Microsoft Internet Explorer Edge
    - Version 44 and later (Windows)
    - Version 81 (Official build) (64-bit)
- Safari on MacOS (Mojave) - Version 14.0
- Mozilla Firefox Version 82.0.2 (64-bit)

</details>

---

