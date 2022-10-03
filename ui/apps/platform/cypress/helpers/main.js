import * as api from '../constants/apiEndpoints';
import { url } from '../constants/DashboardPage';
import navSelectors from '../selectors/navigation';

import { interactAndWaitForResponses } from './request';
import { visit } from './visit';

/*
 * Import relevant alias constants in test files that call visitMainDashboard function
 * with staticResponseMap argument to provide mock data for a widget.
 */
export const mostRecentAlertsAlias = 'mostRecentAlerts';
export const getImagesAlias = 'getImages';
export const deploymentswithprocessinfoAlias = 'deploymentswithprocessinfo';
export const agingImagesQueryAlias = 'agingImagesQuery';
export const alertsSummaryCountsAlias = 'alerts/summary/counts';
export const getAggregatedResultsAlias = 'getAggregatedResults';

const requestConfig = {
    routeMatcherMap: {
        // ViolationsByPolicySeverity
        [mostRecentAlertsAlias]: {
            method: 'POST',
            url: api.graphql('mostRecentAlerts'),
        },
        // ImagesAtMostRisk
        [getImagesAlias]: {
            method: 'POST',
            url: api.graphql('getImages'),
        },
        // DeploymentsAtMostRisk
        [deploymentswithprocessinfoAlias]: {
            method: 'GET',
            url: api.risks.riskyDeployments,
        },
        // AgingImages
        [agingImagesQueryAlias]: {
            method: 'POST',
            url: api.graphql('agingImagesQuery'),
        },
        // ViolationsByPolicySeverity ViolationsByPolicyCategory
        [alertsSummaryCountsAlias]: {
            method: 'GET',
            url: api.alerts.countsByCategory,
        },
        // ComplianceLevelsByStandard
        [getAggregatedResultsAlias]: {
            method: 'POST',
            url: api.graphql('getAggregatedResults'),
        },
    },
};

// visit helpers

export function visitMainDashboardFromLeftNav() {
    interactAndWaitForResponses(() => {
        cy.get(`${navSelectors.navLinks}:contains("Dashboard")`).click();
    }, requestConfig);

    cy.get('h1:contains("Dashboard")');
}

export function visitMainDashboard(staticResponseMap) {
    visit(url, requestConfig, staticResponseMap);

    cy.get('h1:contains("Dashboard")');
}
