import * as api from '../constants/apiEndpoints';
import { selectors as riskPageSelectors, url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';

import { visit } from './visit';

export function visitRiskDeployments() {
    cy.intercept('GET', api.risks.riskyDeployments).as('getDeploymentsWithProcessInfo');
    cy.intercept('GET', api.risks.deploymentsCount).as('getDeploymentsCount');
    cy.intercept('POST', api.graphql('searchOptions')).as('postSearchOptions');
    visit(riskURL);
    cy.wait(['@getDeploymentsWithProcessInfo', '@getDeploymentsCount', '@postSearchOptions']);
    cy.get('h1:contains("Risk")');
}

export function visitRiskDeploymentsWithSearchQuery(search) {
    cy.intercept('GET', api.risks.riskyDeployments).as('getDeploymentsWithProcessInfo');
    cy.intercept('GET', api.risks.deploymentsCount).as('getDeploymentsCount');
    cy.intercept('POST', api.graphql('searchOptions')).as('postSearchOptions');
    visit(`${riskURL}${search}`);
    // Future improvements to RiskTablePanel might fix double requests.
    // Incorrect pair of requests without search filter before response for search options:
    cy.wait(['@getDeploymentsWithProcessInfo', '@getDeploymentsCount', '@postSearchOptions']);
    // Correct pair of requests with search filter after response for search options:
    cy.wait(['@getDeploymentsWithProcessInfo', '@getDeploymentsCount']);
    cy.get('h1:contains("Risk")');
}

export function viewRiskDeploymentByName(deploymentName) {
    // Assume location is risk deployments table.
    cy.intercept('GET', api.risks.fetchDeploymentWithRisk).as('getDeploymentWithRisk');
    cy.get(
        `${selectors.table.rows} ${selectors.table.cells}:nth-child(1):contains("${deploymentName}")`
    ).click();
    cy.wait('@getDeploymentWithRisk');
    cy.get(`${riskPageSelectors.sidePanel.panelHeader}:contains("${deploymentName}")`);
}

export function viewRiskDeploymentInNetworkGraph() {
    // Assume location is risk deployment panel.
    cy.intercept('GET', api.network.networkGraph).as('getNetworkGraphCluster');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('getNetworkPoliciesCluster');
    cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
    cy.wait(['@getNetworkGraphCluster', '@getNetworkPoliciesCluster']);
}
