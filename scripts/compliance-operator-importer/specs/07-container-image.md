# 07 - Container Image Packaging

This spec defines how the importer is packaged and distributed as a
multi-architecture container image.

## Design decisions

- Single-stage build using `ubi9-micro` as base (includes CA certificates).
- Multi-arch: `linux/amd64` and `linux/arm64`.
- Static binary: `CGO_ENABLED=0`, pure Go.
- Non-root: runs as UID `65534` (nobody).

## Requirements

### IMP-IMG-001: Dockerfile

The Dockerfile MUST:
- Use `registry.access.redhat.com/ubi9-micro:latest` as base.
- COPY the pre-compiled binary as `/compliance-operator-importer`.
- Set `USER 65534:65534`.
- Set `ENTRYPOINT ["/compliance-operator-importer"]`.

### IMP-IMG-002: Static binary

The Go binary MUST be compiled with:
- `CGO_ENABLED=0`
- `GOOS=linux`
- `GOARCH` set to the target architecture (`amd64` or `arm64`)

### IMP-IMG-003: Multi-architecture support

The build MUST:
- Build the Go binary once per target architecture.
- Build a container image per architecture, tagged with an `-$ARCH` suffix
  (e.g. `$IMAGE:$TAG-amd64`, `$IMAGE:$TAG-arm64`).
- Create a multi-arch manifest list under the plain tag
  (`$IMAGE:$TAG`) combining all architecture-specific images.
- Support at least `linux/amd64` and `linux/arm64`.

### IMP-IMG-004: Build targets

The Makefile MUST provide:
- `make image` — build container image for the host architecture.
- `make image-push` — build and push multi-arch images + manifest.
- Image name configurable via `IMAGE` env var with a placeholder default.
- Tag configurable via `TAG` env var (default: `latest`).

### IMP-IMG-005: Image metadata

The image MUST include OCI labels:
- `org.opencontainers.image.title=co-acs-importer`
- `org.opencontainers.image.description=Compliance Operator to ACS scan configuration importer`
- `org.opencontainers.image.source=https://github.com/stackrox/stackrox`

## Non-goals

- CI/CD pipeline integration (future work).
- Helm chart or operator packaging.
- Signing or SBOM generation (deferred).
