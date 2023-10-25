/**
 * Application route paths constants.
 */

import { resourceTypes, standardEntityTypes, rbacConfigTypes } from 'constants/entityTypes';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';
import { FeatureFlagEnvVar } from 'types/featureFlag';
import { ResourceName } from 'types/roleResources';

export const mainPath = '/main';
export const loginPath = '/login';
export const testLoginResultsPath = '/test-login-results';
export const authResponsePrefix = '/auth/response/';
export const authorizeRoxctlPath = '/authorize-roxctl';

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
export const clustersInitBundlesPath = `${clustersBasePath}/init-bundles`;
export const clustersInitBundlesPathWithParam = `${clustersInitBundlesPath}/:id?`;
export const collectionsBasePath = `${mainPath}/collections`;
export const collectionsPath = `${mainPath}/collections/:collectionId?`;
export const complianceBasePath = `${mainPath}/compliance`;
export const compliancePath = `${mainPath}/:context(compliance)`;
export const complianceEnhancedBasePath = `${mainPath}/compliance-enhanced`;
export const complianceEnhancedStatusPath = `${complianceEnhancedBasePath}/status`;
export const complianceEnhancedStatusClustersPath = `${complianceEnhancedStatusPath}/clusters/:id`;
export const complianceEnhancedStatusProfilesPath = `${complianceEnhancedStatusPath}/profiles/:id`;
export const complianceEnhancedStatusScansPath = `${complianceEnhancedStatusPath}/scans/:id`;
export const complianceEnhancedScanConfigsBasePath = `${complianceEnhancedBasePath}/scan-configs`;
export const complianceEnhancedScanConfigsPath = `${complianceEnhancedBasePath}/scan-configs/:scanConfigId`;
export const configManagementPath = `${mainPath}/configmanagement`;
export const dashboardPath = `${mainPath}/dashboard`;
export const dataRetentionPath = `${mainPath}/retention`;
export const exceptionConfigurationPath = `${mainPath}/exception-configuration`;
export const integrationsPath = `${mainPath}/integrations`;
export const integrationCreatePath = `${integrationsPath}/:source/:type/create`;
export const integrationDetailsPath = `${integrationsPath}/:source/:type/view/:id`;
export const integrationEditPath = `${integrationsPath}/:source/:type/edit/:id`;
export const integrationsListPath = `${integrationsPath}/:source/:type`;
export const listeningEndpointsBasePath = `${mainPath}/listening-endpoints`;
export const networkBasePath = `${mainPath}/network-graph`;
export const networkPath = `${networkBasePath}/:detailType?/:detailId?`;
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
export const violationsPath = `${violationsBasePath}/:alertId?`;
export const vulnManagementPath = `${mainPath}/vulnerability-management`;
export const vulnManagementReportsPath = `${vulnManagementPath}/reports`;
export const vulnManagementRiskAcceptancePath = `${vulnManagementPath}/risk-acceptance`;
export const vulnerabilitiesBasePath = `${mainPath}/vulnerabilities`;
export const vulnerabilitiesWorkloadCvesPath = `${vulnerabilitiesBasePath}/workload-cves`;
export const vulnerabilityReportsPath = `${vulnerabilitiesBasePath}/reports`;

// Vulnerability Management 1.0 path for links from Dashboard:

export const vulnManagementImagesPath = `${vulnManagementPath}/images`;

// Given an array of feature flags, higher-order functions return true or false based on
// whether all feature flags are enabled or disabled

type FeatureFlagPredicate = (isFeatureFlagEnabled: IsFeatureFlagEnabled) => boolean;

export function allEnabled(featureFlags: FeatureFlagEnvVar[]): FeatureFlagPredicate {
    return (isFeatureFlagEnabled: IsFeatureFlagEnabled): boolean => {
        return featureFlags.every((featureFlag) => isFeatureFlagEnabled(featureFlag));
    };
}

export function allDisabled(featureFlags: FeatureFlagEnvVar[]): FeatureFlagPredicate {
    return (isFeatureFlagEnabled: IsFeatureFlagEnabled): boolean => {
        return featureFlags.every((featureFlag) => !isFeatureFlagEnabled(featureFlag));
    };
}

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
    // Cluster init bundles must precede generic Clusters in Body and so here for consistency.
    | 'clusters/init-bundles'
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
    | 'vulnerabilities/reports' // add prefix because reports might become ambiguous in the future
    // Reports must precede generic Vulnerability Management in Body and so here for consistency.
    | 'vulnerability-management/reports'
    // Risk Acceptance must precede generic Vulnerability Management in Body and so here for consistency.
    | 'vulnerability-management/risk-acceptance'
    | 'vulnerability-management'
    | 'workload-cves'
    ;

// Add properties in same order as type to minimize merge conflicts when multiple people add strings.
const routeRequirementsMap: Record<RouteKey, RouteRequirements> = {
    'access-control': {
        resourceAccessRequirements: everyResource(['Access']),
    },
    'administration-events': {
        featureFlagRequirements: allEnabled(['ROX_ADMINISTRATION_EVENTS']),
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
    // Cluster init bundles must precede generic Clusters in Body and so here for consistency.
    'clusters/init-bundles': {
        featureFlagRequirements: allEnabled(['ROX_MOVE_INIT_BUNDLES_UI']),
        resourceAccessRequirements: everyResource(['Administration', 'Integration']),
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
        featureFlagRequirements: allEnabled(['ROX_COMPLIANCE_ENHANCEMENTS']),
        resourceAccessRequirements: everyResource(['Compliance']),
    },
    configmanagement: {
        // Require at least one resource for a dashboard widget.
        resourceAccessRequirements: someResource([
            'Alert',
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
        featureFlagRequirements: allEnabled(['ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL']),
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
    'vulnerabilities/reports': {
        featureFlagRequirements: allEnabled(['ROX_VULN_MGMT_REPORTING_ENHANCEMENTS']),
        resourceAccessRequirements: everyResource(['WorkflowAdministration']),
    },
    // Reports must precede generic Vulnerability Management in Body and so here for consistency.
    'vulnerability-management/reports': {
        featureFlagRequirements: allDisabled(['ROX_VULN_MGMT_REPORTING_ENHANCEMENTS']),
        resourceAccessRequirements: everyResource(['Integration', 'WorkflowAdministration']),
    },
    // Risk Acceptance must precede generic Vulnerability Management in Body and so here for consistency.
    'vulnerability-management/risk-acceptance': {
        resourceAccessRequirements: everyResource([
            'VulnerabilityManagementApprovals',
            'VulnerabilityManagementRequests',
        ]),
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
    'workload-cves': {
        featureFlagRequirements: allEnabled(['ROX_VULN_MGMT_WORKLOAD_CVES']),
        resourceAccessRequirements: everyResource(['Deployment', 'Image']),
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

export const urlEntityListTypes = {
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

export const urlEntityTypes = {
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

const vulnManagementPathToLabelMap = {
    [vulnManagementPath]: 'Dashboard',
    // TODO: add mapping for Deferrals
    [vulnManagementReportsPath]: 'Reporting',
    [vulnManagementRiskAcceptancePath]: 'Risk Acceptance',
};

const vulnerabilitiesPathToLabelMap = {
    [vulnerabilitiesBasePath]: 'Vulnerabilities',
    [vulnerabilitiesWorkloadCvesPath]: 'Workload CVEs',
    [vulnerabilityReportsPath]: 'Vulnerability Reporting',
};

export const basePathToLabelMap = {
    [dashboardPath]: 'Dashboard',
    [networkBasePath]: 'Network Graph',
    [listeningEndpointsBasePath]: 'Listening Endpoints',
    [violationsBasePath]: 'Violations',
    [complianceBasePath]: 'Compliance',
    [complianceEnhancedBasePath]: 'Compliance (2.0)',
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
