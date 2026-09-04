---
name: create-infra-cluster
description: Create an infra cluster (GKE, OpenShift, etc.) using infractl, download artifacts, and set up KUBECONFIG. Does NOT deploy StackRox - use /run-manual-tests for deployment and testing.
argument-hint: [flavor] [cluster-name]
---

You are an expert at provisioning StackRox test infrastructure. Follow this workflow precisely, pausing for user approval at each checkpoint.

Throughout this workflow, use the correct CLI tool based on the cluster flavor:
- **GKE (`gke-default`)**: use `kubectl`
- **OpenShift (`openshift-4`)**: use `oc`
- **Other flavors**: determine the appropriate CLI based on the flavor type; ask the user if unclear

All examples below use `<kube-cli>` as a placeholder. Substitute with `kubectl` or `oc` as appropriate.

## User Input
${1:User provided: $ARGUMENTS}
${1:else:No arguments provided - you will need to gather requirements interactively.}

## Workflow

### Step 1: Verify INFRA_TOKEN

Check if the `$INFRA_TOKEN` environment variable is set:

```bash
echo "${INFRA_TOKEN:?INFRA_TOKEN is not set}"
```

If not set, stop and ask the user to set it:
```
export INFRA_TOKEN=<your-token>
```
Do NOT proceed without a valid `$INFRA_TOKEN`.

### Step 2: Determine Cluster (Existing or New)

Check if the user specified an **existing cluster** in their prompt. If so, use that cluster and skip to Step 5.

If creating a new cluster, determine the **flavor**:
- If the user specified a flavor (GKE, OpenShift, etc.), use it
- If not specified, ask the user which flavor they want

Check if the user specified a **cluster name**:
- If specified, use that name
- If not, `infractl` will assign a random name

### Step 3: Build the infractl create Command

#### GKE Clusters (flavor: `gke-default`)

Default arguments (override only if user specifies differently):
- `--arg machine-type=e2-standard-8`
- `--arg nodes=3`
- `--arg set-ssd-storage-default=true`

Example:
```bash
infractl create gke-default <cluster-name> \
  --arg machine-type=e2-standard-8 \
  --arg nodes=3 \
  --arg set-ssd-storage-default=true
```

#### OpenShift Clusters (flavor: `openshift-4`)

Default arguments (override only if user specifies differently):
- `--arg master-node-count=3`
- `--arg master-node-type=e2-standard-4`
- `--arg worker-node-type=e2-standard-8`
- `--arg worker-node-count=3`
- `--arg openshift-version=ocp/stable`
- `--arg ssd-storage-class=true`

Example:
```bash
infractl create openshift-4 <cluster-name> \
  --arg master-node-count=3 \
  --arg master-node-type=e2-standard-4 \
  --arg worker-node-type=e2-standard-8 \
  --arg worker-node-count=3 \
  --arg openshift-version=ocp/stable \
  --arg ssd-storage-class=true
```

#### Other Flavors

For any other flavor:
1. Run `infractl flavor list` to find the flavor name
2. Run `infractl flavor get <flavor_name>` to get available arguments and defaults
3. Present the arguments and defaults to the user
4. Let the user customize values before proceeding

You can also use `infractl flavor get gke-default` or `infractl flavor get openshift-4` to look up argument details for GKE/OpenShift if the user requests a customization you don't understand.

### Step 4: Create the Cluster

**CHECKPOINT: Show the generated `infractl create` command to the user and wait for approval before running it.** If the user requests changes, update the command and show it again. Only run the command after explicit approval.

Once approved:
1. Run the `infractl create` command
2. Note the cluster ID/name from the response
3. Run `infractl wait <cluster-name>` in a background session to wait for the cluster to be ready
4. When the cluster is ready, download artifacts:
   ```bash
   mkdir -p tmp/<cluster-name>
   infractl artifacts <cluster-name> -d tmp/<cluster-name>
   ```

### Step 5: Set KUBECONFIG and Verify Context

1. Find the kubeconfig file in the artifacts directory:
   ```bash
   ls tmp/<cluster-name>/kubeconfig
   ```
2. Set the KUBECONFIG environment variable:
   ```bash
   export KUBECONFIG=$(pwd)/tmp/<cluster-name>/kubeconfig
   ```
3. Perform a **three-part verification** to ensure you are targeting the correct cluster:

   **a) Verify `$KUBECONFIG` is set and points to the artifacts directory:**
   ```bash
   echo "$KUBECONFIG"
   # Must point to tmp/<cluster-name>/kubeconfig
   ```

   **b) Verify the cluster name inside the kubeconfig matches the infra cluster:**
   ```bash
   <kube-cli> config get-clusters
   # The cluster name listed must match the infra cluster you created
   ```

   **c) Verify connectivity to the correct cluster:**
   ```bash
   <kube-cli> config current-context
   <kube-cli> cluster-info
   ```

4. Show all three verification outputs to the user for confirmation.

**CHECKPOINT: Do NOT proceed until the user confirms all three checks pass and the context is pointing to the correct cluster.** This is critical to avoid accidentally deploying to or modifying someone else's cluster.

### Step 6: Handoff

Once KUBECONFIG is verified, inform the user:
- Cluster name
- Flavor
- KUBECONFIG path
- The cluster is ready for deployment and testing via `/run-manual-tests`

## Cluster Deletion

**IMPORTANT:** Never automatically delete anything in the `stackrox` namespace or the infra cluster. Cluster deletion is always left to the user. If the user wants to delete the cluster, they must provide explicit step-by-step instructions outside this skill.

## Troubleshooting

- Use `infractl --help` and `infractl <command> --help` for command reference
- Use `infractl get <cluster-name>` to check cluster status
- Use `infractl logs <cluster-name>` to check cluster creation logs
- When in doubt about any argument or customization, always ask the user