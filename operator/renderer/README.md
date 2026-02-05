# StackRox CR Renderer

This tool demonstrates how the StackRox Operator's reconcilers work by:

1. Starting an envtest Kubernetes environment
2. Loading the Central and/or SecuredCluster CRDs
3. Starting the appropriate StackRox operator reconciler(s)
4. Creating required namespaces
5. Applying the provided Custom Resource(s)
6. Waiting for the reconciler(s) to process the CR(s) by watching their status
7. Capturing and filtering all resources created by the reconciler(s)
8. Writing the reconciler-created resources to organized YAML files

## Usage

### Option 1: Using the Makefile target (Recommended)

```bash
# From the operator directory, run the target which handles all setup
cd operator
make run-central-renderer
```

This target automatically:
- Sets up all prerequisites (manifests, code generation, formatting, vetting)
- Downloads and configures the required envtest binaries
- Builds the central-renderer binary
- Runs it with the example Central CR and proper environment setup

### Option 2: Manual execution

```bash
# Build the tool
go build -o central-renderer ./operator/renderer

# Run with an example Central CR
./central-renderer --central-cr ./operator/renderer/example-central.yaml --verbose

# Run with an example SecuredCluster CR
./central-renderer --securedcluster-cr ./operator/renderer/example-securedcluster.yaml --verbose

# Run with both Central and SecuredCluster CRs
./central-renderer --central-cr ./operator/renderer/example-central.yaml --securedcluster-cr ./operator/renderer/example-securedcluster.yaml --verbose

# Run with custom timeout
./central-renderer --central-cr ./path/to/your/central.yaml --timeout 10m
```

**Note**: Manual execution requires that you have already set up envtest binaries and the `KUBEBUILDER_ASSETS` environment variable.

## Command Line Options

- `--central-cr`: Path to Central CR YAML file (optional, but at least one of --central-cr or --securedcluster-cr is required)
- `--securedcluster-cr`: Path to SecuredCluster CR YAML file (optional, but at least one of --central-cr or --securedcluster-cr is required)
- `--timeout`: Maximum time to wait for reconciliation (default: 5m)
- `--verbose`: Enable verbose logging including control plane output

## Example Custom Resources

See `example-central.yaml` for a basic Central CR and `example-securedcluster.yaml` for a basic SecuredCluster CR that can be used for testing.

## What the Tool Does

1. **Environment Setup**: Creates an envtest environment with the Central and/or SecuredCluster CRDs loaded
2. **Reconciler Registration**: Registers the appropriate reconciler(s) used by the actual operator
3. **Baseline Capture**: Captures all existing resources before applying any CRs
4. **CR Application**: Creates required namespaces and applies the provided Custom Resource(s)
5. **Status Monitoring**: Watches the CR status(es) for completion or error conditions
6. **Resource Discovery**: Uses the Kubernetes discovery API to find all resources created during reconciliation
7. **Resource Filtering**: Filters out baseline resources to identify only reconciler-created resources
8. **Resource Output**: Writes organized YAML files containing the reconciler-created resources

## Output

The tool will:
- Print progress messages about the reconciliation process
- Create a `reconciler-output/` directory containing YAML files organized by API group
- Each YAML file contains all resources of that type created by the reconciler(s)
- Strip `metadata.managedFields` from saved resources for cleaner output
- Include both cluster-scoped and namespaced resources

This provides insight into exactly what resources the StackRox reconcilers create when processing Custom Resources, with all manifests saved for further analysis or comparison.

## Requirements

- Go 1.24.0+
- Access to the StackRox codebase (for CRD definitions and reconciler code)
- Sufficient system resources to run an embedded Kubernetes control plane (envtest)
- **Kubebuilder test binaries** (etcd, kube-apiserver) - Install with:
  ```bash
  # Option 1: Use the StackRox development environment setup
  make install-dev-tools

  # Option 2: Install kubebuilder directly
  curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
  chmod +x kubebuilder && sudo mv kubebuilder /usr/local/bin/
  ```

## Setup

1. Ensure you have the required kubebuilder test binaries installed
2. Build the tool:
   ```bash
   go build -o central-renderer ./operator/renderer
   ```
3. Run with an example Central CR:
   ```bash
   ./central-renderer --central-cr ./operator/renderer/example-central.yaml --verbose
   ```

## Limitations

- The tool uses envtest, so it doesn't actually deploy running containers
- Some resources that depend on cluster-specific features may not be fully functional
- The reconciler may not complete all operations that require a real cluster (e.g., actual workload deployments)
- Requires kubebuilder test binaries to be available in PATH or `/usr/local/kubebuilder/bin/`