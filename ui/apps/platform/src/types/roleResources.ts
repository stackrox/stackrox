// Source of truth: central/role/resources/list.go

/*
 * Whenever you edit this file, make corresponding changes:
 * ui/apps/platform/cypress/fixtures/auth/mypermissions*.json
 * ui/apps/platform/src/Containers/AccessControl/PermissionSets/ResourceDescription.tsx
 */

// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// prettier-ignore
export type ResourceName =
    | 'Access' // Access is the new resource grouping all access related resources.
    | 'Administration' // Administration is the new resource grouping all administration-like resources.
    | 'Alert'
    | 'CVE' // SAC check is not performed directly on CVE resource. It exists here for postgres sac generation to pass.
    | 'Cluster'
    | 'Compliance'
    | 'Deployment'
    | 'DeploymentExtension' // DeploymentExtension is the new resource grouping all deployment extending resources.
    | 'Detection'
    | 'Image'
    | 'Integration' // Integration is the new resource grouping all integration resources.
    | 'K8sRole'
    | 'K8sRoleBinding'
    | 'K8sSubject'
    | 'Namespace'
    | 'NetworkGraph'
    | 'NetworkPolicy'
    | 'Node'
    | 'Secret'
    | 'ServiceAccount'
    | 'VulnerabilityManagementApprovals'
    | 'VulnerabilityManagementRequests'
    | 'WatchedImage'
    | 'WorkflowAdministration'
    ;
