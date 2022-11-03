import { url as basePath } from '../constants/DashboardPage';
import navSelectors from '../selectors/navigation';

import { getRouteMatcherForGraphQL, interactAndWaitForResponses } from './request';
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

const routeMatcherMap = {
    [summaryCountsOpname]: getRouteMatcherForGraphQL(summaryCountsOpname),
    [getAllNamespacesByClusterOpname]: getRouteMatcherForGraphQL(getAllNamespacesByClusterOpname),
    [alertsSummaryCountsAlias]: {
        method: 'GET',
        url: '/v1/alerts/summary/counts?request.query=',
    },

    // ViolationsByPolicySeverity
    [mostRecentAlertsOpname]: getRouteMatcherForGraphQL(mostRecentAlertsOpname),

    // ImagesAtMostRisk
    [getImagesOpname]: getRouteMatcherForGraphQL(getImagesOpname),

    // DeploymentsAtMostRisk
    [deploymentsWithProcessInfoAlias]: {
        method: 'GET',
        url: '/v1/deploymentswithprocessinfo?*',
    },

    // AgingImages
    [agingImagesQueryOpname]: getRouteMatcherForGraphQL(agingImagesQueryOpname),

    // ViolationsByPolicySeverity ViolationsByPolicyCategory
    [alertsSummaryCountsGroupByCategoryAlias]: {
        method: 'GET',
        url: '/v1/alerts/summary/counts?request.query=&group_by=CATEGORY',
    },

    // ComplianceLevelsByStandard
    [getAggregatedResultsOpname]: getRouteMatcherForGraphQL(getAggregatedResultsOpname),
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

export function visitMainDashboard(staticResponseMap) {
    visit(basePath, routeMatcherMap, staticResponseMap);

    cy.get(`.pf-c-nav__link.pf-m-current:contains("${title}")`);
    cy.get(`h1:contains("${title}")`);
}
