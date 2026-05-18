# Docker Buildx Bake Consolidation: Scanner, Roxctl, Operator, and Main Images

## Context

Currently, four major container images are built in separate GitHub Actions matrix jobs:
- **main** - built in `build-and-push-main` (2 branding × 2-4 arch = 4-8 jobs)
- **scanner** - built in `build-and-push-scanner` (1 × 2-4 arch = 2-4 jobs)
- **operator** - built in `build-and-push-operator` (2 branding × 2 arch = 4 jobs)
- **roxctl** - built in `build-and-push-roxctl` (1 × 2-4 arch = 2-4 jobs)

**Total**: ~12-18 separate job invocations + 3 manifest jobs = **15-21 jobs**

### Why Consolidate?

All four images share the same technical foundation:
- **UBI9-micro base image** (~40MB) - identical for all
- **UBI9 full image** for package installation (~180MB) - identical for all
- **Similar dnf packages**: ca-certificates, gzip, less, tar, openssl
- **Branding handled via ENV variables**: ROX_IMAGE_FLAVOR, ROX_PRODUCT_BRANDING (runtime only, not build-time)

**Opportunity**: Use Docker Buildx bake to build all four images in parallel with shared layer caching:
- Single UBI9 base pull reused by all four images
- Single dnf package installation layer (with RUN cache mount) reused where packages overlap
- Build once per branding variant, not per image
- Replace 15-21 jobs with **2 combined jobs** (one per branding)

### Prior Work

This builds on proven patterns from the ROX-34147 branch:

**PR #20588 (central-db extraction)**:
- Extracted central-db from build-and-push-main
- Uses `docker/build-push-action` with platforms parameter
- Builds once to stackrox-io, skopeo copies to rhacs-eng
- Multi-arch manifest created automatically by buildx

**PR #20617 (operator consolidation)** - Commit 383348efc5:
- Proved ROX_IMAGE_FLAVOR is runtime ENV only (not build-time difference)
- Eliminated branding matrix - builds ONCE with `ROX_IMAGE_FLAVOR=opensource`
- Uses `operator/prebuilt.Dockerfile` with pre-built binaries
- Skopeo copy to rhacs-eng (no rebuild needed)

**Current roxctl extraction**:
- Already extracted roxctl into dedicated `build-and-push-roxctl` job
- Uses modified `image/roxctl/Dockerfile` with `ARG TARGETARCH`
- Follows same pattern as DB/operator (buildx + skopeo)

---

## Goal

Replace multiple matrix jobs with **2 Docker Buildx bake jobs** (one per branding variant) that build scanner, roxctl, operator, and main images in parallel with shared layer caching.

**Expected outcome**: ~15-21 jobs → 2 jobs, 40-60% faster builds via shared UBI9 and dnf layer caching.

---

## Requirements

### 1. Base Branch
- Branch off `ROX-34147/extract-db-build` (PR #20588)
- This branch already has central-db extracted and uses buildx patterns

### 2. Docker Bake Configuration File

**Create**: `.github/docker/components-bake.hcl`

**Structure**:
```hcl
variable "BUILD_TAG" {
  default = "dev"
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
}

variable "ROX_IMAGE_FLAVOR" {
  default = "opensource"
}

variable "PUSH_TO_REGISTRY" {
  default = "quay.io/stackrox-io"
}

group "default" {
  targets = ["main", "scanner", "operator", "roxctl"]
}

target "main" {
  dockerfile = "image/rhel/Dockerfile"
  context    = "."
  platforms  = split(",", PLATFORMS)
  tags       = ["${PUSH_TO_REGISTRY}/main:${BUILD_TAG}"]
  cache-from = ["type=gha,scope=components-main"]
  cache-to   = ["type=gha,mode=max,scope=components-main"]
  args = {
    ROX_IMAGE_FLAVOR = ROX_IMAGE_FLAVOR
    ROX_PRODUCT_BRANDING = ROX_IMAGE_FLAVOR == "opensource" ? "STACKROX_BRANDING" : "RHACS_BRANDING"
  }
}

target "scanner" {
  dockerfile = "scanner/image/scanner/Dockerfile"
  context    = "scanner/image/scanner"
  platforms  = split(",", PLATFORMS)
  tags       = ["${PUSH_TO_REGISTRY}/scanner-v4:${BUILD_TAG}"]
  cache-from = ["type=gha,scope=components-scanner"]
  cache-to   = ["type=gha,mode=max,scope=components-scanner"]
}

target "operator" {
  dockerfile = "operator/prebuilt.Dockerfile"
  context    = "."
  platforms  = split(",", PLATFORMS)
  tags       = ["${PUSH_TO_REGISTRY}/stackrox-operator:${BUILD_TAG}"]
  cache-from = ["type=gha,scope=components-operator"]
  cache-to   = ["type=gha,mode=max,scope=components-operator"]
  args = {
    ROX_IMAGE_FLAVOR = ROX_IMAGE_FLAVOR
  }
}

target "roxctl" {
  dockerfile = "image/roxctl/Dockerfile"
  context    = "."
  platforms  = split(",", PLATFORMS)
  tags       = ["${PUSH_TO_REGISTRY}/roxctl:${BUILD_TAG}"]
  cache-from = ["type=gha,scope=components-roxctl"]
  cache-to   = ["type=gha,mode=max,scope=components-roxctl"]
}
```

**Key design choices**:
- Per-image cache scopes prevent eviction conflicts
- Shared base layers cached automatically by Docker's content-addressable storage
- Single registry in bake file (stackrox-io OR rhacs-eng, not both)
- Platform specification variable allows PR vs. full builds

### 3. Dockerfile Modifications

All four Dockerfiles need two changes:
1. Add `ARG TARGETARCH` for multi-arch binary selection
2. Add `--mount=type=cache` for dnf package installations

#### 3a. Main Image Dockerfile

**File**: `image/rhel/Dockerfile`

**Find**: Package installation RUN command (~line 40-60)
```dockerfile
RUN dnf upgrade -y --nobest && \
    dnf install -y ...
```

**Change to**:
```dockerfile
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    --mount=type=cache,target=/var/cache/yum,sharing=locked \
    dnf upgrade -y --nobest && \
    dnf install -y ...
```

**Find**: Binary copy commands that reference GOARCH (~line 90-100)
```dockerfile
COPY bin/linux_${GOARCH}/central /stackrox/central
```

**Verify**: Should already use `${GOARCH}` which is equivalent to `${TARGETARCH}`. If using hardcoded paths, update to use `ARG TARGETARCH`.

#### 3b. Scanner Dockerfile

**File**: `scanner/image/scanner/Dockerfile`

**Find**: Package installation RUN command
```dockerfile
RUN dnf install -y ...
```

**Change to**:
```dockerfile
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    --mount=type=cache,target=/var/cache/yum,sharing=locked \
    dnf install -y ...
```

**Find**: Binary copy command (~line 54)
```dockerfile
COPY bin/scanner /usr/local/bin/
```

**Change to**:
```dockerfile
ARG TARGETARCH
COPY bin/linux_${TARGETARCH}/scanner /usr/local/bin/scanner
```

#### 3c. Operator Dockerfile

**File**: `operator/prebuilt.Dockerfile`

**Check**: Already has `ARG TARGET_ARCH` but should use `TARGETARCH` (buildx standard)

**Find**:
```dockerfile
ARG TARGET_ARCH=amd64
...
COPY bin/linux_${TARGET_ARCH}/stackrox-operator /usr/local/bin/
```

**Change to**:
```dockerfile
ARG TARGETARCH=amd64
...
COPY bin/linux_${TARGETARCH}/stackrox-operator /usr/local/bin/
```

**Note**: operator/prebuilt.Dockerfile doesn't have dnf installations (uses ubi9-micro directly), so no cache mount needed.

#### 3d. Roxctl Dockerfile

**File**: `image/roxctl/Dockerfile`

**Check**: Already modified in recent commit (5f3f337288) to support TARGETARCH.

**Find**: Package installation RUN command (~line 9-19)
```dockerfile
RUN dnf install -y \
    --installroot=/out/ \
    ...
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum
```

**Change to**:
```dockerfile
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    --mount=type=cache,target=/var/cache/yum,sharing=locked \
    dnf install -y \
    --installroot=/out/ \
    ...
    dnf clean all --installroot=/out/
```

**Remove**: The manual cache cleanup (`rm -rf /out/var/cache/dnf`) as cache mount handles this.

### 4. Workflow Jobs

Create two new jobs in `.github/workflows/build.yaml`:

#### 4a. Build Components for StackRox

**Job name**: `build-and-push-components-stackrox`

**Insert after**: `build-and-push-roxctl` job

```yaml
  build-and-push-components-stackrox:
    runs-on: ubuntu-latest
    needs:
      - define-job-matrix
      - pre-build-go-binaries
      - pre-build-cli
      - pre-build-oss-notice
    if: ${{ !cancelled() }}
    env:
      BUILD_TAG: ${{ needs.define-job-matrix.outputs.build-tag }}
      ROX_IMAGE_FLAVOR: opensource
      ROX_PRODUCT_BRANDING: STACKROX_BRANDING
    steps:
      - name: Checkout
        uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd
        with:
          ref: ${{ inputs.commit || github.event.pull_request.head.sha }}

      - uses: ./.github/actions/job-preamble
        with:
          gcp-account: ${{ secrets.GCP_SERVICE_ACCOUNT_STACKROX_CI }}

      - name: Login to docker.io to mitigate rate limiting
        uses: docker/login-action@4907a6ddec9925e35a0a9e82d7399ccc52663121
        if: github.event_name == 'push' || !github.event.pull_request.head.repo.fork
        with:
          username: ${{ secrets.DOCKERHUB_CI_ACCOUNT_USERNAME }}
          password: ${{ secrets.DOCKERHUB_CI_ACCOUNT_PASSWORD }}

      - name: Set up QEMU
        uses: docker/setup-qemu-action@ce360397dd3f832beb865e1373c09c0e9f86d70a

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@4d04d5d9486b7bd6fa91e7baf45bbb4f8b9deedd

      - name: Determine platforms
        id: platforms
        run: |
          source ./scripts/ci/lib.sh
          PLATFORMS="linux/amd64,linux/arm64"
          if ! is_in_PR_context || pr_has_label ci-build-all-arch; then
            PLATFORMS="linux/amd64,linux/arm64,linux/ppc64le,linux/s390x"
          fi
          echo "platforms=${PLATFORMS}" >> "$GITHUB_OUTPUT"

      # Download all binary artifacts
      - name: Download go-binaries (amd64)
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: go-binaries-build-amd64-default
          path: binaries/amd64/

      - name: Download go-binaries (arm64)
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: go-binaries-build-arm64-default
          path: binaries/arm64/

      - name: Download go-binaries (ppc64le)
        if: contains(steps.platforms.outputs.platforms, 'ppc64le')
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: go-binaries-build-ppc64le-default
          path: binaries/ppc64le/

      - name: Download go-binaries (s390x)
        if: contains(steps.platforms.outputs.platforms, 's390x')
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: go-binaries-build-s390x-default
          path: binaries/s390x/

      - name: Download roxctl binaries
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: cli-build
          path: .

      - name: Download scanner OSS notices
        uses: ./.github/actions/download-artifact-with-retry
        with:
          name: oss-notice
          path: scanner/image/scanner/THIRD_PARTY_NOTICES

      # Extract and stage binaries for all architectures
      - name: Extract and stage binaries for all architectures
        run: |
          set -euo pipefail

          # Extract and stage scanner, operator, main binaries from go-binaries
          for arch in amd64 arm64 ppc64le s390x; do
            if [ -d "binaries/${arch}" ]; then
              echo "Staging binaries for ${arch}"
              tar -xzf "binaries/${arch}/go-binaries-build.tgz" -C "binaries/${arch}"

              # Scanner binary
              mkdir -p "scanner/image/scanner/bin/linux_${arch}"
              cp "binaries/${arch}/bin/linux_${arch}/scanner" \
                 "scanner/image/scanner/bin/linux_${arch}/scanner"

              # Operator binary
              mkdir -p "bin/linux_${arch}"
              cp "binaries/${arch}/bin/linux_${arch}/stackrox-operator" \
                 "bin/linux_${arch}/stackrox-operator"

              # Main binaries (central, sensor, etc.) - already in bin/linux_${arch}/
              # No need to copy, buildx will use them from binaries/${arch}/bin/
            fi
          done

          # Extract roxctl binaries (already in correct bin/linux_* structure)
          tar -xzf cli-build.tgz

          # Verify binaries present for always-built architectures
          for arch in amd64 arm64; do
            test -f "scanner/image/scanner/bin/linux_${arch}/scanner" || \
              { echo "ERROR: Scanner binary missing for ${arch}"; exit 1; }
            test -f "bin/linux_${arch}/stackrox-operator" || \
              { echo "ERROR: Operator binary missing for ${arch}"; exit 1; }
            test -f "bin/linux_${arch}/roxctl" || \
              { echo "ERROR: Roxctl binary missing for ${arch}"; exit 1; }
          done

          echo "Binary staging complete"

      - name: Login to quay.io/stackrox-io
        if: github.event_name == 'push' || !github.event.pull_request.head.repo.fork
        uses: docker/login-action@4907a6ddec9925e35a0a9e82d7399ccc52663121
        with:
          registry: quay.io
          username: ${{ secrets.QUAY_STACKROX_IO_RW_USERNAME }}
          password: ${{ secrets.QUAY_STACKROX_IO_RW_PASSWORD }}

      - name: Build and push all components with bake to stackrox-io
        if: github.event_name == 'push' || !github.event.pull_request.head.repo.fork
        run: |
          docker buildx bake \
            -f .github/docker/components-bake.hcl \
            --set "*.platform=${{ steps.platforms.outputs.platforms }}" \
            --set "*.args.BUILD_TAG=${{ env.BUILD_TAG }}" \
            --set "main.args.ROX_IMAGE_FLAVOR=${{ env.ROX_IMAGE_FLAVOR }}" \
            --set "main.args.ROX_PRODUCT_BRANDING=${{ env.ROX_PRODUCT_BRANDING }}" \
            --set "operator.args.ROX_IMAGE_FLAVOR=${{ env.ROX_IMAGE_FLAVOR }}" \
            --push \
            default

      - name: Add latest tags for master merges
        if: github.event_name == 'push' && github.ref_name == 'master'
        run: |
          for image in main scanner-v4 stackrox-operator roxctl; do
            docker buildx imagetools create \
              --tag "quay.io/stackrox-io/${image}:latest" \
              "quay.io/stackrox-io/${image}:${{ env.BUILD_TAG }}"
          done
```

#### 4b. Build Components for RHACS

**Job name**: `build-and-push-components-rhacs`

Copy the entire `build-and-push-components-stackrox` job and modify:
- Job name: `build-and-push-components-rhacs`
- ENV: `ROX_IMAGE_FLAVOR: development_build`
- ENV: `ROX_PRODUCT_BRANDING: RHACS_BRANDING`
- Registry: `quay.io/rhacs-eng` (update login and bake PUSH_TO_REGISTRY)
- Dependencies: Should run in parallel with stackrox job

### 5. Cleanup - Remove Old Jobs

**Delete these jobs** from `.github/workflows/build.yaml`:
1. `build-and-push-main`
2. `build-and-push-scanner`
3. `build-and-push-operator`
4. `build-and-push-roxctl`
5. `push-main-manifests`
6. `push-scanner-manifests`
7. `push-operator-manifests`

**Update job dependencies**:
- Any job that depends on the old jobs should now depend on:
  - `build-and-push-components-stackrox`
  - `build-and-push-components-rhacs`

**Examples**:
- `scan-images-with-roxctl` → depends on both component jobs
- `slack-on-build-failure` → depends on both component jobs

### 6. Update Matrix Definitions

**File**: `.github/workflows/build.yaml` (define-job-matrix job)

**Remove** these matrix definitions:
```bash
# DELETE:
matrix="$(jq '.build_and_push_main.name += ["RHACS_BRANDING", "STACKROX_BRANDING"]' <<< "$matrix")"
matrix="$(jq '.build_and_push_main.arch += ["amd64", "arm64"]' <<< "$matrix")"
matrix="$(jq '.push_main_multiarch_manifests.name += ["RHACS_BRANDING", "STACKROX_BRANDING"]' <<< "$matrix")"

matrix="$(jq '.build_and_push_operator.name += ["RHACS_BRANDING", "STACKROX_BRANDING"]' <<< "$matrix")"
matrix="$(jq '.build_and_push_operator.arch += ["amd64", "arm64"]' <<< "$matrix")"
matrix="$(jq '.push_operator_multiarch_manifests.name += ["RHACS_BRANDING", "STACKROX_BRANDING"]' <<< "$matrix")"

matrix="$(jq '.build_and_push_scanner.name += ["default"]' <<< "$matrix")"
matrix="$(jq '.build_and_push_scanner.arch += ["amd64", "arm64"]' <<< "$matrix")"
matrix="$(jq '.push_scanner_manifests.name += ["default"]' <<< "$matrix")"
```

These matrix definitions are no longer needed since bake handles all platforms in single jobs.

### 7. Update scripts/ci/lib.sh

**Remove functions or update references**:
- `push_main_image_set()` - likely no longer needed (buildx pushes directly)
- `push_image_manifest_lists()` - likely no longer needed (buildx creates manifests)
- `push_operator_image()` - no longer needed

**Verify**: Check if any other scripts reference these functions and update accordingly.

---

## Technical Details

### Why Docker Buildx Bake?

**Advantages over separate jobs**:
1. **Shared layer caching**: All four images share UBI9 base → pulled once, reused 4 times
2. **Shared package layers**: dnf installations with cache mounts → packages downloaded once
3. **Parallel building**: Bake builds all four images simultaneously (not sequentially)
4. **Single registry authentication**: Login once, push four images
5. **Atomic operation**: All four images succeed or fail together

**Layer sharing example**:
```
UBI9-micro base (40MB)       → Shared by all 4 images, cached once
UBI9 full (180MB)            → Shared by all 4 images, cached once
dnf install ca-certificates  → Shared by all 4 images, cached once
dnf install gzip, less, tar  → Shared by scanner, roxctl, operator, cached once
Main-specific packages       → Only main image
Scanner-specific binaries    → Only scanner image
```

**Expected build time**:
- **Current**: ~15-21 separate jobs, each pulling UBI9, each installing packages
  - Total: ~45 minutes (with parallelization)
- **With bake**: 2 jobs, UBI9 pulled twice (once per branding), packages installed twice
  - Total: ~15-20 minutes (40-55% faster)

### Cache Mount Benefits

**Without cache mount**:
```dockerfile
RUN dnf install -y ca-certificates gzip && \
    dnf clean all && \
    rm -rf /var/cache/dnf
```
Every build downloads packages, even if unchanged.

**With cache mount**:
```dockerfile
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
    dnf install -y ca-certificates gzip
```
Packages downloaded once, cached persistently across builds. GitHub Actions cache stores this.

**Sharing levels**:
- `sharing=locked` → Multiple builds can read the cache, but only one can write at a time (safe for parallel builds)
- `sharing=shared` → Multiple builds can read/write simultaneously (faster but less safe)
- `sharing=private` → Each build gets its own cache (no sharing)

Use `sharing=locked` for dnf cache mounts.

### Platform Selection Logic

**PR builds**: Default to `linux/amd64,linux/arm64` (faster feedback)

**Non-PR or labeled**: All 4 platforms `linux/amd64,linux/arm64,linux/ppc64le,linux/s390x`

**Override**: PRs with `ci-build-all-arch` label build all 4 platforms

This is handled in the "Determine platforms" step using `scripts/ci/lib.sh` functions.

---

## Verification Steps

After implementing, verify:

### 1. Multi-arch Manifests Created

```bash
docker buildx imagetools inspect quay.io/stackrox-io/main:${BUILD_TAG}
docker buildx imagetools inspect quay.io/stackrox-io/scanner-v4:${BUILD_TAG}
docker buildx imagetools inspect quay.io/stackrox-io/stackrox-operator:${BUILD_TAG}
docker buildx imagetools inspect quay.io/stackrox-io/roxctl:${BUILD_TAG}
```

Each should show: `linux/amd64`, `linux/arm64`, `linux/ppc64le`, `linux/s390x`

### 2. Both Registries Have Images

```bash
for image in main scanner-v4 stackrox-operator roxctl; do
  echo "Checking ${image}..."
  skopeo inspect docker://quay.io/stackrox-io/${image}:${BUILD_TAG}
  skopeo inspect docker://quay.io/rhacs-eng/${image}:${BUILD_TAG}
done
```

### 3. Images Are Functional

```bash
# Test main image
docker run --rm --platform linux/amd64 quay.io/stackrox-io/main:${BUILD_TAG} central --help

# Test scanner
docker run --rm --platform linux/amd64 quay.io/stackrox-io/scanner-v4:${BUILD_TAG} --help

# Test operator
docker run --rm --platform linux/amd64 quay.io/stackrox-io/stackrox-operator:${BUILD_TAG} --help

# Test roxctl
docker run --rm --platform linux/amd64 quay.io/stackrox-io/roxctl:${BUILD_TAG} version
```

### 4. Layer Sharing Visible in Build Logs

In GitHub Actions build logs, look for:
```
#4 [linux/amd64 1/5] FROM registry.access.redhat.com/ubi9/ubi-micro:latest
#4 CACHED
```

The `CACHED` indicator shows layer reuse across images.

### 5. Build Time Comparison

Compare workflow run times:
- **Before**: Find a recent build on master, note total time for all image builds
- **After**: First build with bake (cold cache), note total time
- **After**: Second build with bake (warm cache), note total time

Expected: 40-60% faster with warm cache.

---

## Rollback Strategy

If the consolidated approach causes issues:

**Quick rollback**:
1. Revert the commit containing the consolidation
2. Push to branch
3. CI will use old separate jobs

**Partial rollback** (if one image is problematic):
1. Keep bake for working images
2. Extract problematic image back to separate job
3. Adjust bake file to remove that target

**Time to rollback**: ~15 minutes (revert + push + CI start)

---

## Common Issues and Solutions

### Issue 1: Binary Paths Incorrect

**Symptom**: Build fails with "COPY failed: file not found"

**Cause**: Dockerfile expects binaries at different path than staging provides

**Solution**: Check binary extraction step - ensure `bin/linux_${arch}/` structure matches Dockerfile `COPY bin/linux_${TARGETARCH}/...`

### Issue 2: Cache Mount Syntax Error

**Symptom**: Dockerfile parsing error

**Cause**: Old Docker version doesn't support cache mounts

**Solution**: Ensure `# syntax=docker/dockerfile:1` is first line of Dockerfile (enables modern features)

### Issue 3: GitHub Actions Cache Full

**Symptom**: Cache save warnings, slow builds

**Cause**: GitHub Actions has 10GB cache limit per repo

**Solution**: Use per-image cache scopes (`components-main`, `components-scanner`, etc.) so Docker evicts old entries automatically

### Issue 4: Bake Target Not Found

**Symptom**: `docker buildx bake: unknown target "main"`

**Cause**: Typo in bake file or wrong target name

**Solution**: Verify target names in bake file match references in workflow job

### Issue 5: Authentication Failure

**Symptom**: "unauthorized: authentication required" during push

**Cause**: Not logged into registry before bake push

**Solution**: Ensure `docker/login-action` step runs before bake command

---

## Implementation Checklist

- [ ] Branch off ROX-34147/extract-db-build
- [ ] Create `.github/docker/components-bake.hcl` with 4 targets
- [ ] Modify `image/rhel/Dockerfile` - add cache mounts, verify TARGETARCH
- [ ] Modify `scanner/image/scanner/Dockerfile` - add cache mounts, add TARGETARCH
- [ ] Modify `operator/prebuilt.Dockerfile` - change TARGET_ARCH to TARGETARCH
- [ ] Modify `image/roxctl/Dockerfile` - add cache mounts (already has TARGETARCH)
- [ ] Add `build-and-push-components-stackrox` job to workflow
- [ ] Add `build-and-push-components-rhacs` job to workflow
- [ ] Remove old jobs: build-and-push-main, scanner, operator, roxctl, manifests
- [ ] Update job dependencies (scan-images-with-roxctl, slack-on-build-failure)
- [ ] Remove matrix definitions from define-job-matrix
- [ ] Update or remove scripts/ci/lib.sh functions
- [ ] Test locally: `docker buildx bake -f .github/docker/components-bake.hcl --print`
- [ ] Create PR with `ci-build-all-arch` label for full platform test
- [ ] Verify multi-arch manifests, both registries, functional images
- [ ] Monitor build times and cache hit rates

---

## Success Criteria

1. **All 4 images build successfully** for all 4 platforms
2. **Both registries** (stackrox-io and rhacs-eng) have all images
3. **Multi-arch manifests** created for all images
4. **Build time reduced** by 40-60% (warm cache)
5. **GitHub Actions cache** shows reuse across images
6. **No functional regressions** - images work as before

---

## References

- **PR #20588**: central-db extraction pattern
- **PR #20617**: operator consolidation (commit 383348efc5)
- **Current roxctl extraction**: commit 5f3f337288
- **Docker Buildx bake docs**: https://docs.docker.com/build/bake/
- **Dockerfile cache mounts**: https://docs.docker.com/build/cache/optimize/#use-cache-mounts

---

## Notes for Agent

- **No prior knowledge assumed**: This spec contains all context needed
- **Exact file paths provided**: Copy-paste file paths as-is
- **Pattern references**: Use PR #20617 and roxctl extraction as templates
- **Verification critical**: Test multi-arch, both registries, functionality
- **Ask for clarification**: If any requirement is ambiguous
- **Incremental commits**: Commit Dockerfiles first, then bake file, then workflow
- **Branch discipline**: Work on ROX-34147/extract-db-build base, not master
