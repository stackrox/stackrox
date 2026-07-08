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
-   [npm](https://npmjs.com/) (will be installed with Node.js, or use the version compatible with your Node.js version)

### Dev Env Setup

_Before starting, make sure you have the above tools installed on your machine
and you've run `npm ci` in the `apps/platform` directory to download dependencies._

The front end development environment consists of a Vite dev server to serve
static UI assets and deployed StackRox Docker containers that provide backend API.

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
   you want to deploy by setting the `MAIN_IMAGE_TAG` env var.

1. **Start** - Start your local dev server by running `npm run start`. This will build
   the application in watch mode. To see
   available options to `npm run start`, first ensure that `npm run build` has been
   run from the top level and then refer to the [README.md](./apps/platform/README.md#running-the-development-server)
   in the `apps/platform` directory.

_Note: to redeploy a newer version of StackRox, delete existing app using
`teardown` script from the [workflow](https://github.com/stackrox/workflow/)
repo, and repeat the steps above._

#### Using a Remote StackRox Deployment

To point your local dev server at a remote Central, set `UI_START_TARGET` to
the remote endpoint and start the dev server:

```sh
UI_START_TARGET=https://<central-ip>:443 npm run start
```

Prefer using a publicly accessible IP or load balancer for the remote Central.
`kubectl port-forward` can be used as a last resort if no accessible IP is
available, but it is unreliable — connections drop on machine sleep and under
load.

See [`apps/platform/README.md`](./apps/platform/README.md#running-the-development-server)
for all available proxy environment variables (`UI_START_TARGET`,
`UI_CUSTOM_PROXIES`).

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
-   [Redux DevTools](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd?hl=en) (for legacy Redux code inspection)

## Dependency Vulnerability Management

### Why audit UI dependencies?

StackRox UI bundles many npm dependencies into minified JS. Vulnerabilities in
these dependencies carry real risk:

-   Customers may expose the StackRox UI to the public (relying on
    authentication only), making client-side vulnerabilities exploitable.
-   Government and enterprise contracts often require timely remediation of
    known critical vulnerabilities.
-   Annual penetration testing reports go to customers — discovered
    vulnerabilities require explanation and a timeline of exposure.

### Running an audit

From the `ui/apps/platform/` directory:

```sh
npm audit --omit=dev
```

The `--omit=dev` flag limits the audit to production dependencies (dev and
optional dependencies are not shipped to users). HIGH and CRITICAL findings are prioritized according to ProdSec guidelines.

### Updating a vulnerable dependency

**Direct dependency** — update it in `package.json`:

```sh
npm install <package>@<fixed-version>
```

**Indirect (transitive) dependency** — when the vulnerable package is pulled in
by another dependency:

1. Check whether the parent package's version range already accepts the fixed
   version (e.g. `"dompurify": "^2.0.3"` accepts `2.0.7`).
2. If it does, delete the vulnerable package's entry from `package-lock.json`
   and run `npm install`. npm will resolve to the latest version allowed by the
   parent's range.
3. If the parent's range does not accept the fix, check for a newer version of
   the parent package that widens the range, or use
   [`npm overrides`](https://docs.npmjs.com/cli/v10/configuring-npm/package-json#overrides)
   as a last resort.
4. Commit the updated `package-lock.json`.
