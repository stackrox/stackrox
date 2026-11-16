# roxie-overlay

Generates a [roxie](https://github.com/stackrox/roxie) deployment config for deploying **locally built** StackRox images to a local cluster.

## Why

Locally built images are tagged with the current dev tag (`make tag`, e.g. `5.1.x-562-g87fca7b4db`). Deploying that tag normally requires *every* main-tagged image (`main`, `central-db`, `scanner-v4`, `scanner-v4-db`, operator bundle) to exist for it. But an engineer usually only works on, and builds, some of them.

roxie-overlay lets you build just the images you're working on. It lays them as an **overlay** over a **base**: a nearby, CI-built tag that has a complete set of images. Images missing for the overlay tag come from the base.

## Usage

```sh
make image                              # build what you're working on
roxie deploy --config <(deploy/roxie-overlay.sh "$(make tag)")
```

On failure, `deploy/roxie-overlay.sh` writes an invisible ANSI reset sequence to stdout, which makes roxie
abort with a YAML error (`control characters are not allowed`) instead of deploying with an empty config.

If no overlay tag is given, the `MAIN_IMAGE_TAG` environment variable is tried. The resolution table and warnings go to stderr, the config to stdout.

## How it works

- **Each main-tagged image** is looked up for the overlay tag, locally (via container runtime) first, then in the remote image registry. If it isn't found, the base tag is used instead.
- **The base** is auto-resolved from the most recent commits (`git log --first-parent`): the newest one whose `main` image exists. You can also set it explicitly using `--base <tag>`.
- **The operator bundle** decides `roxie.version`. Overrides (`operator.operandTagOverrides`) are only written for images whose tag differs from it.
- **Collector and fact** are versioned independently (`COLLECTOR_VERSION`/`FACT_VERSION`), so by default the deployed operator's own versions apply. To deploy specific builds, pass `--collector-tag`/`--fact-tag`. Locally built collector or fact images are only transferred to the cluster when set this way (see [roxie#355](https://github.com/stackrox/roxie/issues/355)).

The tool aborts with an error if not a single image resolved to the overlay tag, or if any image can't be found anywhere. It warns when overlay and base tags are mixed.

## Container runtime for local lookups

Local image lookups talk to the container runtime over the Docker API, honoring
`DOCKER_HOST` (default: the local Docker socket). Docker needs no setup.

With **podman**, export `DOCKER_HOST` so lookups hit podman's socket — otherwise
they check the Docker socket and podman-built images are not found locally (they
then resolve from the registry, or fail if unpublished). Discover the socket with
`podman info`:

```sh
export DOCKER_HOST="unix://$(podman info --format '{{.Host.RemoteSocket.Path}}')"
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--base <tag>` | auto-resolved | base tag for images missing at the overlay tag |
| `--registry <registry>` | `quay.io/stackrox-io` | registry to look up images in (private registries use your local docker/podman login credentials) |
| `--collector-tag <tag>` | operator default | collector tag to deploy |
| `--fact-tag <tag>` | operator default | fact tag to deploy |
