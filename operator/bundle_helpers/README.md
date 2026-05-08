# Bundle Helpers

Go-based tools for managing operator bundle manifests.

## Tools

### fix-spec-descriptor-order
Sorts specDescriptors in CRDs and resolves field dependencies.

### patch-csv
Patches ClusterServiceVersion files with version updates, image replacements,
and related images configuration.

## Usage

### Via dispatch wrapper:
```bash
./dispatch.sh patch-csv [flags] < input.yaml > output.yaml
./dispatch.sh fix-spec-descriptor-order < input.yaml > output.yaml
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
```

## Architecture

- `cmd/` - CLI command implementations
- `pkg/csv/` - CSV patching logic, version handling
- `pkg/descriptor/` - Descriptor sorting and field resolution
- `pkg/rewrite/` - String replacement utilities
- `pkg/values/` - Values handling

## Testing

The Go implementation supports all three related-images modes:
- `omit` - No related images in CSV
- `downstream` - Related images populated from environment variables
- `konflux` - Related images with explicit relatedImages section
