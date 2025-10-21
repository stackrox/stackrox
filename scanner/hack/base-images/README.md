# Base Image Detection (POC)

A small Go program to **infer an image’s base** by matching the target’s layer-digest prefix against a set of candidate base images. This is a **proof of concept**, not production software.

## What

* Resolves a target image (by tag) to a **platform-specific manifest** without pulling layers.
* Builds a set of **candidate base images** from:

  * Explicit `--base` refs.
  * The last `--max-tags` tags of each `--base-repo` (picked by **Created** time after a semver-ish prefilter).
* Compares **compressed layer digests**; longest matching **prefix** wins.
* Prints one line per target: ✓ `<base>` or ✗ `no match`.

## Why

Images inherit all base layers unchanged; application layers append on top of the base image. If the target’s layer list starts with the base’s layers, we have a strong evidence. From the point of managing these base images it's likely that CVEs and vulnerabilities associated with those layers would be fixed among all images by bumping the base images and rebuilding the application layers.

## Internals

* **Registry access:** uses `go-containerregistry` (`remote.Image`, `crane.ListTags`).
* **No pulls:** reads manifests/config (tiny blobs), not layers.
* **Platform selection:** `--platform os/arch` picks the right manifest in a list.
* **Cache:** JSON file cache under `/tmp/base_image_layers`, keyed by `(ref, platform, auth)`.

  * **One inspect per tag** (ever) for a given platform/auth.

## Run

* Go 1.21+ (modules).
* Optional registry auth (Docker `config.json`).

```
go run .
```

## Example

```
DOCKER_CONFIG=~/.docker/ go run main.go --cache-dir /tmp/base_image_layers --max-probe 50 --max-tags 50 --base-repo registry.access.redhat.com/ubi8 --base registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417 registry.redhat.io/rhel8/redis-6:1-1754453437 registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417
2025/08/14 23:04:45 expanding repo: registry.access.redhat.com/ubi8
2025/08/14 23:04:47 [/tmp/go-build2010359076/b001/exe/main --cache-dir /tmp/base_image_layers --max-probe 50 --max-tags 50 --base-repo registry.access.redhat.com/ubi8 --base registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417 registry.redhat.io/rhel8/redis-6:1-1754453437 registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417]
🕵️  registry.redhat.io/rhel8/redis-6:1-1754453437
✅ registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417
🕵️  registry.access.redhat.com/ubi8/s2i-core:8.10-1754427417
✅ registry.access.redhat.com/ubi8:8.10-1754402693
```
## Caching

Save layers output to files in a specified directory.

* At `/tmp/base_image_layers`.
* Stores the **entire inspect payload** (layers + created).
* Keyed by **ref + platform + auth-hash**.
* Speeds up repeated runs and multi-target analyses.

## Limitations and Outcomes

* Layer-prefix equality might not match successfuly if:

  * Squashed images, rebases, or rebuilds that alter layers.
  * Multi-stage builds where the final stage copies but doesn’t share base layers from the build images.

Outcomes:

* This validates the idea of a **quick, pull-free** signal of base lineage for many images.
* This **evaluates** whether digest-prefix matching is viable.
* This is **simple to read and change** while exploring the problem.
