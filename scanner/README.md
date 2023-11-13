# StackRox Scanner

The container image scanner.  Built with [ClairCore](https://github.com/quay/claircore) technology.

## Development

It is recommended to use the [EXPECTED_GO_VERSION](../EXPECTED_GO_VERSION), but this is not enforced for development.
It is, however, enforced in CI.

### Running locally

To run Scanner locally for development, copy the sample config and edit it to your liking:

```sh
cp config.yaml.sample config.yaml
```

Build Scanner binaries and generate the development TLS certificates:

```sh
make build certs
```

Run:

```sh
./bin/scanner -conf config.yaml
```

**Note**: Scanner requires a PostgreSQL database, so be sure one is running and `config.yaml` points to it.

### Running standalone in Kubernetes

Scanner contains a testing helm chart to deploy it standalone.  This is used for E2E testing or development.

Run Scanner and Scanner DB:

```sh
make e2e-deploy
```

## scannerctl

There is a CLI that allows you to interact with Scanner, called [`scannerctl`](cmd/scannerctl/main.go).

To build it, use:

```sh
make build
```

Or, specifically:

```sh
make bin/scannerctl
```

### Examples

There are options to control how to run `scannerctl`.  See `scannerctl help`.

#### Example 1: Connecting to local Scanner 

A common use case is testing Scanner locally.  Once you have built local Scanner, certificates, and Scanner running with those certificates, you can run `scannerctl`:

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
sed -e 's/http_listen_addr: .*/http_listen_addr: ":9444"/' \
    -e 's/grpc_listen_addr: .*/grpc_listen_addr: ":8444"/' \
    matcher-config.yaml
./bin/scanner -conf indexer-config.yaml &
ROX_METRICS_PORT=:9091 ./bin/scanner -conf matcher-config.yaml &
```

Check updates:

```
% diff -du config.yaml.sample indexer-config.yaml
--- config.yaml.sample	2023-10-28 10:24:35.123934825 -0700
+++ indexer-config.yaml	2023-11-02 13:53:06.051843337 -0700
@@ -7,10 +7,10 @@
     password_file: ""
   get_layer_timeout: 1m
 matcher:
-  enable: true
+  enable: false
   database:
     conn_string: "host=/var/run/postgresql"
     password_file: ""
 mtls:
-  certs_dir: ""
+  certs_dir: certs/scanner-v4
 log_level: info
% diff -du config.yaml.sample matcher-config.yaml
--- config.yaml.sample	2023-10-28 10:24:35.123934825 -0700
+++ matcher-config.yaml	2023-11-02 13:58:45.276489478 -0700
@@ -1,7 +1,7 @@
-http_listen_addr: 127.0.0.1:9443
-grpc_listen_addr: 127.0.0.1:8443
+http_listen_addr: 127.0.0.1:9444
+grpc_listen_addr: 127.0.0.1:8444
 indexer:
-  enable: true
+  enable: false
   database:
     conn_string: "host=/var/run/postgresql"
     password_file: ""
@@ -12,5 +12,5 @@
     conn_string: "host=/var/run/postgresql"
     password_file: ""
 mtls:
-  certs_dir: ""
+  certs_dir: certs/scanner-v4
 log_level: info
```

Call `scannerctl`:

```sh
./bin/scannerctl scan \
    --certs certs/scannerctl \
    --indexer-address=:8443 \
    --matcher-address=:8444 \
    'https://docker.io/library/ubuntu:16.04'
```
