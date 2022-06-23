import * as api from '../constants/apiEndpoints';
import { selectors as riskPageSelectors, url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';

import { visit } from './visit';

// visit

const routeMatcherMap = {
    deploymentswithprocessinfo: {
        method: 'GET',
        url: api.risks.riskyDeployments,
    },
    deploymentscount: {
        method: 'GET',
        url: api.risks.deploymentsCount,
    },
};

export function visitRiskDeployments() {
    visit(riskURL, { routeMatcherMap });

    cy.get('h1:contains("Risk")');
}

export function visitRiskDeploymentsWithSearchQuery(search) {
    visit(`${riskURL}${search}`, { routeMatcherMap });

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
    // Assume location is risk deployment panel.
    cy.intercept('GET', api.network.networkGraph).as('networkgraph/cluster/id');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('networkpolicies/cluster/id');

    cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();

    cy.wait(['@networkgraph/cluster/id', '@networkpolicies/cluster/id']);
}
