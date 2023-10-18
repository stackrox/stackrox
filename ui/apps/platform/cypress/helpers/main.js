import navSelectors from '../selectors/navigation';

import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from './request';
import { visit, visitWithStaticResponseForPermissions } from './visit';

/*
 * Import relevant alias constants in test files that call visitMainDashboard function
 * with staticResponseMap argument to provide mock data for a widget.
 */
export const summaryCountsOpname = 'summary_counts';
export const getAllNamespacesByClusterOpname = 'getAllNamespacesByCluster';
export const alertCountsBySeverityOpname = 'alertCountsBySeverity';
export const mostRecentAlertsOpname = 'mostRecentAlerts';
export const getImagesAtMostRiskOpname = 'getImagesAtMostRisk';
export const deploymentsWithProcessInfoAlias = 'deploymentswithprocessinfo';
export const agingImagesQueryOpname = 'agingImagesQuery';
export const alertsSummaryCountsGroupByCategoryAlias = 'alerts/summary/counts_CATEGORY';
export const getAggregatedResultsOpname = 'getAggregatedResults';

const routeMatcherMapForSummaryCounts = getRouteMatcherMapForGraphQL([summaryCountsOpname]);
const routeMatcherMapForSearchFilter = getRouteMatcherMapForGraphQL([
    getAllNamespacesByClusterOpname,
]);
const routeMatcherMapForViolationsByPolicySeverity = {
    ...getRouteMatcherMapForGraphQL([alertCountsBySeverityOpname]),
    ...getRouteMatcherMapForGraphQL([mostRecentAlertsOpname]),
};
const routeMatcherMapForImagesAtMostRisk = getRouteMatcherMapForGraphQL([
    getImagesAtMostRiskOpname,
]);
const routeMatcherMapForDeploymentsAtMostRisk = {
    [deploymentsWithProcessInfoAlias]: {
        method: 'GET',
        url: '/v1/deploymentswithprocessinfo?*',
    },
};
const routeMatcherMapForAgingImages = getRouteMatcherMapForGraphQL([agingImagesQueryOpname]);
const routeMatcherMapForViolationsByPolicyCategory = {
    [alertsSummaryCountsGroupByCategoryAlias]: {
        method: 'GET',
        url: '/v1/alerts/summary/counts?request.query=&group_by=CATEGORY',
    },
};
const routeMatcherMapForComplianceLevelsByStandard = getRouteMatcherMapForGraphQL([
    getAggregatedResultsOpname,
]);

const routeMatcherMap = {
    ...routeMatcherMapForSummaryCounts,
    ...routeMatcherMapForSearchFilter,
    ...routeMatcherMapForViolationsByPolicySeverity,
    ...routeMatcherMapForImagesAtMostRisk,
    ...routeMatcherMapForDeploymentsAtMostRisk,
    ...routeMatcherMapForAgingImages,
    ...routeMatcherMapForViolationsByPolicyCategory,
    ...routeMatcherMapForComplianceLevelsByStandard,
};

const basePath = '/main/dashboard';

const title = 'Dashboard';

// visit helpers

export function visitMainDashboardFromLeftNav() {
    interactAndWaitForResponses(() => {
        cy.get(`${navSelectors.navLinks}:contains("${title}")`).click();
    }, routeMatcherMap);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitMainDashboard(staticResponseMap) {
    visit(basePath, routeMatcherMap, staticResponseMap);

    cy.get(`.pf-c-nav__link.pf-m-current:contains("${title}")`);
    cy.get(`h1:contains("${title}")`);
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
