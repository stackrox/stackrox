import { HasReadAccess } from 'hooks/usePermissions';
import { ResourceName } from 'types/roleResources';

// Apply subset of patterns from routePaths.ts file.

type ResourcePredicate = (hasReadAccess: HasReadAccess) => boolean;

type ResourceItem = ResourceName | ResourcePredicate;

// Given array of resource names, higher-order function returns predicate function.
function everyResource(resourceItems: ResourceItem[]): ResourcePredicate {
    return (hasReadAccess: HasReadAccess) =>
        resourceItems.every((resourceItem) => evaluateItem(resourceItem, hasReadAccess));
}

// Given either predicate or name, does it have read access?
function evaluateItem(resourceItem: ResourceItem, hasReadAccess: HasReadAccess) {
    if (typeof resourceItem === 'function') {
        return resourceItem(hasReadAccess);
    }

    return hasReadAccess(resourceItem);
}

// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// prettier-ignore
type ComplianceRouteKey =
    | 'compliance/clusters'
    | 'compliance/deployments'
    | 'compliance/namespaces'
    | 'compliance/nodes'
    ;

// Add properties in same order as route keys to minimize merge conflicts when multiple people add strings.
const routeRequirementsMap: Record<ComplianceRouteKey, ResourcePredicate> = {
    'compliance/clusters': everyResource(['Cluster']),
    'compliance/deployments': everyResource(['Cluster', 'Deployment', 'Namespace']),
    'compliance/namespaces': everyResource(['Cluster', 'Deployment', 'Namespace']),
    'compliance/nodes': everyResource(['Cluster', 'Node']),
};

// Component provides hasReadAccess function from usePermissions hook.
export function isComplianceRouteEnabled(
    hasReadAccess: HasReadAccess,
    routeKey: ComplianceRouteKey
) {
    const resourcePredicate = routeRequirementsMap[routeKey];
    return resourcePredicate(hasReadAccess);
}
