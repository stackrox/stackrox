import { url as basePath } from '../constants/DashboardPage';
import navSelectors from '../selectors/navigation';

import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from './request';
import { visit } from './visit';

/*
 * Import relevant alias constants in test files that call visitMainDashboard function
 * with staticResponseMap argument to provide mock data for a widget.
 */
export const summaryCountsOpname = 'summary_counts';
export const getAllNamespacesByClusterOpname = 'getAllNamespacesByCluster';
export const alertsSummaryCountsAlias = 'alerts/summary/counts';
export const mostRecentAlertsOpname = 'mostRecentAlerts';
export const getImagesOpname = 'getImages';
export const deploymentsWithProcessInfoAlias = 'deploymentswithprocessinfo';
export const agingImagesQueryOpname = 'agingImagesQuery';
export const alertsSummaryCountsGroupByCategoryAlias = 'alerts/summary/counts_CATEGORY';
export const getAggregatedResultsOpname = 'getAggregatedResults';

const routeMatcherMapForSummaryCounts = getRouteMatcherMapForGraphQL([summaryCountsOpname]);
const routeMatcherMapForSearchFilter = getRouteMatcherMapForGraphQL([
    getAllNamespacesByClusterOpname,
]);
const routeMatcherMapForViolationsByPolicySeverity = {
    [alertsSummaryCountsAlias]: {
        method: 'GET',
        url: '/v1/alerts/summary/counts?request.query=',
    },
    ...getRouteMatcherMapForGraphQL([mostRecentAlertsOpname]),
};
const routeMatcherMapForImagesAtMostRisk = getRouteMatcherMapForGraphQL([getImagesOpname]);
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
 * @param {{data: Record<string, number>}} staticResponseForSummaryCounts
 */
export function visitMainDashboardWithStaticResponseForSummaryCounts(
    staticResponseForSummaryCounts
) {
    // Omit requests for widgets because Dashboard redirects to Clusters page.
    const staticResponseMapForSummaryCounts = {
        [summaryCountsOpname]: staticResponseForSummaryCounts,
    };
    visit(basePath, routeMatcherMapForSummaryCounts, staticResponseMapForSummaryCounts);

    // Omit assertion for Dashboard heading.
}
