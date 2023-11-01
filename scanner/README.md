# StackRox Scanner

Static Image, Node, and Orchestrator Scanner.

# Dev

## scanner

Scanner requires the Go version be aligned with the [EXPECTED_GO_VERSION](../EXPECTED_GO_VERSION).
This is verified when using any of Scanner's `make` targets.

### Local

Copy the sample config and edit it to your liking:
```sh
$ cp config.sample.yaml config.yaml
```

Generate the development TLS certificates:
```sh
$ make certs
```

Now you can build the dependencies and run scanner:
```sh
$ make deps
$ go run cmd/scanner -conf config.yaml
```

## scannerctl

### Local

If you're connecting to your local scanner, you should already have the TLS certificates generated. If not, refer to
running scanner in the above section.

Once you have the certificates and scanner running with those certificates, you can run scannerctl:
```sh
$ go run cmd/scannerctl/main.go \
    -certs certs/client \
    -server-name localhost \
    -port 8443 \
    https://registry.hub.docker.com/library/hello-world:latest
```