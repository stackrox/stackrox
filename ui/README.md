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
[Inner source](https://en.wikipedia.org/wiki/Inner_Source) model describes the
intent well.

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
-   `prepublishOnly` - recommended for preparing the package before publishing,
    e.g. it can in turn run the `build` step.

Note that not having a particular script defined will simply mean that this
package will be skipped when the corresponding script is run on the monorepo
root level. E.g. if the package doesn't have `test` script, then when CI runs
unit tests step for this monorepo, no tests will be executed for that package.

Ensure the following fields are correctly set in `package.json`:

```json
"name": "@stackrox/{package-name}",
"repository": {
    "type": "git",
    "url": "https://github.com/stackrox/rox.git",
    "directory": "ui/packages/{package-dir-name}"
},
"license": "UNLICENSED"
```

Note that the package should be scoped to `@stackrox` and no `"publishConfig"`
defined in `package.json` as publishing configuration is defined on the monorepo
root level.

Finally, make build modifications:

-   update [.ossls.yml](../.ossls.yml) to include package's `node_modules` dir;

#### Adding a Dependency to Another Package

If you need to add a dependency on `@stackrox/package-a` to
`@stackrox/package-b`, on the monorepo root level run the command

```
yarn lerna add @stackrox/package-a --scope @stackrox/package-b
```

(add `--dev` to add `@stackrox/package-a` as a dev dependency).

#### Publishing a New Package Version

-   Create a new branch you'll use to create a PR for versions bump.
-   Run `yarn lerna:version` that will ask you to pick new versions for the packages.
-   Commit the changes to the `package.json` files, push the branch and create a PR.
-   Once the PR is merged, CI will automatically publish new versions to GitHub
    Packages NPM registry.

As we're not using [conventional commits](https://www.conventionalcommits.org/),
use your best judgement for the version increase, considering
[semantic versioning](https://semver.org/) best practices.

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

```json
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

If you need to add a dependency on `@stackrox/package-a` to `@stackrox/my-app`,
on the monorepo root level run the command

```
yarn lerna add @stackrox/package-a --scope @stackrox/my-app
```

(add `--dev` to add `@stackrox/package-a` as a dev dependency).

Once the command succeeds, find `@stackrox/package-a` dependency in the
`package.json` file of `@stackrox/my-app` and change the version to `"*"` so
it's

```json
"@stackrox/package-a": "*"
```

The reason is that the app is never published therefore it should always depend
on the version of the package in the same monorepo. Yet when
[updating package versions](#publishing-a-new-package-version) the
`lerna:version` script will not touch any `package.json` files with
`"private": true`, potentially leaving the application to depend on an older
version of a package.

## Development

If you are developing only StackRox UI, then you don't have to install all the
build tooling described in the parent [README.md](../README.md). Instead, follow
the instructions below.

### Build Tooling

-   [Docker](https://www.docker.com/)
-   [Node.js](https://nodejs.org/en/) version compatible with the `"engine"`
    requirements in the [package.json](./package.json) file (It's highly
    recommended to use the latest LTS version. If you're managing multiple versions of
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
   [Knowledge Base article for debugging instructions](https://github.com/stackrox/dev-docs/blob/main/docs/troubleshooting/Troubleshooting-local-deployment.md).

1. **Start** - Start your local dev server by running `yarn start`. This will build 
   all monorepo packages in watch mode. To build and watch only the main UI and to see 
   available options to `yarn start`, first ensure that `yarn build` has been
   run from the top level and then refer to the [README.md](./apps/platform/README.md#running-the-development-server) 
   in the `apps/platform` directory.

_Note: to redeploy a newer version of StackRox, delete existing app using
`teardown` script from the [workflow](https://github.com/stackrox/workflow/)
repo, and repeat the steps above._

#### Using a Remote StackRox Deployment

To develop the front-end platform locally, but use a remote Central, please
refer to the detailed instructions in the how-to article
[Use remote Central for local front-end dev](https://github.com/stackrox/dev-docs/blob/main/docs/knowledge-base/%5BFE%5D%20Use-remote-Central-for-local-front-end-dev.md)

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

```json
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

