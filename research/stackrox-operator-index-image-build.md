# How `quay.io/rhacs-eng/stackrox-operator-index` is Produced

This document describes the build process for the StackRox operator index (catalog) image.

## Overview

The operator index image is an **OLM (Operator Lifecycle Manager) catalog image** that allows OLM to discover and install the StackRox operator through a subscription. It is built through a multi-step process primarily driven by:

- **GitHub Actions workflow**: `.github/workflows/build.yaml`
- **Operator Makefile**: `operator/Makefile`
- **Build script**: `operator/hack/build-index-image.sh`

## Build Pipeline

The image is produced by the `build-and-push-operator` job in GitHub Actions:

```
1. Build operator bundle
2. Push bundle image
3. Build index image (requires bundle to be pushed first)
4. Push index image
```

## Image Naming

| Component | Value |
|-----------|-------|
| Registry | `quay.io/rhacs-eng` |
| Image name | `stackrox-operator-index` |
| Tag format | `v{VERSION}` (e.g., `v4.7.0`) |
| Full image | `quay.io/rhacs-eng/stackrox-operator-index:v{VERSION}` |

The image tag base is constructed in `operator/Makefile`:

```makefile
IMAGE_REPO ?= $(shell $(MAKE) --quiet --no-print-directory -C .. default-image-registry)
IMAGE_TAG_BASE ?= $(IMAGE_REPO)/stackrox-operator

INDEX_IMG_BASE = $(IMAGE_TAG_BASE)-index
INDEX_IMG_TAG ?= $(CSV_VERSION)
INDEX_IMG ?= $(INDEX_IMG_BASE):$(INDEX_IMG_TAG)
```

## Build Process Details

### 1. Makefile Target: `index-build`

Located in `operator/Makefile`:

```makefile
index-build: bundle-post-process yq
    replaced_version=$$(sed -E -n 's/^[[:space:]]*replaces:[[:space:]]*[^.]+\.(.*)$$/\1/p' \
        build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml)
    YQ=$(YQ) ./hack/build-index-image.sh \
        --base-index-tag "$(INDEX_IMG_BASE):$${replaced_version}" \
        --index-tag "$(INDEX_IMG)" \
        --bundle-tag "$(BUNDLE_IMG)" \
        --replaced-version "$${replaced_version}" \
        --clean-output-dir
```

Key points:
- Depends on `bundle-post-process` — the bundle image must be pushed first
- Extracts the `replaces` version from the ClusterServiceVersion manifest
- Uses the previous version's index as a base

### 2. Build Script: `build-index-image.sh`

Located at `operator/hack/build-index-image.sh`, this script performs the actual build:

#### Step 1: Download OPM
```bash
OPM_VERSION="1.21.0"
# Downloads opm from GitHub releases
```

#### Step 2: Generate Dockerfile and Render Base Index
```bash
"${OPM}" generate dockerfile --binary-image "quay.io/operator-framework/opm:v${OPM_VERSION}" "${BUILD_INDEX_DIR}"
"${OPM}" render "${BASE_INDEX_TAG}" --output=yaml > "${BUILD_INDEX_DIR}/index.yaml"
```

#### Step 3: Add New Bundle Entry
```bash
YQ_NEW_BUNDLE_ENTRY=$(cat <<EOF
{
    "name": "rhacs-operator.v${BUNDLE_VERSION}",
    "replaces": "rhacs-operator.v${REPLACED_VERSION}",
    "skipRange": ">= ${REPLACED_VERSION} < ${BUNDLE_VERSION}"
}
EOF
)

"${YQ}" --inplace --prettyPrint "with(select(${YQ_FILTER_CHANNEL_DOCUMENT}); .entries += ${YQ_NEW_BUNDLE_ENTRY})" "${BUILD_INDEX_DIR}/index.yaml"
"${OPM}" render "${BUNDLE_TAG}" --output=yaml >> "${BUILD_INDEX_DIR}/index.yaml"
```

#### Step 4: Validate and Build
```bash
"${OPM}" validate "${BUILD_INDEX_DIR}"
docker build --quiet --file "${BUILD_INDEX_DIR}.Dockerfile" --tag "${INDEX_TAG}" "${BUILD_INDEX_DIR}/.."
```

### 3. GitHub Actions Trigger

From `.github/workflows/build.yaml`:

```yaml
# Index image can only be built once bundle was pushed
- name: Build index
  if: |
    github.event_name == 'push' || !github.event.pull_request.head.repo.fork
  run: |
    make -C operator/ index-build

- name: Push index image
  if: |
    github.event_name == 'push' || !github.event.pull_request.head.repo.fork
  run: |
    make -C operator/ docker-push-index | cat
```

**Trigger conditions:**
- Push events to any branch
- Pull requests from non-fork repositories (internal PRs)
- External contributions (forks) skip index build/push

## Technical Details

| Aspect | Detail |
|--------|--------|
| Build tool | `opm` (Operator Package Manager) v1.21.0 |
| Catalog format | File-Based Catalog (FBC) with YAML index |
| Base image | `quay.io/operator-framework/opm:v1.21.0` |
| OLM channel | `latest` |
| Upgrade strategy | Incremental — each new index builds on the previous version's index |

## Upgrade Chain

The index maintains an upgrade chain through the `replaces` field:

```
v3.62.0 → v3.63.0 → v3.64.0 → ... → current version
```

Each new version:
1. Pulls the previous version's index as a base
2. Adds the new bundle with `replaces` pointing to the previous version
3. Includes a `skipRange` to allow upgrading from any version in range

## Local Development

To build the index image locally:

```bash
# Build everything (operator, bundle, index) and push
make -C operator/ everything

# Or build just the index (requires bundle to be pushed first)
make -C operator/ index-build

# Push the index image
make -C operator/ docker-push-index
```

## Related Images

| Image | Purpose |
|-------|---------|
| `stackrox-operator` | The operator controller image |
| `stackrox-operator-bundle` | OLM bundle containing manifests and metadata |
| `stackrox-operator-index` | OLM catalog/index containing the bundle references |

## References

- [Operator Makefile](../operator/Makefile)
- [Build Index Script](../operator/hack/build-index-image.sh)
- [GitHub Actions Workflow](../.github/workflows/build.yaml)
- [OPM Documentation](https://olm.operatorframework.io/docs/cli-tools/opm/)
