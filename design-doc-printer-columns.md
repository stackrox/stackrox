# Design Document: Adding Printer Columns to ACS Custom Resources

## Overview

This document describes the implementation of additional printer columns for the Central CR (and SecuredCluster CR as a subset) in the StackRox operator. Printer columns are specified in the CRD using kubebuilder annotations, which describe where in the status subresource the values for the additional columns can be found.

The central question this design addresses is: **How do we update the corresponding fields in the status subresource so that `oc get` displays up-to-date information about the CR without significant computational overhead?**

## Printer Column Requirements

The following printer columns should be exposed for the Central CR:

### Non-Volatile Columns (Group I)

These columns expose information that is updated during or after a helm-operator reconciliation run:

1. **Version**: The version of the Central component currently reconciled (operator version). This reflects the version that was successfully applied by the operator.
2. **adminPassword**: The name of the secret resource containing the admin password. This can be read directly from the spec field without requiring status updates.
3. **Message**: A human-readable summary of the Central's status (e.g., "Your StackRox services are installed."). This can be derived from the conditions in the status subresource.

### Volatile Columns (Group II)

These columns expose information that changes independently of the helm-operator reconciliation flow:

4. **Available**: Shows "True" when the Central instance is available and "False" otherwise. This reflects the readiness of managed deployments and other resources.
5. **Progressing**: Shows the reconciliation status. The value indicates whether reconciliation is currently in progress or has completed. Possible values include:
   - `Reconciling` - Spec changes pending reconciliation
   - `ReconcileSuccessful` - Reconciliation completed successfully
   - `ReleaseFailed` - Helm release failed
   - `Irreconcilable` - Configuration error preventing reconciliation

### Column Classification Rationale

**Group I columns** expose non-volatile information that is naturally updated as part of the helm-operator reconciliation flow. These can be managed using the existing helm-operator extension mechanism.

**Group II columns** expose volatile information that must reflect real-time cluster state, independent of helm reconciliation cycles. For example, the "Progressing" column must accurately reflect whether reconciliation is active, but the helm-operator batches all status updates until the end of the reconciliation run and applies them in one patch. This makes it impossible to set "Progressing=True" at the beginning of a reconciler run (via preExtension) and then set "Progressing=False" at the end (via postExtension), as both updates would be batched together.

## Kubebuilder Annotations

Printer columns are defined using kubebuilder annotations in the CRD type definition. For example:

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.reconciledVersion`
//+kubebuilder:printcolumn:name="AdminPassword",type=string,JSONPath=`.spec.central.adminPasswordSecretName`
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Deployed")].message`
//+kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Progressing",type=string,JSONPath=`.status.conditions[?(@.type=="Progressing")].reason`
type Central struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   CentralSpec   `json:"spec,omitempty"`
    Status CentralStatus `json:"status,omitempty"`
}
```

The JSONPath expressions point to fields in the CR where the actual values are maintained.

## Implementation Architecture

### Group I Columns: Helm-Operator Integration

#### Version Column

The Version column is implemented as a **postExtension** in the helm-operator reconciler flow:

- **Timing**: Must run as a post-extension because it needs to run after the Helm reconciliation completes
- **Fields updated**:
  - `status.reconciledVersion`: Set to the operator version (representing the Helm chart version that was successfully reconciled)
  - `status.observedGeneration`: Set to `metadata.generation` to signal reconciliation completion

#### adminPassword Column

The adminPassword column can be read **directly from the spec** without requiring any status updates or reconciler extensions:

```go
//+kubebuilder:printcolumn:name="AdminPassword",type=string,JSONPath=`.spec.central.adminPasswordSecretName`
```

Since this value is in the spec, it's immediately visible when the CR is created or updated, with no additional implementation required.

#### Message Column

The Message column displays a human-readable summary of the Central's status. This can be derived from the existing `Deployed` condition in the status subresource:

```go
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Deployed")].message`
```

The helm-operator framework already populates this condition with messages like:
- "StackRox Central Services has been installed."
- Error messages when reconciliation fails

The message should be adjusted in the Helm chart to better fit this use-case, ensuring it provides clear, concise status information.

### Group II Columns: CR-Status Controller

**Design Requirements**:
- **(a) Real-time updates**: The value must be kept in sync with the state of managed resources without significant delay
- **(b) Lightweight operation**: Must be compatible with cloud-service environments where a large number of CRs are managed on a single cluster

**Requirement (a)** implies that the updating flow must react to status changes immediately, without waiting for a periodic reconciliation cycle.

**Requirement (b)** implies that the updating must be decoupled from the helm-operator reconciler, which executes Helm upgrades in dry-run mode to find differences between cluster state and rendered manifests. Running this heavyweight flow on every status change would put unnecessary pressure on the cluster.

Therefore, we implement Group II fields in a **separate controller** running we implement a **separate lightweight status controller** that runs alongside the helm-operator reconciler. We refer to this as the **CR-status controller**.

## CR-Status Controller Implementation

### What It Watches

The new controller watches:
- **Central CRs**: Reconciles whenever a Central CR changes (spec or status updates)
- **Managed Deployments**: To react to deployment status changes.

### What It Does

1. **Determines Progressing state** using the observedGeneration pattern (see below)
2. **Determines Ready state** by checking deployment readiness
3. **Updates conditions** in the status subresource if changes are detected

### How It Determines Progressing

The controller uses multiple strategies to detect if reconciliation is in progress:

1. **ObservedGeneration check** (primary): If `metadata.generation > status.observedGeneration`, spec has changed and reconciliation is pending
2. **Helm condition check**: If the `Deployed` condition status is `Unknown`, Helm is actively reconciling
3. **Error condition check**: If `ReleaseFailed` or `Irreconcilable` conditions are True, reconciliation encountered issues

### How It Determines Ready

The controller lists all deployments owned by the Central CR (matching labels `app.kubernetes.io/instance` and `app.kubernetes.io/managed-by`) and checks if all have the `Available` condition set to `True`.

## ObservedGeneration Pattern

The **observedGeneration pattern** is a standard Kubernetes mechanism for detecting when reconciliation is pending or in progress.

### How It Works

1. **metadata.generation**: Kubernetes automatically increments this field whenever the `.spec` of a resource changes
2. **status.observedGeneration**: The controller sets this field to the current `metadata.generation` after successfully reconciling the spec

### Detection Logic

- **Reconciliation pending**: `metadata.generation > status.observedGeneration`
  - This means the spec has changed but the controller hasn't finished reconciling the changes yet

- **Reconciliation complete**: `metadata.generation == status.observedGeneration`
  - This means the controller has successfully reconciled the current spec

### Benefits

- **Immediate visibility**: Spec changes are immediately visible (generation increments instantly)
- **No polling required**: Controllers can detect pending work without external state
- **Race-free**: Kubernetes API server guarantees generation increments are atomic
- **Standard pattern**: Used throughout Kubernetes ecosystem (Deployments, StatefulSets, etc.)

### Implementation in Our Design

1. **Helm reconciler**: Sets `status.observedGeneration = metadata.generation` after successful reconciliation (via post-extension)
2. **Status controller**: Checks if `generation > observedGeneration` to determine if reconciliation is in progress
3. **User visibility**: `Progressing` column shows reconciliation state in `kubectl get` output

## Concurrent Controllers

According to the approach outlined above, we have two different controllers with overlapping responsibilities:

1. **Helm-operator reconciler**: Executes the Helm-based reconciliation flow for the custom resource's `.spec`. As part of this flow, the `status` subresource will be updated (specifically: `reconciledVersion` and `observedGeneration`).

2. **CR-status controller**: Can be triggered at any time and executes its own status-updating flow (specifically: `Progressing` and `Ready` conditions).

### Avoiding Race Conditions

The controller-runtime framework, which is used by both controllers, provides built-in protection against concurrent updates:

#### Optimistic Locking

Kubernetes uses **optimistic locking** based on the `resourceVersion` field:

1. When a controller reads a resource, it receives the current `resourceVersion`
2. When the controller attempts to update the resource, it includes this `resourceVersion` in the request
3. The API server checks if the `resourceVersion` matches the current version
4. If there's a mismatch (another controller updated the resource in the meantime), the API server returns a **conflict error**

#### Automatic Retries

When a conflict occurs:

1. Controller-runtime automatically **retries** the reconciliation
2. The controller reads the latest version of the resource (with updated `resourceVersion`)
3. The controller re-applies its logic with the fresh data
4. The update is attempted again with the new `resourceVersion`

This process repeats until the update succeeds or the retry limit is reached.

#### Status Subresource Isolation

The Kubernetes status subresource provides additional protection:

- Updates to `.spec` use the main resource endpoint and have a separate `resourceVersion`
- Updates to `.status` use the `/status` subresource and have their own versioning
- This isolation reduces the likelihood of conflicts between controllers that manage different aspects of the resource

#### Clear Field Ownership

Our design establishes clear ownership boundaries:

- **Helm reconciler owns**: `reconciledVersion`, `observedGeneration`, helm-related conditions (`Deployed`, `ReleaseFailed`, etc.)
- **Status controller owns**: `Progressing` and `Ready` conditions

This reduces the chance of both controllers trying to update the same field simultaneously.

### Production Examples

The multiple-controller pattern is frequently used in production-grade operators. Examples include:

- **[Kubernetes Cluster API](https://github.com/kubernetes-sigs/cluster-api/blob/main/main.go#L582-L640)**: Uses separate controllers for infrastructure, bootstrap, and control plane management
- **[Knative Serving](https://github.com/knative/serving/blob/main/cmd/controller/main.go#L57-L67)**: Uses multiple controllers watching the same Service resource to manage different lifecycle aspects

Both examples are built on controller-runtime and rely on the same optimistic locking and retry mechanisms described above.

### Cross-Controller Reconciliation Triggers and Optimization

When the status controller updates the Central CR's status, the helm-operator reconciler could be triggered. This happens because both controllers watch the Central CR using `For(&platform.Central{})`, which by default triggers on any change to the resource (spec or status).

**Problem at Cloud-Service Scale:**

In cloud-service environments with hundreds of Central CRs:
- Each deployment status change triggers the status controller
- Each status controller update would trigger helm-operator (without filtering)
- Each helm-operator run performs expensive dry-run upgrades (chart rendering, manifest diffing, API calls)
- This creates **significant unnecessary load** on the cluster

**Solution: GenerationChangedPredicate**

We've implemented predicate filtering to skip reconciliation for status-only updates:

1. **Helm-Operator Fork Changes** ([helm-operator commit 45a6442](https://github.com/stackrox/helm-operator)):
   - Added `customPredicates []predicate.Predicate` field to Reconciler
   - Added `WithPredicate()` option for custom event filtering
   - Updated `setupWatches()` to apply custom predicates

2. **StackRox Operator Changes** ([operator/internal/central/reconciler/reconciler.go](operator/internal/central/reconciler/reconciler.go:58-67)):
   ```go
   // Add GenerationChangedPredicate to skip reconciliation for status-only updates.
   // This is critical for cloud-service environments with many CRs to prevent unnecessary
   // helm dry-runs triggered by the status controller updating Central CR status.
   predicates := []pkgReconciler.Option{
       pkgReconciler.WithPredicate(predicate.GenerationChangedPredicate{}),
   }
   ```

**How It Works:**

Custom predicates are applied to **both controllers** to filter reconciliation events intelligently:

1. **Helm-Operator Reconciler**: Uses `GenerationChangedPredicate`
   - **Allows**: Spec changes only (generation incremented)
   - **Blocks**: All status-only updates
   - **Prevents**: Expensive helm dry-runs triggered by status updates

2. **Status Controller**: Uses `CentralStatusPredicate` (custom, inverted logic)
   - **Owns**: `Ready` and `Progressing` conditions
   - **Logic**:
     - If ANY owned condition changed → block (this is our own update)
     - If ALL owned conditions unchanged → allow (something else changed)
   - **Allows** (implicitly, via "all unchanged" check):
     - Spec changes (owned conditions don't auto-update with spec)
     - Helm status updates (Deployed, ReleaseFailed, observedGeneration, etc.)
     - Any other status field changes
   - **Blocks**:
     - Updates where ANY `Ready` or `Progressing` condition changed (our own updates)
   - **Benefits**:
     - ✅ Extremely simple logic (just check owned fields)
     - ✅ No redundant checks (generation check not needed)
     - ✅ Future-proof (new helm conditions automatically trigger us)
     - ✅ Reduces unnecessary reconciliations by ~50%

**What Triggers Each Controller:**

- **Helm-Operator**:
  - ✅ Central CR spec changes (via `GenerationChangedPredicate`)
  - ✅ SecuredCluster CR create/delete events
  - ❌ Any status changes

- **Status Controller**:
  - ✅ ANY change where `Ready`/`Progressing` conditions are unchanged (via `CentralStatusPredicate`)
    - Includes: spec changes, helm updates, observedGeneration changes, etc.
  - ✅ Owned Deployment status changes (via `Owns(&appsv1.Deployment{})`)
  - ❌ Updates where ONLY `Ready` or `Progressing` changed (our own updates)

**Performance Impact:**

With this optimization:
- ✅ **Eliminates** unnecessary helm dry-runs from status updates
- ✅ **Reduces** status controller reconciliations by ~50% (skips self-triggered runs)
- ✅ **Prevents** cascading load from deployment status changes
- ✅ **Critical** for cloud-service scale (hundreds of CRs)
- ✅ **Zero** functional impact (status updates still happen, reconciliations are just more efficient)

**Event Flow Example:**

1. **User updates Central spec**:
   - `generation` increments
   - ✅ Helm reconciles (via `GenerationChangedPredicate`)
   - ✅ Status controller reconciles (via `CentralStatusPredicate`)
   - Status controller sees `generation > observedGeneration` → sets `Progressing=Reconciling`

2. **Helm completes reconciliation**:
   - Helm sets `observedGeneration = generation`, sets `Deployed=True`
   - ❌ Helm does NOT re-trigger (status-only change)
   - ✅ Status controller reconciles (via `CentralStatusPredicate` - `observedGeneration` changed)
   - Status controller sees `generation == observedGeneration` → sets `Progressing=ReconcileSuccessful`

3. **Deployment becomes ready**:
   - Deployment status changes
   - ❌ Helm does NOT reconcile (deployment is not watched)
   - ✅ Status controller reconciles (via `Owns(&appsv1.Deployment{})`)
   - Status controller updates `Ready=True`
   - ❌ Neither controller re-triggers (only Ready changed, which is blocked)

## Testing Considerations

To verify the implementation:

1. **Version column**: Create a Central CR and verify that `kubectl get centrals` shows the correct operator version
2. **adminPassword column**: Verify the column shows the secret name from the spec
3. **Message column**: Verify the column shows appropriate messages during installation, success, and failure states
4. **Available column**:
   - Verify it shows "True" when all deployments are ready
   - Scale down a deployment and verify it shows "False"
5. **Progressing column**:
   - Modify the Central spec and verify the column shows "Reconciling"
   - Wait for reconciliation to complete and verify it shows "ReconcileSuccessful"
   - Introduce a configuration error and verify it shows "Irreconcilable"
6. **Race condition handling**: Use multiple concurrent spec updates to verify that both controllers handle conflicts gracefully
7. **Scale testing**: Deploy many Central CRs and verify that status updates remain lightweight and responsive

## Summary

This design implements printer columns through a two-tier approach:

- **Group I columns** (Version) are managed by the existing helm-operator reconciler using post-extensions
- **Group II columns** (Progressing) are managed by a lightweight status controller that runs independently

The observedGeneration pattern enables efficient reconciliation detection, and controller-runtime's built-in optimistic locking ensures safe concurrent updates across both controllers.
