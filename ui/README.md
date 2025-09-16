# StackRox Web UI

## Repo Structure and Principles

### Root

The root directory and its `package.json` file serves as an entry point for any
interactions with the applications and packages.

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
-   [npm](https://npmjs.com/) v9.x

### Dev Env Setup

_Before starting, make sure you have the above tools installed on your machine
and you've run `npm ci` in the `apps/platform` directory to download dependencies._

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
   Note that Docker for Mac is no longer the most recommended k8s environment for development.
   Some recommended alternatives are `podman-desktop` or `colima`.

1. **Deploy** - Run `npm run deploy-local` (wraps `../deploy/k8s/deploy-local.sh`)
   to deploy the StackRox k8s app. Make sure that your git working directory is
   clean and that the branch that you're on has a corresponding tag from CI (see
   Roxbot comment for a PR branch). Alternatively, you can specify the image tag
   you want to deploy by setting the `MAIN_IMAGE_TAG` env var. If
   `npm run deploy-local` fails, see this
   [Knowledge Base article for debugging instructions](https://github.com/stackrox/dev-docs/blob/main/docs/troubleshooting/Troubleshooting-local-deployment.md).

1. **Start** - Start your local dev server by running `npm run start`. This will build
   the application in watch mode. To see
   available options to `npm run start`, first ensure that `npm run build` has been
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
