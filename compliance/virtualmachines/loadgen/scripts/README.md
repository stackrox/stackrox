# Scripts

## build-loadgen.sh

Builds the load generator binary and container image.

```bash
./build-loadgen.sh [OPTIONS]

Options:
  --no-push     Build locally only
  --no-restart  Don't restart DaemonSet after push

Environment:
  VSOCK_LOADGEN_IMAGE  Image repository (default: quay.io/${USER}/stackrox/vsock-loadgen)
  VSOCK_LOADGEN_TAG    Image tag (default: latest)
```

## run-loadgen.sh

Deploys the load generator to the cluster.

```bash
./run-loadgen.sh [CONFIG_FILE]

Arguments:
  CONFIG_FILE  Path to config (default: ../deploy/loadgen-config.yaml)
```

Creates/updates the ConfigMap and deploys the DaemonSet.
