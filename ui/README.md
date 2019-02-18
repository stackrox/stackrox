# StackRox Platform Web Application (UI)

This sub-project contains Web UI (SPA) for StackRox Platform.
This project was bootstrapped with [Create React App](https://github.com/facebookincubator/create-react-app).

## Development

If you are developing only StackRox UI, then you don't have to install all the
build tooling described in the parent [README.md](../README.md). Instead, follow
the instructions below.

### Build Tooling

* [Docker](https://www.docker.com/)
* [Node.js](https://nodejs.org/en/) `8.12.0` or above, but still must be Node8.
if you're managing multiple versions of Node.js on your machine, consider using [nvm](https://github.com/creationix/nvm))
* [Yarn](https://yarnpkg.com/en/)

### Dev Env Setup

_Before starting, make sure you have the above tools installed on your machine
and you've run `yarn install` to download dependencies._

The front end development environment consists of a local static file server
used to serve static UI assets and a remote instance of StackRox for data and
API calls. Set up your environment as follows:

#### Using Local StackRox Deployment and Docker for Mac

_Note: Similar instructions apply when using [Minikube](https://kubernetes.io/docs/setup/minikube/)._

1. **Docker for Mac** - Make sure you have Kubernetes enabled in your Docker for Mac and `kubectl` is
pointing to `docker-for-deskop` (see [docker docs](https://docs.docker.com/docker-for-mac/#kubernetes)).  

1. **Deploy** - Make sure that your git working directory is clean and that the branch that you're on has a corresponding tag from CI (see Roxbot comment in a PR). Alternatively, you can check out master before deploying or specify the image tag you want to deploy by setting the `MAIN_IMAGE_TAG` var in your shell. Run the script at `../deploy/k8s/deploy-local.sh` to deploy the StackRox software. 

1. **Start** - Start your local server by running `yarn start`.

_Note: to redeploy a newer version of StackRox, currently the easiest way is by
deleting the whole `stackrox` namespace via `kubectl delete ns stackrox`, and
repeating the steps above._

#### Using Remote StackRox Deployment

1. **Provision back end infrastructure** - Navigate to the [Stackrox setup tool](https://setup.rox.systems/). This tool lets you provision a temporary, self destructing infrastructure in GCloud you will connect to during your development session. Hit the `+` button near the top left. Use the default form settings and provide a "Setup Name" (e.g. `yourname-dev`). Choose the number of hours you would like the cluster to remain active (This should be set to the expected hours of your development session). After you click `run` it may take up to 5 minutes to provision the new cluster. Once the status of your cluster shows as `The cluster is ready`, copy the name of the 'Resource Group' and move on to step 2.  

1. **Connect local machine to infrastructure** - Your local machine needs to be made aware of the cloud infrastructure you just created. run `yarn run connect [rg-name]` where `[rg-name]` is the name found in the 'Resource Group` you created in the previous step. This name can be found by going to  https://setup.rox.systems/ and selecting your setup name from the dropdown list.

1. **Deploy StackRox** - Deploy a fresh copy of the StackRox software to your new infrastructure by running `yarn run deploy`. During the deployment process, you may be asked for your Dockerhub credentials. In addition to deploying, this command will set up port forwarding from port 8000 to 3000 on your machine.

1. **Run local server** - Start your local server by running `yarn start`. This will open your web browser to [https://localhost:3000](https://localhost:3000)

_If your machine goes into sleep mode, you may lose the port forwarding set up during the deploy step. If this happens, run `yarn run forward` to restart port forwarding._

### Testing

#### Unit Tests
Use `yarn test` to run all unit tests and show test coverage.
To run tests and continously watch for changes use `yarn test-watch`.

#### End-to-end Tests (Cypress)

To bring up [Cypress](https://www.cypress.io/) UI use `yarn cypress-open`.
To run all end-to-end tests in a headless mode use `yarn test-e2e-local`.

### IDEs

This project is IDE agnostic. For the best dev experience, it's recommended to
add / configure support for [ESLint](https://eslint.org/) and [Prettier](https://prettier.io/)
in the IDE of your choice.

Examples of configuration for some IDEs:

* [Visual Studio Code](https://code.visualstudio.com/): Install plugins [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) and [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode),
then add configuration:

 ```
 "[javascript]": {
    "editor.formatOnSave": true
  },
  "prettier.eslintIntegration": true
```

* [IntelliJ IDEA](https://www.jetbrains.com/idea/) / [WebStorm](https://www.jetbrains.com/webstorm/) / [GoLand](https://www.jetbrains.com/go/): Install and configure [ESLint plugin](https://plugins.jetbrains.com/plugin/7494-eslint). To apply autofixes on file save add [File Watcher](https://www.jetbrains.com/help/idea/using-file-watchers.html) to watch JavaScript files and to run ESLint program `rox/ui/node_modules/.bin/eslint` with arguments `--fix $FilePath$`.

### Browsers

For better development experience it's recommended to use [Google Chrome Browser](https://www.google.com/chrome/) with the following extensions installed:

* [React Developer Tools](https://chrome.google.com/webstore/detail/react-developer-tools/fmkadmapgofadopljbjfkapdkoienihi?hl=en)
* [Redux DevTools](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd?hl=en)

