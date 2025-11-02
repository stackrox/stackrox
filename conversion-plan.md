# Walker.Walk to Compile-time Schema Generation Conversion Plan

## Overview
Convert all remaining 110 `walker.Walk` usages in the schema directory to use compile-time generated schemas.

## Current Status
- ✅ **Completed**: 5 main entities (Alert, Deployment, Image, Policy, Cluster)
- 🔄 **Remaining**: 110 entities using runtime reflection via `walker.Walk`

## Strategy

### Phase 1: Enhance Schema Generator
1. **Auto-discovery mechanism**: Replace hard-coded list with automatic discovery
2. **Batch processing**: Generate all entities in one command
3. **Search category mapping**: Auto-detect search categories
4. **Scoping resource mapping**: Auto-detect scoping resources

### Phase 2: Batch Conversion
Convert entities in logical groups to manage complexity:

#### Batch 1: Core Infrastructure (15 entities)
- AuthProvider, AuthMachineToMachineConfig
- Config, SystemInfo, InstallationInfo
- Role, K8SRole, K8SRoleBinding, PermissionSet
- Group, ServiceAccount, ServiceIdentity
- Secret, TokenMetadata, SimpleAccessScope

#### Batch 2: Compliance (25 entities)
- ComplianceConfig, ComplianceDomain, ComplianceIntegration
- ComplianceOperator* (15 entities)
- ComplianceRunMetadata, ComplianceRunResults, ComplianceStrings

#### Batch 3: Image/CVE Management (15 entities)
- ImageIntegration, ImageComponent, ImageComponentV2, ImageComponentEdge
- ImageCVE, ImageCVEV2, ImageCVEEdge, ImageV2, WatchedImage
- ClusterCVE, ClusterCVEEdge, NodeCVE, ComponentCVEEdge, NodeComponentCVEEdge, VulnerabilityRequest

#### Batch 4: Network & Security (10 entities)
- NetworkPolicy, NetworkPolicyApplicationUndoRecord, NetworkPolicyApplicationUndoDeploymentRecord
- NetworkBaseline, NetworkEntity, NetworkFlow, NetworkGraphConfig
- Risk, Hash, Blob

#### Batch 5: Node & Process Monitoring (10 entities)
- Node, NodeComponent, NodeComponentEdge
- ProcessBaseline, ProcessBaselineResults, ProcessIndicator, ProcessListeningOnPortStorage
- ActiveComponent, Pod, NamespaceMetadata

#### Batch 6: Operations & Reporting (15 entities)
- AdministrationEvent, ReportConfiguration, ReportSnapshot
- NotificationSchedule, Notifier, NotifierEncConfig
- CloudSource, DelegatedRegistryConfig, DiscoveredCluster
- ExternalBackup, SensorUpgradeConfig, SecuredUnits
- PolicyCategory, PolicyCategoryEdge, ResourceCollection

#### Batch 7: Infrastructure & Misc (20 entities)
- ClusterHealthStatus, InitBundleMeta, IntegrationHealth
- DeclarativeConfigHealth, LogImbue, SignatureIntegration
- Version, ClusterInitBundle, NetworkBaseline
- Test* entities (11 test entities)

## Implementation Steps

### Step 1: Auto-Discovery Enhancement
```go
func (sg *SchemaGenerator) discoverAllSchemas() ([]SchemaData, error) {
    // Scan all schema files
    // Extract storage types from walker.Walk calls
    // Auto-detect search categories and scoping resources
    // Generate complete entity list
}
```

### Step 2: Search Category Detection
```go
func (sg *SchemaGenerator) detectSearchCategory(typeName string) string {
    // Use gopls to find search field registrations
    // Look for mapping.RegisterCategoryToTable calls
    // Extract v1.SearchCategory_* usage
}
```

### Step 3: Scoping Resource Detection
```go
func (sg *SchemaGenerator) detectScopingResource(typeName string) string {
    // Scan for resources.* usage in existing schemas
    // Default mapping based on entity type patterns
}
```

### Step 4: Batch Generation
```bash
# Generate all entities at once
./tools/generate-schema/generate-schema --discover-all --output pkg/postgres/schema

# Generate specific batches for testing
./tools/generate-schema/generate-schema --batch=core-infrastructure
```

### Step 5: Conversion Process (per batch)
1. **Generate schemas**: `generate-schema --batch=X`
2. **Remove old files**: Delete original schema files with walker.Walk
3. **Update imports**: Use gopls to find and update all callers
4. **Test compilation**: Ensure all packages compile
5. **Run tests**: Verify functionality
6. **Commit batch**: One commit per batch for easy rollback

## Risk Mitigation

### 1. Incremental Conversion
- Convert one batch at a time
- Test thoroughly before moving to next batch
- Keep old and new side-by-side during transition

### 2. Validation Strategy
- Compare generated schemas with walker.Walk output
- Unit tests for each batch
- Integration tests for critical paths

### 3. Rollback Plan
- Each batch is a separate commit
- Can revert individual batches if issues found
- Maintain build system compatibility

## Testing Strategy

### 1. Schema Validation
```go
func TestGeneratedSchemaMatchesWalker(t *testing.T) {
    for _, entity := range allEntities {
        generated := GetGeneratedSchema(entity)
        walked := walker.Walk(entity.Type, entity.Table)
        assertSchemasEqual(t, generated, walked)
    }
}
```

### 2. Integration Tests
- Database schema generation
- Search functionality
- CRUD operations
- Performance benchmarks

## Success Criteria

1. **Zero walker.Walk calls** in pkg/postgres/schema/
2. **All schemas compile-time generated**
3. **No performance regression**
4. **All tests passing**
5. **Search functionality preserved**

## Timeline Estimate

- **Phase 1 (Generator Enhancement)**: 1-2 days
- **Phase 2 (Batch Conversion)**: 1 week (7 batches × 1 day each)
- **Testing & Validation**: 1-2 days
- **Total**: ~2 weeks

## Next Actions

1. Start with auto-discovery mechanism
2. Enhance generator to handle all entity types
3. Begin with Batch 1 (Core Infrastructure)
4. Use gopls extensively for caller analysis