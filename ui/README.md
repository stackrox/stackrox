# StackRox Web UI Monorepo

This directory is managed as a monorepo that is composed of several NPM
packages. The primary sub-package is [apps/platform](./apps/platform/README.md)
with the StackRox Kubernetes Security Platform Web UI.

## Monorepo Structure and Principles

### Root

The root directory and its `package.json` file serves as an entry point for any
interactions with the applications and packages. Any "outsider" tool isn't
expected to know the details of monorepo structure and packages inside it. E.g.
CI only uses `Makefile` and looks for artifacts (like test results) in the root
directory, never looking into `packages` / `apps` subdirectories.

### Packages

Packages live in the `packages` directory. It's implied that every package is
published to the
[private NPM registry](https://github.com/stackrox/rox/packages) from where it
can be consumed by other StackRox projects.

While working on a particular package, treat it as an independent NPM package,
e.g. it shouldn't assume that any of the dependencies will be provided by the
monorepo root package.

#### Adding a New Package

There is a contract that root package has with the monorepo NPM packages. This
contract defined through the scripts that NPM packages should have defined in
their `package.json` files:

-   `clean` - can be used to ensure a clean state before executing another
    script that might produce transient artifacts like build files etc. Can be
    omitted if running consecutive invocation of any scripts will never collide.
-   `lint` - run lint and static code analysis, should fail in case of errors.
-   `build` - run production optimized build.
-   `start` - build dev optimized version and possibly hot-rebuild on changes to
    the sources by listening to them.
-   `test` - run unit / integration tests. If env var `TEST_RESULTS_OUTPUT_DIR`
    defined, package should place JUnit reports into
    `${TEST_RESULTS_OUTPUT_DIR}/reports` dir, and any artifacts to be stored by
    CI into `${TEST_RESULTS_OUTPUT_DIR}/artifacts`.
-   `test-e2e` - run end-to-end tests. If env var `TEST_RESULTS_OUTPUT_DIR`
    defined, package should place JUnit reports into
    `${TEST_RESULTS_OUTPUT_DIR}/reports` dir, and any artifacts to be stored by
    CI into `${TEST_RESULTS_OUTPUT_DIR}/artifacts`.

Note that not having a particular script defined will simply mean that this
package will be skipped. E.g. if the package doesn't have `test` script, then
when CI runs unit tests step for this monorepo, no tests will be executed for
that package.

_TBD (additional scripts will be added for publishing etc. when least one
package is there)_

#### Adding a Dependency to Another Package

_TBD (will be filled once at least one package is there)_

### Applications

Applications live in the 'apps' directory. They're never published to any
registry. They serve as entry points for the builds to produce a final set of
static assets to be included into the Docker image etc.

Same as packages, they still should be treated as independent NPM packages w/o
an implicit dependencies on the packages provided by the monorepo root.

#### Adding a New Application

It's an unlikely event a new application is needed, like if the product has two
offerings with one being a functional slimmed down version of the main one.

In this case ensure the following fields are correctly set in `package.json`:

```
"name": "@stackrox/{app-name}",
"version": "0.0.0",
"private": true,
"repository": {
    "type": "git",
    "url": "https://github.com/stackrox/rox.git",
    "directory": "ui/apps/{app-dir-name}"
},
"license": "UNLICENSED"
```

#### Adding a Dependency to a Package

The same considerations are applied as with a package (see
[above](#adding-a-dependency-to-another-package)).

## Development

If you are developing only StackRox UI, then you don't have to install all the
build tooling described in the parent [README.md](../README.md). Instead, follow
the instructions below.

### Build Tooling

-   [Docker](https://www.docker.com/)
-   [Node.js](https://nodejs.org/en/) `12.18.3 LTS` or higher (it's highly
    recommended to use an LTS version, if you're managing multiple versions of
    Node.js on your machine, consider using
    [nvm](https://github.com/creationix/nvm))
-   [Yarn](https://yarnpkg.com/en/) v1.x

### Dev Env Setup

_Before starting, make sure you have the above tools installed on your machine
and you've run `yarn install` to download dependencies._

The front end development environment consists of a CRA-provided dev server to
serve static UI assets and deployed StackRox Docker containers that provide
backend API.

Set up your environment as follows:

#### Using Local StackRox Deployment and Docker for Mac

_Note: Similar instructions apply when using
[Minikube](https://kubernetes.io/docs/setup/minikube/)._

1. **Docker for Mac** - Make sure you have Kubernetes enabled in your Docker for
   Mac and `kubectl` is pointing to `docker-desktop` (see
   [docker docs](https://docs.docker.com/docker-for-mac/#kubernetes)).

1. **Deploy** - Run `yarn deploy-local` (wraps `../deploy/k8s/deploy-local.sh`)
   to deploy the StackRox k8s app. Make sure that your git working directory is
   clean and that the branch that you're on has a corresponding tag from CI (see
   Roxbot comment for a PR branch). Alternatively, you can specify the image tag
   you want to deploy by setting the `MAIN_IMAGE_TAG` env var. If
   `yarn deploy-local` fails, see this
   [Knowledge Base article for debugging instructions](https://stack-rox.atlassian.net/wiki/spaces/ENGKB/pages/883229760/Troubleshooting+local+deployment+of+StackRox).

1. **Start** - Start your local dev server by running `yarn start`.

_Note: to redeploy a newer version of StackRox, delete existing app using
`teardown` script from the [workflow](https://github.com/stackrox/workflow/)
repo, and repeat the steps above._

#### Using a Remote StackRox Deployment

1. **Provision back end cluster** - Navigate to the
   [Stackrox setup tool](https://setup.rox.systems/). This tool lets you
   provision a temporary, self-destructing cluster in GCloud you will connect to
   during your development session. Once your cluster is provisioned and the
   status shows as `The cluster is ready`, copy the name of the 'Resource Group'
   and move on to step 2.

1. **Connect local machine to cluster** - Your local machine needs to be made
   aware of the cloud cluster you just created. Run `yarn connect [rg-name]`
   where `[rg-name]` is the name found in **Resource Group** you created in the
   previous step.

1. **Deploy StackRox** - Deploy a fresh copy of the StackRox software to your
   new cluster. During the deployment process, you may be asked for your
   DockerHub credentials.

    - Set up a load balancer by setting the env. variable by running
      `export LOAD_BALANCER=lb` (optional, if you want to add multiple clusters
      or just avoid port forwarding)
    - Set up persistent storage by setting the env. variable by running
      `export STORAGE=pvc` (optional, but if you need to bounce central during
      testing, then your changes will be saved)
    - Run `yarn deploy`

1. **Run local server** - Start your local server.
    - Ensure port forwarding is working by running `yarn forward` (The deploy
      script tries to do this, but it is flakey.)
    - Start the front-end by running `yarn start`.
    - This will open your web browser to
      [https://localhost:3000](https://localhost:3000)

_If your machine goes into sleep mode, you may lose the port forwarding set up
during the deploy step. If this happens, run `yarn forward` to restart port
forwarding._

#### Using an Existing StackRox Deployment with a Local Frontend

If you want to connect your local frontend app to a StackRox deployment that is
already running, you can use one of the following, depending on whether it has a
public IP. For both, start by visiting
[https://setup.rox.systems/](https://setup.rox.systems/).

-   If it has a public IP, find that IP by looking in the **nodes/pods** section
    in the center-right panel.
    -   Copy the **External IP** value.
    -   Export that in the `YARN_START_TARGET` env var and start the front-end
        by running, `export YARN_START_TARGET=<external_IP>; yarn start`
-   If it is a demo cluster, you can use the demo URL for `YARN_START_TARGET`.
-   If it does not have a public IP or a demo URL, you can use steps 2 and 4
    from the section above, **Using Remote StackRox Deployment**.
    1. Run `yarn connect [rg-name]` where `[rg-name]` is the name found in the
       'Resource Group' found in the existing cluster’s page in Setup.
    1. Run `yarn forward` in one terminal, and `yarn start` in another.

### IDEs

This project is IDE agnostic. For the best dev experience, it's recommended to
add / configure support for [ESLint](https://eslint.org/) and
[Prettier](https://prettier.io/) in the IDE of your choice.

Examples of configuration for some IDEs:

-   [Visual Studio Code](https://code.visualstudio.com/): Install plugins
    [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint)
    and
    [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode),
    then add configuration to `settings.json`:

```
"eslint.alwaysShowStatus": true,
"eslint.codeAction.showDocumentation": {
    "enable": true
},
"editor.codeActionsOnSave": {
    "source.fixAll": true
},
"[markdown]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
},
"[json]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
},
"[javascript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
}
```

-   [IntelliJ IDEA](https://www.jetbrains.com/idea/) /
    [WebStorm](https://www.jetbrains.com/webstorm/) /
    [GoLand](https://www.jetbrains.com/go/): Install and configure
    [ESLint plugin](https://plugins.jetbrains.com/plugin/7494-eslint). To apply
    autofixes on file save add
    [File Watcher](https://www.jetbrains.com/help/idea/using-file-watchers.html)
    to watch JavaScript files and to run ESLint program
    `rox/ui/node_modules/.bin/eslint` with arguments `--fix $FilePath$`.

### Browsers

For better development experience it's recommended to use
[Google Chrome Browser](https://www.google.com/chrome/) with the following
extensions installed:

-   [React Developer Tools](https://chrome.google.com/webstore/detail/react-developer-tools/fmkadmapgofadopljbjfkapdkoienihi?hl=en)
-   [Redux DevTools](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd?hl=en)
