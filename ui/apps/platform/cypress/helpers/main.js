import pf6 from '../selectors/pf6';

import { hasFeatureFlag } from './features';
import { visit, visitConsole, visitWithStaticResponseForPermissions } from './visit';

const summaryCountMatchers = {
    Cluster: { method: 'GET', url: '/v1/clusters' },
    Node: { method: 'GET', url: '/v1/search?*' },
    Alert: { method: 'GET', url: '/v1/alertscount*' },
    Deployment: { method: 'GET', url: '/v1/deploymentscount*' },
    Image: { method: 'GET', url: '/v1/imagescount*' },
    Secret: { method: 'GET', url: '/v1/secretscount*' },
};

/**
 * REST route matchers for Dashboard summary counts. Pass only the resources the
 * role can read so visit helpers do not wait for requests that never fire.
 *
 * @param {Array<keyof typeof summaryCountMatchers>} resources
 */
export function routeMatcherMapForSummaryCountResources(resources) {
    return Object.fromEntries(
        resources.map((resource) => [`summary${resource}`, summaryCountMatchers[resource]])
    );
}

export const routeMatcherMapForSummaryCounts = routeMatcherMapForSummaryCountResources([
    'Cluster',
    'Node',
    'Alert',
    'Deployment',
    'Image',
    'Secret',
]);

export const deploymentsWithProcessInfoAlias = 'deploymentswithprocessinfo';
export const alertsSummaryCountsGroupByCategoryAlias = 'alerts/summary/counts_CATEGORY';

const routeMatcherMapForSearchFilter = {
    scopeNamespaces: {
        method: 'GET',
        url: '/v1/namespaces',
    },
};
const routeMatcherMapForViolationsByPolicySeverity = {
    alertCountsBySeverity: {
        method: 'GET',
        url: '/v1/alerts/summary/counts*',
    },
    mostRecentAlerts: {
        method: 'GET',
        url: '/v1/alerts?*',
    },
};
const routeMatcherMapForImagesAtMostRisk = {
    imagesAtMostRisk: {
        method: 'GET',
        url: '/v1/images?*',
    },
};
const routeMatcherMapForDeploymentsAtMostRisk = {
    [deploymentsWithProcessInfoAlias]: {
        method: 'GET',
        url: '/v1/deploymentswithprocessinfo?*',
    },
};
const routeMatcherMapForAgingImages = {
    agingImages: {
        method: 'GET',
        url: '/v1/imagescount*',
    },
};
const routeMatcherMapForViolationsByPolicyCategory = {
    [alertsSummaryCountsGroupByCategoryAlias]: {
        method: 'GET',
        url: '/v1/alerts/summary/counts?request.query=&group_by=CATEGORY',
    },
};
const routeMatcherMapForComplianceLevelsByStandard = {
    aggregatedResults: {
        method: 'GET',
        url: '/v1/compliance/aggregatedresults*',
    },
    complianceStandards: {
        method: 'GET',
        url: '/v1/compliance/standards',
    },
};

function getRouteMatcherMap() {
    return {
        ...routeMatcherMapForSummaryCounts,
        ...routeMatcherMapForSearchFilter,
        ...routeMatcherMapForViolationsByPolicySeverity,
        ...routeMatcherMapForImagesAtMostRisk,
        ...routeMatcherMapForDeploymentsAtMostRisk,
        ...routeMatcherMapForAgingImages,
        ...routeMatcherMapForViolationsByPolicyCategory,
        ...(hasFeatureFlag('ROX_DEPRECATED_COMPLIANCE_DASHBOARD')
            ? routeMatcherMapForComplianceLevelsByStandard
            : {}),
    };
}

const routeMatcherMapForConsole = {
    mypermissions: {
        method: 'GET',
        url: '**/api-service/**/v1/mypermissions',
    },
    featureflags: {
        method: 'GET',
        url: '**/api-service/**/v1/featureflags',
    },
    metadata: {
        method: 'GET',
        url: '**/api-service/**/v1/metadata',
    },
    'config/public': {
        method: 'GET',
        url: '**/api-service/**/v1/config/public',
    },
};

const basePath = '/main/dashboard';

const title = 'Dashboard';

// visit helpers

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitMainDashboard(staticResponseMap) {
    visit(basePath, getRouteMatcherMap(), staticResponseMap);

    cy.get(`.pf-v6-c-nav__link.pf-m-current:contains("${title}")`);
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitConsoleMainDashboard(staticResponseMap) {
    visitConsole('/dashboards', routeMatcherMapForConsole, staticResponseMap);
    cy.get(`${pf6.navExpandable} button:contains("Home")`).click();
    cy.get(`${pf6.navExpandable} ${pf6.navItem} a.pf-m-current:contains("Overview")`);
    cy.get(`h1:contains("Overview")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitConsoleDynamicPluginsStatusPage(staticResponseMap) {
    visitConsole(
        '/k8s/cluster/operator.openshift.io~v1~Console/cluster/console-plugins',
        routeMatcherMapForConsole,
        staticResponseMap
    );
}

/**
 * Visit main dashboard to test conditional rendering for user role permissions specified as response or fixture.
 * Conditional rendering for permissions might make a subset of requests.
 *
 * { body: { resourceToAccess: { … } } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {{ body: { resourceToAccess: Record<string, string> } } | { fixture: string }} staticResponseForPermissions
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMapForSubsetOfRequests]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMapForSubsetOfRequests]
 */
export function visitMainDashboardWithStaticResponseForPermissions(
    staticResponseForPermissions,
    routeMatcherMapForSubsetOfRequests,
    staticResponseMapForSubsetOfRequests
) {
    visitWithStaticResponseForPermissions(
        basePath,
        staticResponseForPermissions,
        routeMatcherMapForSubsetOfRequests,
        staticResponseMapForSubsetOfRequests
    );

    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {{data: Record<string, number>}} staticResponseForClustersForPermissions
 */
export function visitMainDashboardWithStaticResponseForClustersForPermission(
    staticResponseForClustersForPermissions
) {
    // Omit requests for widgets because Dashboard redirects to Clusters page.
    const clustersForPermissionsAlias = 'sac/clusters';
    const routeMatcherMapForClustersForPermissions = {
        [clustersForPermissionsAlias]: {
            method: 'GET',
            url: '/v1/sac/clusters?',
        },
    };
    const staticResponseMapForClustersForPermissions = {
        [clustersForPermissionsAlias]: staticResponseForClustersForPermissions,
    };
    visit(
        basePath,
        routeMatcherMapForClustersForPermissions,
        staticResponseMapForClustersForPermissions
    );

    // Omit assertion for Dashboard heading.
}
