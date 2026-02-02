# CSV Patching Tools

This document describes the Go-based tools for patching Operator Lifecycle Manager (OLM) ClusterServiceVersion (CSV) files during the bundle build process.

## Overview

The operator bundle build process uses two Go tools to prepare CSV files for release:

1. **csv-patcher** - Main tool that updates versions, images, and metadata in CSV files
2. **fix-spec-descriptors** - Helper tool that fixes ordering and field dependencies in CRD spec descriptors

These tools replaced the previous Python-based implementation to improve maintainability, reduce dependencies, and provide better integration with the existing Go codebase.

## Tools

### csv-patcher

The `csv-patcher` tool reads a CSV YAML file from stdin, applies various patches based on command-line flags, and outputs the modified CSV to stdout.

#### Usage

```bash
csv-patcher \
  --use-version <version> \
  --first-version <first-version> \
  --operator-image <image> \
  [--related-images-mode <mode>] \
  [--add-supported-arch <archs>] \
  [--unreleased <version>] \
  [--echo-replaced-version-only] \
  < input.yaml > output.yaml
```

#### Flags

- `--use-version` (required) - SemVer version of the operator (e.g., "4.2.3")
- `--first-version` (required) - First version of operator ever published (e.g., "3.62.0")
- `--operator-image` (required) - Operator image reference (e.g., "quay.io/stackrox-io/stackrox-operator:4.2.3")
- `--related-images-mode` - Mode for handling related images (default: "downstream")
  - `downstream` - Remove relatedImages section, inject RELATED_IMAGE_* env vars
  - `omit` - Remove relatedImages section, inject RELATED_IMAGE_* env vars
  - `konflux` - Build relatedImages section from RELATED_IMAGE_* env vars
- `--add-supported-arch` - Comma-separated list of supported architectures (default: "amd64,arm64,ppc64le,s390x")
- `--unreleased` - Not yet released version, used to skip unreleased versions in upgrade path
- `--echo-replaced-version-only` - Only compute and print the replaced version, don't patch CSV

#### What It Does

The csv-patcher performs the following operations:

1. **Updates metadata.name** - Changes from placeholder `rhacs-operator.v0.0.1` to `rhacs-operator.v<version>`
2. **Updates spec.version** - Sets to the specified version
3. **Updates createdAt timestamp** - Sets to current UTC time
4. **Replaces operator image** - Replaces placeholder image with actual operator image reference
5. **Calculates and sets replaces field** - Determines which previous version this release replaces (see Version Calculation below)
6. **Sets olm.skipRange** - Allows OLM to skip intermediate versions during upgrade
7. **Handles related images** - Based on mode, either removes relatedImages or constructs it from environment variables
8. **Adds multi-arch labels** - Adds `operatorframework.io/arch.<arch>: supported` labels
9. **Adds SecurityPolicy CRD** - Injects the SecurityPolicy CRD into the owned CRDs list

#### Version Calculation Logic

The tool implements complex logic to determine the `spec.replaces` field, which tells OLM which previous version this release replaces.

**Y-Stream vs Patch Releases:**

- **Y-Stream release** (patch version = 0, e.g., 4.2.0): Replaces the previous minor version (e.g., 4.1.0)
- **Patch release** (patch version > 0, e.g., 4.2.3): Replaces the previous patch (e.g., 4.2.2)

**Previous Y-Stream Calculation:**

The tool calculates the previous Y-Stream to determine the `olm.skipRange`:

- If minor version > 0: previous Y-Stream is `<major>.<minor-1>.0`
- If minor version = 0 (major version bump): uses hardcoded mapping
  - 4.0.0 → 3.74.0
  - 1.0.0 → 0.0.0

**Skipped Versions:**

The CSV can include a `spec.skips` list for broken versions that should be skipped during upgrades. The replace version calculation will skip over these versions:

```
Example: If 4.2.1 is broken and in skips, then:
- 4.2.2 replaces 4.2.0 (skips over 4.2.1)
- 4.2.3 replaces 4.2.2 (normal progression)

Exception: Immediate fix still replaces broken version
- 4.2.1 is broken and in skips
- 4.2.2 (immediate fix) still replaces 4.2.1
- This works because skipRange allows upgrade from 4.1.0
```

**Unreleased Versions:**

The `--unreleased` flag handles the case where the calculated replace version hasn't been released yet. If the initial replace candidate matches the unreleased version, the tool falls back to the previous Y-Stream.

**First Version:**

The first version ever released gets no `spec.replaces` field, as there's nothing to replace.

#### Environment Variables

When `--related-images-mode` is not "omit", the tool requires environment variables for all related images:

- `RELATED_IMAGE_MAIN` - Main StackRox image
- `RELATED_IMAGE_CENTRAL_DB` - Central database image
- `RELATED_IMAGE_SCANNER` - Scanner image
- `RELATED_IMAGE_SCANNER_DB` - Scanner database image
- `RELATED_IMAGE_SCANNER_V4_DB` - Scanner V4 database image
- etc.

These are injected as `value` fields in the deployment's env vars section.

For `konflux` mode, these environment variables are also used to build the `spec.relatedImages` section.

### fix-spec-descriptors

The `fix-spec-descriptors` tool fixes issues with CRD spec descriptors in the CSV file to ensure proper rendering in the OpenShift console.

#### Usage

```bash
fix-spec-descriptors < input.yaml > output.yaml
```

This tool takes no command-line arguments. It reads CSV YAML from stdin and writes the fixed version to stdout.

#### What It Does

1. **Fixes descriptor ordering** - Performs stable sort so parent fields appear before children
   - Example: `central` must come before `central.db` which must come before `central.db.enabled`
   - This ensures proper rendering in the OpenShift console UI
2. **Converts relative field dependencies to absolute** - Transforms relative paths (starting with `.`) in `fieldDependency` x-descriptors to absolute paths
   - Example: `urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true` becomes `urn:alm:descriptor:com.tectonic.ui:fieldDependency:central.db.enabled:true`

## Build Integration

These tools are integrated into the Makefile bundle build process:

### Building the Tools

```bash
# Build both tools
make csv-patcher fix-spec-descriptors

# Build individual tools
make csv-patcher
make fix-spec-descriptors
```

The built binaries are placed in `bin/csv-patcher` and `bin/fix-spec-descriptors`.

### Bundle Build Process

The tools are used in two phases:

**Phase 1: Bundle Generation (`make bundle`)**
```bash
# After operator-sdk generates the initial bundle
$(FIX_SPEC_DESCRIPTORS) \
  < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
  > bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed
mv bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed \
   bundle/manifests/rhacs-operator.clusterserviceversion.yaml
```

**Phase 2: Bundle Post-Processing (`make bundle-post-process`)**
```bash
# First, check if the candidate replace version has been released
candidate_version=$($(CSV_PATCHER) \
  --use-version $(VERSION) \
  --first-version 3.62.0 \
  --operator-image $(IMG) \
  --echo-replaced-version-only \
  < bundle/manifests/rhacs-operator.clusterserviceversion.yaml)

# If not released, add --unreleased flag
if ! image_exists $candidate_version; then
  unreleased_opt="--unreleased=$candidate_version"
fi

# Apply all patches
$(CSV_PATCHER) \
  --use-version $(VERSION) \
  --first-version 3.62.0 \
  --operator-image $(IMG) \
  --related-images-mode $(RELATED_IMAGES_MODE) \
  --add-supported-arch amd64,arm64,ppc64le,s390x \
  $unreleased_opt \
  < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
  > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run csv-patcher tests
cd cmd/csv-patcher && go test -v ./...

# Run fix-spec-descriptors tests
cd cmd/fix-spec-descriptors && go test -v ./...
```

### Adding New Features

When adding new CSV patching features:

1. Add the logic to `cmd/csv-patcher/patch.go`
2. Add command-line flags in `cmd/csv-patcher/main.go` if needed
3. Add tests in `cmd/csv-patcher/*_test.go`
4. Update this documentation

For descriptor fixes:

1. Modify `cmd/fix-spec-descriptors/main.go`
2. Add tests in `cmd/fix-spec-descriptors/main_test.go`
3. Update this documentation

### Code Structure

**csv-patcher:**
- `main.go` - CLI entry point, flag parsing, I/O handling
- `version.go` - Version types and Y-Stream/replace calculation logic
- `patch.go` - Main patching logic and CSV manipulation
- `stringreplace.go` - Recursive string replacement utility
- `crd.go` - CRD structure types for type-safe manipulation

**fix-spec-descriptors:**
- `main.go` - All logic in a single file (simpler tool)

## Migration from Python

These Go tools replaced the previous Python-based implementation located in `scripts/csv-bundle-helpers/`:

**Advantages of Go implementation:**

- No Python dependency or version management needed
- Better integration with Go-based build system
- Faster execution
- Type safety for CSV structure manipulation
- Easier to maintain alongside operator Go code
- Consistent tooling across the project

**What was removed:**

- `scripts/csv-bundle-helpers/patch-csv.py`
- `scripts/csv-bundle-helpers/fix-descriptors.py`
- Python version requirement in documentation

The Python scripts and their dependencies were fully removed as part of the migration.

## Troubleshooting

### Common Issues

**Error: "required environment variable RELATED_IMAGE_* is not set"**

Solution: Make sure all required RELATED_IMAGE_* environment variables are set before running csv-patcher with `--related-images-mode` not set to "omit".

**Error: "don't know the previous Y-Stream for X.Y"**

Solution: For new major version bumps, update the hardcoded mapping in `GetPreviousYStream()` in `version.go`.

**Error: "metadata.name does not end with .v0.0.1"**

Solution: The input CSV template must have `metadata.name` ending with `.v0.0.1` as a placeholder. Check your CSV template generation.

**Console UI shows fields in wrong order**

Solution: Make sure `fix-spec-descriptors` is run on the CSV before validation. The tool should be run as part of `make bundle`.

### Debugging

To see what the csv-patcher would do without applying changes:

```bash
# Just calculate and print the replaced version
csv-patcher \
  --use-version 4.2.3 \
  --first-version 3.62.0 \
  --operator-image quay.io/stackrox-io/stackrox-operator:4.2.3 \
  --echo-replaced-version-only \
  < bundle/manifests/rhacs-operator.clusterserviceversion.yaml
```

To inspect CSV changes:

```bash
# Before
cat bundle/manifests/rhacs-operator.clusterserviceversion.yaml > /tmp/before.yaml

# After
csv-patcher [flags] < /tmp/before.yaml > /tmp/after.yaml

# Compare
diff -u /tmp/before.yaml /tmp/after.yaml
```

## References

- [Operator SDK Bundle Documentation](https://sdk.operatorframework.io/docs/olm-integration/generation/#bundle)
- [OLM ClusterServiceVersion Spec](https://olm.operatorframework.io/docs/concepts/crds/clusterserviceversion/)
- [Semantic Versioning](https://semver.org/)
- Operator EXTENDING_CRDS.md (CRD development guide)
- Operator DEFAULTING.md (CRD defaulting mechanisms)
