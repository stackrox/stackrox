# StackRox Prevent UI

This sub-project contains Web UI (SPA) for Prevent product.
This project was bootstrapped with [Create React App](https://github.com/facebookincubator/create-react-app).

## Development

If you are developing only Prevent UI, then you don't have to install build tooling described in the parent [README.md](../README.md).
Instead follow the instructions below.

### Build Tooling

* [Docker](https://www.docker.com/)
* [Node.js](https://nodejs.org/en/) `8.12.0` or above (it's highly recommended to use an LTS version, if you're managing multiple versions of Node.js on your machine, consider using [nvm](https://github.com/creationix/nvm))
* [yarn](https://yarnpkg.com/en/)

### Dev Env Setup

The following steps assume that your working directory is the root `apollo` directory (not `apollo/ui` where this `README.md` is placed).

1. `docker login` - login to Docker Hub with your Docker ID
2. `docker pull stackrox/prevent:$(git describe --tags --abbrev=10 --dirty origin/master)` - pull the latest Prevent image
3. `docker swarm init` - initialize local Swarm cluster
4. `./deploy/swarm/deploy-local.sh` - deploy Prevent to a local Swarm cluster
5. `make -C ui start` - start a dev server to serve UI

### IDEs

This project is IDE agnostic. For the best dev experience it's recommended to add / configure support for [ESLint](https://eslint.org/) and [Prettier](https://prettier.io/) in IDE of your choice.

Examples of configuration for some IDEs:

* [Visual Studio Code](https://code.visualstudio.com/): Install plugins [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) and [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode), then add configuration:

 ```
 "[javascript]": {
    "editor.formatOnSave": true
  },
  "prettier.eslintIntegration": true
```

* [IntelliJ IDEA](https://www.jetbrains.com/idea/) / [WebStorm](https://www.jetbrains.com/webstorm/) / [GoLand](https://www.jetbrains.com/go/): Install and configure [ESLint plugin](https://plugins.jetbrains.com/plugin/7494-eslint). To apply autofixes on file save add [File Watcher](https://www.jetbrains.com/help/idea/using-file-watchers.html) to watch JavaScript files and to run ESLint program `apollo/ui/node_modules/.bin/eslint` with arguments `--fix $FilePath$`.

### Browsers

For better development experience it's recommended to use [Google Chrome Browser](https://www.google.com/chrome/) with the following extensions installed:

* [React Developer Tools](https://chrome.google.com/webstore/detail/react-developer-tools/fmkadmapgofadopljbjfkapdkoienihi?hl=en)
* [Redux DevTools](https://chrome.google.com/webstore/detail/redux-devtools/lmhkpmbekcpmknklioeibfkpmmfibljd?hl=en)
