import * as api from '../constants/apiEndpoints';
import { selectors as riskPageSelectors, url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';

import { interactAndVisitNetworkGraphWithDeploymentSelected } from './networkGraph';
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

export function visitRiskDeployments() {
    visit(riskURL, routeMatcherMap);

    cy.get('h1:contains("Risk")');
}

export function visitRiskDeploymentsWithSearchQuery(search) {
    visit(`${riskURL}${search}`, routeMatcherMap);

    cy.get('h1:contains("Risk")');
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
    interactAndVisitNetworkGraphWithDeploymentSelected(() => {
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
    });
}
