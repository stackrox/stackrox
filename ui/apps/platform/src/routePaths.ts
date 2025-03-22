/**
 * Application route paths constants.
 */

import { resourceTypes, standardEntityTypes, rbacConfigTypes } from 'constants/entityTypes';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';
import { ResourceName } from 'types/roleResources';
import { FeatureFlagPredicate, allEnabled } from 'utils/featureFlagUtils';

export const mainPath = '/main';
export const loginPath = '/login';
export const testLoginResultsPath = '/test-login-results';
export const authResponsePrefix = '/auth/response/';
export const authorizeRoxctlPath = '/authorize-roxctl';
export const vulnerabilitiesBasePath = `${mainPath}/vulnerabilities`;

// Add (related) path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
export const accessControlBasePath = `${mainPath}/access-control`;
export const accessControlPath = `${accessControlBasePath}/:entitySegment?/:entityId?`;
export const administrationEventsBasePath = `${mainPath}/administration-events`;
export const administrationEventsPathWithParam = `${administrationEventsBasePath}/:id?`;
export const apidocsPath = `${mainPath}/apidocs`;
export const apidocsPathV2 = `${mainPath}/apidocs-v2`;
export const clustersBasePath = `${mainPath}/clusters`;
export const clustersPathWithParam = `${clustersBasePath}/:clusterId?`;
export const clustersListPath = `${mainPath}/clusters-pf`;
export const clustersDelegatedScanningPath = `${clustersBasePath}/delegated-image-scanning`;
export const clustersDiscoveredClustersPath = `${clustersBasePath}/discovered-clusters`;
export const clustersInitBundlesPath = `${clustersBasePath}/init-bundles`;
export const clustersInitBundlesPathWithParam = `${clustersInitBundlesPath}/:id?`;
export const clustersSecureClusterPath = `${clustersBasePath}/secure-a-cluster`;
export const collectionsBasePath = `${mainPath}/collections`;
export const collectionsPath = `${mainPath}/collections/:collectionId?`;
export const complianceBasePath = `${mainPath}/compliance`;
export const compliancePath = `${mainPath}/:context(compliance)`;
export const complianceEnhancedBasePath = `${mainPath}/compliance`;
export const complianceEnhancedCoveragePath = `${complianceEnhancedBasePath}/coverage`;
export const complianceEnhancedSchedulesPath = `${complianceEnhancedBasePath}/schedules`;
export const configManagementPath = `${mainPath}/configmanagement`;
export const dashboardPath = `${mainPath}/dashboard`;
export const dataRetentionPath = `${mainPath}/retention`;
export const exceptionConfigurationPath = `${mainPath}/exception-configuration`;
export const exceptionManagementPath = `${vulnerabilitiesBasePath}/exception-management`;
export const integrationsPath = `${mainPath}/integrations`;
export const integrationCreatePath = `${integrationsPath}/:source/:type/create`;
export const integrationDetailsPath = `${integrationsPath}/:source/:type/view/:id`;
export const integrationEditPath = `${integrationsPath}/:source/:type/edit/:id`;
export const integrationsListPath = `${integrationsPath}/:source/:type`;
export const listeningEndpointsBasePath = `${mainPath}/listening-endpoints`;
export const networkBasePath = `${mainPath}/network-graph`;
export const networkPath = `${networkBasePath}/:nodeType?/:nodeId?/:detailType?/:detailID?`;
export const policyManagementBasePath = `${mainPath}/policy-management`;
export const policiesBasePath = `${policyManagementBasePath}/policies`;
export const policiesPath = `${policiesBasePath}/:policyId?/:command?`;
export const policyCategoriesPath = `${policyManagementBasePath}/policy-categories`;
export const deprecatedPoliciesBasePath = `${mainPath}/policies`;
export const deprecatedPoliciesPath = `${deprecatedPoliciesBasePath}/:policyId?/:command?`;
export const riskBasePath = `${mainPath}/risk`;
export const riskPath = `${riskBasePath}/:deploymentId?`;
export const searchPath = `${mainPath}/search`;
export const secretsPath = `${mainPath}/configmanagement/secrets/:secretId?`;
export const systemConfigPath = `${mainPath}/systemconfig`;
export const systemHealthPath = `${mainPath}/system-health`;
export const userBasePath = `${mainPath}/user`;
export const userRolePath = `${userBasePath}/roles/:roleName`;
export const violationsBasePath = `${mainPath}/violations`;
export const violationsUserWorkloadsViewPath = `${mainPath}/violations?filteredWorkflowView=Application view`;
export const violationsPlatformViewPath = `${mainPath}/violations?filteredWorkflowView=Platform view`;
export const violationsFullViewPath = `${mainPath}/violations?filteredWorkflowView=Full view`;
export const violationsPath = `${violationsBasePath}/:alertId?`;
export const vulnManagementPath = `${mainPath}/vulnerability-management`;
// TODO Deprecate these paths
export const vulnerabilitiesWorkloadCvesPath = `${vulnerabilitiesBasePath}/workload-cves`;
export const vulnerabilitiesPlatformCvesPath = `${vulnerabilitiesBasePath}/platform-cves`;
// TODO End Deprecate

export const vulnerabilitiesUserWorkloadsPath = `${vulnerabilitiesBasePath}/user-workloads`;
export const vulnerabilitiesPlatformPath = `${vulnerabilitiesBasePath}/platform`;
export const vulnerabilitiesNodeCvesPath = `${vulnerabilitiesBasePath}/node-cves`;
// System defined "views"
export const vulnerabilitiesAllImagesPath = `${vulnerabilitiesBasePath}/all-images`;
export const vulnerabilitiesInactiveImagesPath = `${vulnerabilitiesBasePath}/inactive-images`;
export const vulnerabilitiesImagesWithoutCvesPath = `${vulnerabilitiesBasePath}/images-without-cves`;
// user-workload template views path
export const vulnerabilitiesViewPath = `${vulnerabilitiesBasePath}/results/:viewTemplate/:viewId`;

export const vulnerabilityReportsPath = `${vulnerabilitiesBasePath}/reports`;

// Vulnerability Management 1.0 path for links from Dashboard:

export const vulnManagementImagesPath = `${vulnManagementPath}/images`;

// Compose resourceAccessRequirements from resource names and predicates.

type ResourcePredicate = (hasReadAccess: HasReadAccess) => boolean;

type ResourceItem = ResourceName | ResourcePredicate;

function evaluateItem(resourceItem: ResourceItem, hasReadAccess: HasReadAccess) {
    if (typeof resourceItem === 'function') {
        return resourceItem(hasReadAccess);
    }

    return hasReadAccess(resourceItem);
}

// Given array of resource names, higher-order functions return predicate function.
// You can also compose every with some, if requirements ever become so complicated.

export function everyResource(resourceItems: ResourceItem[]): ResourcePredicate {
    return (hasReadAccess: HasReadAccess) =>
        resourceItems.every((resourceItem) => evaluateItem(resourceItem, hasReadAccess));
}

export function someResource(resourceItems: ResourceItem[]): ResourcePredicate {
    return (hasReadAccess: HasReadAccess) =>
        resourceItems.some((resourceItem) => evaluateItem(resourceItem, hasReadAccess));
}

// Source of truth for conditional rendering of Body route paths and NavigationSidebar links.

// Factor out for useFetchClustersForPermissions and useFetchClusterNamespacesForPermissions.
// If the route ever requires global resources, spread them in resourceAccessRequirements property.
export const nonGlobalResourceNamesForNetworkGraph: ResourceName[] = [
    'Deployment',
    // 'DeploymentExtension',
    'NetworkGraph',
    // 'NetworkPolicy',
];

type RouteRequirements = {
    featureFlagRequirements?: FeatureFlagPredicate;
    resourceAccessRequirements: ResourcePredicate; // assume READ_ACCESS
};

// Semicolon on separate line following the strings prevents an extra changed line to add a string at the end.
// However, add strings in alphabetical order to minimize merge conflicts when multiple people add strings.
// prettier-ignore
export type RouteKey =
    | 'access-control'
    | 'administration-events'
    | 'apidocs'
    | 'apidocs-v2'
    // Delegated image scanning must precede generic Clusters in Body and so here for consistency.
    | 'clusters/delegated-image-scanning'
    // Discovered clusters must precede generic Clusters in Body and so here for consistency.
    | 'clusters/discovered-clusters'
    // Cluster init bundles must precede generic Clusters in Body and so here for consistency.
    | 'clusters/init-bundles'
    // Cluster secure-a-cluster must precede generic Clusters in Body and so here for consistency.
    | 'clusters/secure-a-cluster'
    | 'clusters'
    | 'collections'
    | 'compliance'
    | 'compliance-enhanced'
    | 'configmanagement'
    | 'dashboard'
    | 'exception-configuration'
    | 'integrations'
    | 'listening-endpoints'
    | 'network-graph'
    | 'policy-management'
    | 'risk'
    | 'search'
    | 'system-health'
    | 'systemconfig'
    | 'user'
    | 'violations'
    | 'vulnerabilities/exception-management'
    | 'vulnerabilities/node-cves'
    | 'vulnerabilities/reports'
    | 'vulnerabilities/user-workloads'
    | 'vulnerabilities/platform'
    | 'vulnerabilities/all-images'
    | 'vulnerabilities/inactive-images'
    | 'vulnerabilities/images-without-cves'
    | 'vulnerabilities/platform-cves'
    | 'vulnerabilities/workload-cves'
    | 'vulnerability-management'
    ;

// Add properties in same order as type to minimize merge conflicts when multiple people add strings.
const routeRequirementsMap: Record<RouteKey, RouteRequirements> = {
    'access-control': {
        resourceAccessRequirements: everyResource(['Access']),
    },
    'administration-events': {
        resourceAccessRequirements: everyResource(['Administration']),
    },
    apidocs: {
        resourceAccessRequirements: everyResource([]),
    },
    'apidocs-v2': {
        resourceAccessRequirements: everyResource([]),
    },
    // Delegated image scanning must precede generic Clusters in Body and so here for consistency.
    'clusters/delegated-image-scanning': {
        resourceAccessRequirements: everyResource(['Administration']),
    },
    // Discovered clusters must precede generic Clusters in Body and so here for consistency.
    'clusters/discovered-clusters': {
        resourceAccessRequirements: everyResource(['Administration']),
    },
    // Cluster init bundles must precede generic Clusters in Body and so here for consistency.
    'clusters/init-bundles': {
        resourceAccessRequirements: everyResource(['Administration', 'Integration']),
    },
    // Clusters secure-a-cluster must precede generic Clusters in Body and so here for consistency.
    'clusters/secure-a-cluster': {
        resourceAccessRequirements: everyResource([]),
    },
    clusters: {
        resourceAccessRequirements: everyResource(['Cluster']),
    },
    collections: {
        resourceAccessRequirements: everyResource(['Deployment', 'WorkflowAdministration']),
    },
    compliance: {
        // Same resources as compliance-enhanced although lack of commented-out resources affects entire list or entity pages.
        resourceAccessRequirements: everyResource([
            // 'Alert', // for Deployment
            // 'Cluster',
            'Compliance',
            // 'Deployment',
            // 'Image', // for Deployment and Namespace
            // 'K8sRole', // for Cluster
            // 'K8sRoleBinding', // for Cluster
            // 'K8sSubject', // for Cluster
            // 'Namespace',
            // 'NetworkPolicy', // for Namespace
            // 'Node',
            // 'Secret', // for Deployment and Namespace
            // 'ServiceAccount', // for Cluster and Deployment
        ]),
    },
    'compliance-enhanced': {
        resourceAccessRequirements: everyResource(['Compliance']),
    },
    configmanagement: {
        // Require at least one resource for a dashboard widget.
        resourceAccessRequirements: someResource([
            everyResource(['Alert', 'WorkflowAdministration']), // PolicyViolationsBySeverity
            // 'Cluster',
            'Compliance',
            // 'Deployment',
            // 'Image',
            // 'K8sRole',
            // 'K8sRoleBinding',
            'K8sSubject',
            // 'Namespace',
            // 'Node',
            'Secret',
            // 'ServiceAccount',
            // 'WorkflowAdministration',
        ]),
    },
    dashboard: {
        resourceAccessRequirements: everyResource([]),
    },
    'exception-configuration': {
        resourceAccessRequirements: everyResource(['Administration']),
    },
    integrations: {
        resourceAccessRequirements: everyResource(['Integration']),
    },
    'listening-endpoints': {
        resourceAccessRequirements: everyResource(['Deployment', 'DeploymentExtension']),
    },
    'network-graph': {
        resourceAccessRequirements: everyResource(nonGlobalResourceNamesForNetworkGraph),
    },
    'policy-management': {
        // The resources that are optional to view policies might become required to clone/create/edit a policy.
        resourceAccessRequirements: everyResource([
            // 'Deployment',
            // 'Image',
            // 'Integration',
            'WorkflowAdministration',
        ]),
    },
    risk: {
        resourceAccessRequirements: everyResource(['Deployment']),
    },
    search: {
        resourceAccessRequirements: everyResource([
            'Alert', // ALERTS
            'Cluster', // CLUSTERS
            'Deployment', // DEPLOYMENTS
            'Image', // IMAGES
            'Integration', // IMAGE_INTEGRATIONS
            'K8sRole', // ROLES
            'K8sRoleBinding', // ROLEBINDINGS
            'K8sSubject', // SUBJECTS
            'Namespace', // NAMESPACES
            'Node', // NODES
            'Secret', // SECRETS
            'ServiceAccount', // SERVICE_ACCOUNTS
            'WorkflowAdministration', // POLICIES POLICY_CATEGORIES
        ]),
    },
    'system-health': {
        resourceAccessRequirements: someResource(['Administration', 'Cluster', 'Integration']),
    },
    systemconfig: {
        resourceAccessRequirements: everyResource(['Administration']),
    },
    user: {
        resourceAccessRequirements: everyResource([]),
    },
    violations: {
        resourceAccessRequirements: everyResource(['Alert']),
    },
    'vulnerabilities/exception-management': {
        resourceAccessRequirements: someResource([
            'VulnerabilityManagementRequests',
            'VulnerabilityManagementApprovals',
        ]),
    },
    'vulnerabilities/node-cves': {
        resourceAccessRequirements: everyResource(['Cluster', 'Node']),
    },
    'vulnerabilities/platform-cves': {
        resourceAccessRequirements: everyResource(['Cluster']),
    },
    'vulnerabilities/reports': {
        resourceAccessRequirements: everyResource(['WorkflowAdministration']),
    },
    'vulnerabilities/workload-cves': {
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerabilities/user-workloads': {
        featureFlagRequirements: allEnabled(['ROX_PLATFORM_CVE_SPLIT']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerabilities/platform': {
        featureFlagRequirements: allEnabled(['ROX_PLATFORM_CVE_SPLIT']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerabilities/all-images': {
        featureFlagRequirements: allEnabled(['ROX_PLATFORM_CVE_SPLIT']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerabilities/inactive-images': {
        featureFlagRequirements: allEnabled(['ROX_PLATFORM_CVE_SPLIT']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerabilities/images-without-cves': {
        featureFlagRequirements: allEnabled(['ROX_PLATFORM_CVE_SPLIT']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
    },
    'vulnerability-management': {
        resourceAccessRequirements: everyResource([
            // 'Alert', // for Cluster and Deployment and Namespace
            // 'Cluster', // on Dashboard for with most widget
            'Deployment', // on Dashboard for Top Risky, Recently Detected, Most Common widgets
            'Image',
            // 'Namespace',
            // 'Node',
            // 'WatchedImage', // for Image
        ]),
    },
};

type RoutePredicates = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

export function isRouteEnabled(
    { hasReadAccess, isFeatureFlagEnabled }: RoutePredicates,
    routeKey: RouteKey
) {
    const { featureFlagRequirements, resourceAccessRequirements } = routeRequirementsMap[routeKey];

    const areFeatureFlagRequirementsMet = featureFlagRequirements
        ? featureFlagRequirements(isFeatureFlagEnabled)
        : true;

    const areResourceAccessRequirementsMet = resourceAccessRequirements(hasReadAccess);

    return areFeatureFlagRequirementsMet && areResourceAccessRequirementsMet;
}

/**
 * New Framwork-related route paths
 */

export const urlEntityListTypes: Record<string, string> = {
    [resourceTypes.NAMESPACE]: 'namespaces',
    [resourceTypes.CLUSTER]: 'clusters',
    [resourceTypes.NODE]: 'nodes',
    [resourceTypes.DEPLOYMENT]: 'deployments',
    [resourceTypes.IMAGE]: 'images',
    [resourceTypes.SECRET]: 'secrets',
    [resourceTypes.POLICY]: 'policies', // TODO verify if used for Configuration Management
    [resourceTypes.CVE]: 'cves', // TODO verify obsolete because non-postgres
    [resourceTypes.IMAGE_CVE]: 'image-cves',
    [resourceTypes.NODE_CVE]: 'node-cves',
    [resourceTypes.CLUSTER_CVE]: 'cluster-cves',
    [resourceTypes.COMPONENT]: 'components', // TODO verify obsolete because non-postgres
    [resourceTypes.NODE_COMPONENT]: 'node-components',
    [resourceTypes.IMAGE_COMPONENT]: 'image-components',
    [standardEntityTypes.CONTROL]: 'controls',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccounts',
    [rbacConfigTypes.SUBJECT]: 'subjects',
    [rbacConfigTypes.ROLE]: 'roles',
};

export const urlEntityTypes: Record<string, string> = {
    [resourceTypes.NAMESPACE]: 'namespace',
    [resourceTypes.CLUSTER]: 'cluster',
    [resourceTypes.NODE]: 'node',
    [resourceTypes.DEPLOYMENT]: 'deployment',
    [resourceTypes.IMAGE]: 'image',
    [resourceTypes.SECRET]: 'secret',
    [resourceTypes.POLICY]: 'policy', // TODO verify if used for Configuration Management
    [resourceTypes.CVE]: 'cve', // TODO verify obsolete because non-postgres
    [resourceTypes.IMAGE_CVE]: 'image-cve',
    [resourceTypes.NODE_CVE]: 'node-cve',
    [resourceTypes.CLUSTER_CVE]: 'cluster-cve',
    [resourceTypes.COMPONENT]: 'component', // TODO verify obsolete because non-postgres
    [resourceTypes.NODE_COMPONENT]: 'node-component',
    [resourceTypes.IMAGE_COMPONENT]: 'image-component',
    [standardEntityTypes.CONTROL]: 'control',
    [standardEntityTypes.STANDARD]: 'standard',
    [rbacConfigTypes.SERVICE_ACCOUNT]: 'serviceaccount',
    [rbacConfigTypes.SUBJECT]: 'subject',
    [rbacConfigTypes.ROLE]: 'role',
};

const vulnManagementPathToLabelMap: Record<string, string> = {
    [vulnManagementPath]: 'Dashboard',
};

const vulnerabilitiesPathToLabelMap: Record<string, string> = {
    [vulnerabilitiesBasePath]: 'Vulnerabilities',
    [vulnerabilitiesWorkloadCvesPath]: 'Workload CVEs',
    [vulnerabilitiesPlatformCvesPath]: 'Platform CVEs',
    [vulnerabilitiesNodeCvesPath]: 'Node CVEs',
    [vulnerabilityReportsPath]: 'Vulnerability Reporting',
    [exceptionManagementPath]: 'Exception Management',
};

export const basePathToLabelMap: Record<string, string> = {
    [dashboardPath]: 'Dashboard',
    [networkBasePath]: 'Network Graph',
    [listeningEndpointsBasePath]: 'Listening Endpoints',
    [violationsBasePath]: 'Violations',
    [complianceBasePath]: 'Compliance',
    // [complianceEnhancedBasePath]: 'Compliance (2.0)',
    ...vulnerabilitiesPathToLabelMap,
    ...vulnManagementPathToLabelMap,
    [configManagementPath]: 'Configuration Management',
    [riskBasePath]: 'Risk',
    [apidocsPath]: 'API Reference (v1)',
    [apidocsPathV2]: 'API Reference (v2)',
    [clustersBasePath]: 'Clusters',
    [policyManagementBasePath]: 'Policy Management',
    [policiesBasePath]: 'Policy Management',
    [policyCategoriesPath]: 'Policy Categories',
    [collectionsBasePath]: 'Collections',
    [integrationsPath]: 'Integrations',
    [accessControlPath]: 'Access Control',
    [accessControlBasePath]: 'Access Control',
    [systemConfigPath]: 'System Configuration',
    [systemHealthPath]: 'System Health',
    [loginPath]: 'Log In',
    [userBasePath]: 'User Profile',
};

const entityListTypeMatcher = `(${Object.values(urlEntityListTypes).join('|')})`;
const entityTypeMatcher = `(${Object.values(urlEntityTypes).join('|')})`;

export const workflowPaths = {
    DASHBOARD: `${mainPath}/:context`,
    LIST: `${mainPath}/:context/:pageEntityListType${entityListTypeMatcher}/:entityId1?/:entityType2?/:entityId2?`,
    ENTITY: `${mainPath}/:context/:pageEntityType${entityTypeMatcher}/:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`,
};
