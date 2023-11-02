# StackRox Scanner

The container image scanner.  Built with ClairCore technology.

## Development

Scanner requires the Go version to be aligned with the [EXPECTED_GO_VERSION](../EXPECTED_GO_VERSION).  This is verified using Scanner's `make` targets that depend on go tooling.

For local development, you can overwrite this restriction by specifying `EXPECTED_GO_VERSION` in the make targets that will depend on go tools, for example:

```
make build EXPECTED_GO_VERSION=$(go version | {read _ _ v _; echo $v})
```

### Running Scanner locally

Copy the sample config and edit it to your liking:

```sh
cp config.sample.yaml config.yaml
```

Build Scanner and generate the development TLS certificates:

```sh
make build certs
```

Run:

```sh
./bin/scanner -conf config.yaml
```

The build system by default builds for `GOOS=linux`.  If you are running a non-Linux OS specify the `GOOS` yourself, or use `HOST_OS`.

```sh
make GOOS='$(HOST_OS)' build
```

## Running Scanner with Kubernetes for testing 

Scanner contains a testing helm chart to deploy it standalone.  This is used for E2E testing or development.

```
make e2e-deploy
```

## scannerctl

There is a CLI that allows you to interact with Scanner, called [`scannerctl`](cmd/scannerctl/main.go).

### Local build

```sh
make build
```

Or:

```
make bin/scannerctl
```

### Running

There are many options to control how `scannerctl`.  See `scannerctl help`.

### Example 1: Connecting to local Scanner 

A common use case is testing Scanner locally.  Once you have the local scanner build, certificates, and Scanner running with those certificates, you can run `scannerctl`:

```sh
./bin/scannerctl scan \
    --certs certs/scannerctl \
    https://registry.hub.docker.com/library/hello-world:latest
```



#### Example 2: Connect to local Scanner in different modes

Setup Scanner to run locally in different modes: 

```sh
sed 's@certs_dir: ""@certs_dir: certs/scanner-v4@' config.yaml.sample > matcher-config.yaml
sed 's@certs_dir: ""@certs_dir: certs/scanner-v4@' config.yaml.sample > indexer-config.yaml
sed -i '/matcher:/!b;n; s/enable: .*/enable: false/' indexer-config.yaml
sed -i '/indexer:/!b;n; s/enable: .*/enable: false/' matcher-config.yaml
./bin/scanner -conf indexer-config.yaml &
ROX_METRICS_PORT=:9091 ./bin/scanner -conf matcher-config.yaml &
```

Call `scannerctl`:

```sh
./bin/scannerctl scan \
    --certs certs/scannerctl \
    --indexer-address=:8443 \
    --matcher-address=:8444 \
    'https://docker.io/library/ubuntu:16.04'
```
