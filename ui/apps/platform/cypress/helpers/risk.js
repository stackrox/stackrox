import * as api from '../constants/apiEndpoints';
import { selectors as riskPageSelectors, url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';

import { reachNetworkGraphWithDeploymentSelected } from './networkGraph';
import { interceptAndWaitForResponses } from './request';
import { visit } from './visit';

// visit

export const deploymentswithprocessinfoAlias = 'deploymentswithprocessinfo';
export const deploymentscountAlias = 'deploymentscount';
export const searchOptionsAlias = 'searchOptions';

const routeMatcherMap = {
    [deploymentswithprocessinfoAlias]: {
        method: 'GET',
        url: api.risks.riskyDeployments,
    },
    [deploymentscountAlias]: {
        method: 'GET',
        url: api.risks.deploymentsCount,
    },
    [searchOptionsAlias]: {
        method: 'POST',
        url: api.graphql('searchOptions'),
    },
};

const title = 'Risk';

export function visitRiskDeployments() {
    visit(riskURL);

    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap);
}

export function visitRiskDeploymentsWithSearchQuery(search) {
    visit(`${riskURL}${search}`, routeMatcherMap);

    cy.get(`h1:contains("${title}")`);

    interceptAndWaitForResponses(routeMatcherMap);
}

export function viewRiskDeploymentByName(deploymentName) {
    // Assume location is risk deployments table.
    cy.intercept('GET', api.risks.fetchDeploymentWithRisk).as('deploymentswithrisk/id');

    cy.get(
        `${selectors.table.rows} ${selectors.table.cells}:nth-child(1):contains("${deploymentName}")`
    ).click();

    cy.wait('@deploymentswithrisk/id');
    cy.get(`${riskPageSelectors.sidePanel.panelHeader}:contains("${deploymentName}")`);
}

export function viewRiskDeploymentInNetworkGraph() {
    reachNetworkGraphWithDeploymentSelected(() => {
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
    });
}
