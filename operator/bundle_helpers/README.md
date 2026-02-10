# Bundle Helpers

Go-based tools for managing operator bundle manifests.

## Tools

### fix-spec-descriptor-order
Sorts specDescriptors in CRDs and resolves field dependencies.

### patch-csv
Patches ClusterServiceVersion files with version updates, image replacements,
and related images configuration.

## Usage

### Via dispatch wrapper (supports feature flag):
```bash
# Use Go implementation (default)
./dispatch.sh patch-csv [flags] < input.yaml > output.yaml

# Use Python implementation (fallback)
USE_GO_BUNDLE_HELPER=false ./dispatch.sh patch-csv [flags] < input.yaml > output.yaml
```

### Direct Go execution:
```bash
go run ./main.go patch-csv [flags] < input.yaml > output.yaml
go run ./main.go fix-spec-descriptor-order < input.yaml > output.yaml
```

## Development

### Running tests:
```bash
cd operator/bundle_helpers
go test ./...
make golangci-lint
```

### Comparing implementations:
```bash
cd operator
./bundle_helpers/compare-implementations.sh
RELATED_IMAGES_MODE=omit ./bundle_helpers/compare-implementations.sh
RELATED_IMAGES_MODE=downstream ./bundle_helpers/compare-implementations.sh
```

**Note:** For `downstream` and `konflux` modes, you need to set RELATED_IMAGE_* environment variables:
```bash
export RELATED_IMAGE_MAIN=foo \
  RELATED_IMAGE_SCANNER=foo \
  RELATED_IMAGE_SCANNER_SLIM=foo \
  RELATED_IMAGE_SCANNER_DB=foo \
  RELATED_IMAGE_SCANNER_DB_SLIM=foo \
  RELATED_IMAGE_COLLECTOR=foo \
  RELATED_IMAGE_ROXCTL=foo \
  RELATED_IMAGE_CENTRAL_DB=foo \
  RELATED_IMAGE_SCANNER_V4_DB=foo \
  RELATED_IMAGE_SCANNER_V4=foo

RELATED_IMAGES_MODE=downstream ./bundle_helpers/compare-implementations.sh
```

## Architecture

- `cmd/` - CLI command implementations
- `pkg/csv/` - CSV patching logic, version handling
- `pkg/descriptor/` - Descriptor sorting and field resolution
- `pkg/rewrite/` - String replacement utilities
- `pkg/yamlformat/` - YAML formatting for Python compatibility

## Implementation Status

The Go implementation is now the default (`USE_GO_BUNDLE_HELPER` defaults to `true`).
Python scripts are retained for emergency fallback and reference.

### Available Commands
- ✅ `fix-spec-descriptor-order` - Fully implemented in Go
- ✅ `patch-csv` - Fully implemented in Go

### Feature Flag
Set `USE_GO_BUNDLE_HELPER=false` to use the Python implementation:
```bash
USE_GO_BUNDLE_HELPER=false make bundle bundle-post-process
```

## Testing

The Go implementation produces byte-by-byte identical output to the Python implementation
for all three related-images modes:
- `omit` - No related images in CSV
- `downstream` - Related images populated from environment variables
- `konflux` - Related images with explicit relatedImages section

CI automatically validates equivalence between Go and Python implementations.
