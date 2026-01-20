# ROX-27912: Restructure and Simplify LookupOrCreateClusterFromConfig Logic

## Problem Statement

The function `LookupOrCreateClusterFromConfig` in `central/cluster/datastore/datastore_impl.go` (lines 894-1014) has become increasingly complex with multiple responsibilities, side effects, and no transactional semantics. This makes it difficult to understand, test, and maintain.

## Current Analysis

### Function Location
- File: `central/cluster/datastore/datastore_impl.go`
- Lines: 894-1014 (121 lines)
- Interface: Part of the `DataStore` interface

### Current Responsibilities (10 distinct concerns)

1. **Security Access Control (SAC) checking**
   - Validates write permissions for the cluster

2. **Configuration extraction**
   - Extracts Helm config and manager type from SensorHello message

3. **Mutual exclusion (locking)**
   - Acquires datastore lock for the entire operation

4. **Cluster name resolution**
   - Extracts cluster name from config
   - Looks up cluster ID by name from cache if not provided

5. **Conditional cluster lookup/creation**
   - Three paths:
     - If cluster ID exists: Lookup and validate name match
     - If cluster name exists but no ID: Create new cluster
     - Otherwise: Return error

6. **Grace period enforcement**
   - For Helm/Operator managed clusters only
   - Prevents cluster moves within 3-minute grace period
   - Checks sensor deployment identification

7. **Short-circuit optimization**
   - Early return if cluster hasn't changed (fingerprint, capabilities, bundle ID, manager type)

8. **Cluster configuration update**
   - Updates manager type, init bundle ID, sensor capabilities
   - Configures from Helm config or clears it for manual management

9. **Conditional database update**
   - Only writes to DB if cluster is "dirty" (has changes)

10. **Return result**
    - Returns the updated or created cluster

### Side Effects

1. **Database reads**
   - `ds.GetCluster()` - reads cluster by ID
   - `ds.nameToIDCache.Get()` - cache lookup

2. **Database writes**
   - `ds.addClusterNoLock()` - creates cluster, network flow store, updates cache
   - `ds.updateClusterNoLock()` - updates cluster, validates, persists to DB

3. **Cache operations**
   - Updates `nameToIDCache` and `idToNameCache` via nested calls

4. **Metrics/tracking**
   - `trackClusterRegistered()` called during cluster creation

5. **Network flow store creation**
   - Creates network flow store for new clusters

### Complexity Issues

1. **Function length**: 121 lines with deeply nested conditionals
2. **Mixed concerns**: Validation, lookup, creation, update all in one function
3. **Different code paths**: New vs. existing, Helm/Operator vs. Manual
4. **Hard to test**: Testing requires full datastore setup
5. **No transactional semantics**: Multiple DB operations without rollback capability
6. **Error handling**: Scattered throughout, hard to reason about failure modes

### Test Coverage

Existing test: `TestLookupOrCreateClusterFromConfig` in `datastore_impl_postgres_test.go` (lines 241-487)
- Covers 11 test cases
- Tests various scenarios: name mismatch, ID not found, namespace changes, config updates, capabilities
- Integration test (requires full datastore setup)

## Proposed Solution

### Design Goals

1. **Separation of Concerns**: Extract distinct responsibilities into separate functions
2. **Testability**: Each extracted function should be independently testable
3. **Readability**: Clear, linear flow with minimal nesting
4. **Maintainability**: Easy to understand and modify individual pieces
5. **Backwards Compatibility**: No changes to public API or behavior
6. **Transaction Safety**: Consider adding transactional semantics where appropriate

### Restructuring Strategy

#### Phase 0: Add Helper Type

Define a struct to hold extracted cluster configuration data:

```go
// clusterConfigData holds extracted configuration from SensorHello.
// This allows pure functions to work with structured data instead of raw protobuf messages.
type clusterConfigData struct {
    helmManagedConfigInit    *central.HelmManagedConfigInit
    deploymentIdentification *storage.SensorDeploymentIdentification
    capabilities             []string
}
```

Helper methods for convenient access:
```go
func (c clusterConfigData) clusterName() string {
    return c.helmManagedConfigInit.GetClusterName()
}

func (c clusterConfigData) manager() storage.ManagerType {
    return c.helmManagedConfigInit.GetManagedBy()
}

func (c clusterConfigData) helmConfig() *storage.CompleteClusterConfig {
    return c.helmManagedConfigInit.GetClusterConfig()
}
```

#### Phase 1: Extract Helper Functions (Pure Logic)

Create pure functions (no side effects) for:

1. **`extractClusterConfig(hello *central.SensorHello) clusterConfigData`**
   - Extracts relevant data from SensorHello message
   - Returns struct with helmManagedConfigInit, deploymentIdentification, and capabilities
   - Pure function, easy to unit test

2. **`shouldUpdateCluster(existing *storage.Cluster, config clusterConfigData, registrantID string) (bool, string)`**
   - Determines if cluster needs updating
   - Returns: (needsUpdate bool, reason string)
   - Checks: sensor capabilities, fingerprint, init bundle ID, manager type
   - Pure function based on existing cluster state

3. **`validateClusterConfig(clusterID, clusterName string, existing *storage.Cluster) error`**
   - Validates cluster configuration consistency
   - Checks name match for existing clusters
   - Returns descriptive errors
   - Pure validation logic

4. **`buildClusterFromConfig(clusterName, registrantID string, config clusterConfigData) *storage.Cluster`**
   - Builds new cluster object from configuration
   - Does not persist, just constructs the object
   - Pure function (no SensorHello dependency, uses extracted config)

5. **`applyConfigToCluster(cluster *storage.Cluster, config clusterConfigData, registrantID string) *storage.Cluster`**
   - Applies configuration updates to cluster (creates clone)
   - Updates: manager, init bundle, capabilities, helm config
   - Returns new cluster object (immutable pattern)
   - Pure function

#### Phase 2: Extract Side-Effect Functions

Create functions with controlled side effects:

1. **`checkGracePeriodForReconnect(cluster *storage.Cluster, deploymentID *storage.SensorDeploymentIdentification, manager storage.ManagerType) error`**
   - Checks if reconnection is allowed based on grace period
   - Returns descriptive error if not allowed
   - Only needs deploymentID from the config, not the entire SensorHello message
   - Encapsulates grace period logic

2. **`lookupOrCreateCluster(ctx context.Context, clusterID, clusterName, registrantID string, config clusterConfigData) (*storage.Cluster, bool, error)`**
   - Handles the lookup-or-create logic
   - Returns: (cluster, wasExistingCluster, error)
   - The bool is semantically meaningful:
     - `true` = found existing cluster (need to check grace period, check if update needed)
     - `false` = created new cluster (skip those checks)
   - Uses extracted config instead of raw SensorHello
   - Encapsulates three code paths: lookup by ID, create by name, or error

#### Phase 3: Refactor Main Function

Restructure `LookupOrCreateClusterFromConfig` to be a coordinator:

```go
func (ds *datastoreImpl) LookupOrCreateClusterFromConfig(ctx context.Context, clusterID, registrantID string, hello *central.SensorHello) (*storage.Cluster, error) {
    // 1. SAC check (unchanged)
    if err := checkWriteSac(ctx, clusterID); err != nil {
        return nil, err
    }

    // 2. Extract configuration (pure function)
    config := extractClusterConfig(hello)

    // 3. Lock for database operations
    ds.lock.Lock()
    defer ds.lock.Unlock()

    // 4. Lookup or create cluster
    cluster, isExisting, err := ds.lookupOrCreateCluster(ctx, clusterID, config.clusterName(), registrantID, config)
    if err != nil {
        return nil, err
    }

    // 5. For existing clusters, check if update is needed
    if isExisting && config.manager() != storage.ManagerType_MANAGER_TYPE_MANUAL {
        // Check grace period
        if err := checkGracePeriodForReconnect(cluster, config.deploymentIdentification, config.manager()); err != nil {
            return nil, err
        }

        // Check if update needed
        needsUpdate, _ := shouldUpdateCluster(cluster, config, registrantID)
        if !needsUpdate {
            return cluster, nil // Short-circuit
        }
    }

    // 6. Apply configuration updates
    updatedCluster := applyConfigToCluster(cluster, config, registrantID)

    // 7. Persist if changed
    if !cluster.EqualVT(updatedCluster) {
        if err := ds.updateClusterNoLock(ctx, updatedCluster); err != nil {
            return nil, err
        }
    }

    return updatedCluster, nil
}
```

#### Phase 4: Implementation Details for lookupOrCreateCluster

```go
// lookupOrCreateCluster handles the lookup-or-create logic.
// Returns the cluster, a bool indicating whether it was an existing cluster (true) or newly created (false), and an error.
// The bool is important because existing clusters need grace period checks and update checks, while new clusters skip those.
func (ds *datastoreImpl) lookupOrCreateCluster(ctx context.Context, clusterID, clusterName, registrantID string, config clusterConfigData) (*storage.Cluster, bool, error) {
    // Try to resolve cluster ID from name if not provided
    if clusterID == "" && clusterName != "" {
        if cachedID, ok := ds.nameToIDCache.Get(clusterName); ok {
            clusterID = cachedID.(string)
        }
    }

    // Path 1: Lookup existing cluster by ID
    if clusterID != "" {
        cluster, exists, err := ds.GetCluster(ctx, clusterID)
        if err != nil {
            return nil, false, err
        }
        if !exists {
            return nil, false, errors.Errorf("cluster with ID %q does not exist", clusterID)
        }

        // Validate name match
        if err := validateClusterConfig(clusterID, clusterName, cluster); err != nil {
            return nil, false, err
        }

        return cluster, true, nil
    }

    // Path 2: Create new cluster by name
    if clusterName != "" {
        // Use the config that was already extracted in the main function
        cluster := buildClusterFromConfig(clusterName, registrantID, config)

        if _, err := ds.addClusterNoLock(ctx, cluster); err != nil {
            return nil, false, errors.Wrapf(err, "failed to dynamically add cluster with name %q", clusterName)
        }

        return cluster, false, nil
    }

    // Path 3: Neither ID nor name provided
    return nil, false, errors.New("neither a cluster ID nor a cluster name was specified")
}
```

### Implementation Plan

#### Step 1: Add Helper Type
- Add `clusterConfigData` struct to hold extracted configuration
- Add helper methods: `clusterName()`, `manager()`, `helmConfig()`
- Location: Near `LookupOrCreateClusterFromConfig`

#### Step 2: Implement Pure Functions
- `extractClusterConfig()`
- `shouldUpdateCluster()`
- `validateClusterConfig()`
- `buildClusterFromConfig()`
- `applyConfigToCluster()`

#### Step 3: Add Unit Tests for Pure Functions
- Create `datastore_impl_helpers_test.go` for unit tests
- Test each pure function independently
- No database required for these tests

#### Step 4: Implement Side-Effect Functions
- `checkGracePeriodForReconnect()`
- `lookupOrCreateCluster()` (internal helper with lookup-or-create logic)

#### Step 5: Refactor Main Function
- Update `LookupOrCreateClusterFromConfig` to use new helpers
- Ensure same behavior as before
- Keep existing integration test passing

#### Step 6: Add Additional Tests
- Add unit tests for new functions
- Consider adding more integration test cases if needed

### Files to Modify

1. **`central/cluster/datastore/datastore_impl.go`**
   - Add helper type `clusterConfigData`
   - Add pure helper functions
   - Add side-effect helper functions
   - Refactor `LookupOrCreateClusterFromConfig`

2. **`central/cluster/datastore/datastore_impl_test.go`** (if exists, or create it)
   - Add unit tests for pure functions
   - These don't require database

3. **`central/cluster/datastore/datastore_impl_postgres_test.go`**
   - Ensure existing test still passes
   - Potentially add more test cases for edge cases

### Testing Strategy

1. **Unit Tests** (new)
   - Test pure functions: `extractClusterConfig`, `shouldUpdateCluster`, `validateClusterConfig`, `buildClusterFromConfig`, `applyConfigToCluster`
   - No database required
   - Fast, focused tests

2. **Integration Tests** (existing)
   - Keep existing `TestLookupOrCreateClusterFromConfig`
   - Ensure all existing test cases still pass
   - Add new test cases for edge cases if discovered

3. **Manual Testing**
   - Deploy locally and test cluster registration
   - Test init-bundle flow
   - Test Helm-managed cluster updates
   - Test Operator-managed cluster updates

### Benefits

1. **Easier to Understand**
   - Main function is now ~30 lines instead of 121
   - Clear, linear flow
   - Each helper has a single responsibility

2. **Easier to Test**
   - Pure functions can be unit tested without database
   - Integration tests focus on coordination logic
   - Better test coverage of edge cases

3. **More Maintainable**
   - Changes to specific logic (e.g., grace period) are localized
   - Adding new validation rules is easier
   - Debugging is simpler with smaller functions

4. **Potentially More Reliable**
   - Clearer error handling
   - Easier to reason about failure modes
   - Foundation for adding transactional semantics later

### Future Enhancements (Out of Scope)

1. **Transactional Semantics**
   - Wrap database operations in transaction
   - Rollback on failure
   - Requires database transaction support

2. **Further Decomposition**
   - Extract grace period logic to separate service
   - Extract cluster validation to validator service
   - Move towards domain-driven design

3. **Metrics and Observability**
   - Add structured logging for each step
   - Add metrics for update reasons
   - Trace cluster lifecycle events

## Risk Assessment

### Low Risk
- Pure function extraction: No behavior change
- Adding unit tests: Only improves coverage
- Code is well-tested with existing integration tests

### Medium Risk
- Refactoring main function: Could introduce subtle bugs
- Mitigation: Keep existing integration test, manual testing

### High Risk
- None identified

## Acceptance Criteria

1. ✅ Code is restructured with extracted helper functions
2. ✅ `LookupOrCreateClusterFromConfig` is significantly shorter and clearer
3. ✅ All existing integration tests pass without modification
4. ✅ New unit tests added for pure functions
5. ✅ No change in external behavior or API
6. ✅ Code passes `make style` and `make golangci-lint`
7. ✅ PR description explains the refactoring and benefits

## Implementation Checklist

- [ ] Create helper type `clusterConfigData` with helper methods
  - [ ] Add struct definition
  - [ ] Add `clusterName()` method
  - [ ] Add `manager()` method
  - [ ] Add `helmConfig()` method
- [ ] Implement `extractClusterConfig()`
- [ ] Implement `shouldUpdateCluster()`
- [ ] Implement `validateClusterConfig()`
- [ ] Implement `buildClusterFromConfig()`
- [ ] Implement `applyConfigToCluster()`
- [ ] Implement `checkGracePeriodForReconnect()`
- [ ] Implement `lookupOrCreateCluster()`
- [ ] Refactor `LookupOrCreateClusterFromConfig`
- [ ] Add unit tests for pure functions
- [ ] Run existing integration tests
- [ ] Run `make style` and fix any issues
- [ ] Run `make golangci-lint` and fix any issues
- [ ] Manual testing of cluster registration flows
- [ ] Create commits following conventional commits
- [ ] Create draft PR with detailed description
