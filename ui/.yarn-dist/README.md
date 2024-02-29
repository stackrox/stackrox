# .yarn-dist

Here you find a distribution of [yarn](https://yarnpkg.com/) package manager in a `.tgz` file.

It is intended for installing in a container such as [registry.access.redhat.com/ubi8/nodejs-18](https://catalog.redhat.com/software/containers/ubi8/nodejs-18/6278e5c078709f5277f26998), for example. This container has `node`, `npm` but does not have `yarn` which is required for UI builds.

This distribution was prepared with [Corepack](https://nodejs.org/docs/latest-v18.x/api/corepack.html#corepack) and is installable also only with Corepack. It can be installed without network access.

It is intended for Konflux builds where eventually all our builds have to execute without network access. It is also suitable for downstream CPaaS/OSBS builds where we can migrate from ACM yarn builder image to the above-mentioned `ubi8/nodejs-18` or similar.

## Usage

`Makefile` in this directory provides the following commands.

### Prepare `yarn` distribution tarball

For example, when you changed version to prepare.

```shell
$ make prepare
```

### Install `yarn` from prepared tarball

For example, in a container before starting the UI build.

```shell
$ make install
```
