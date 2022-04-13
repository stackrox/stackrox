import * as api from '../constants/apiEndpoints';
import { selectors as riskPageSelectors, url as riskURL } from '../constants/RiskPage';
import selectors from '../selectors/index';

export function visitRiskDeployments() {
    cy.intercept('GET', api.risks.riskyDeployments).as('getDeploymentsWithProcessInfo');
    cy.intercept('GET', api.risks.deploymentsCount).as('getDeploymentsCount');
    cy.visit(riskURL);
    cy.wait(['@getDeploymentsWithProcessInfo', '@getDeploymentsCount']);
}

export function viewRiskDeploymentByName(deploymentName) {
    // Assume location is risk deployments table.
    cy.intercept('GET', api.risks.fetchDeploymentWithRisk).as('getDeploymentWithRisk');
    cy.get(`${selectors.table.rows}:contains("${deploymentName}")`).click();
    cy.wait('@getDeploymentWithRisk');
}

export function viewRiskDeploymentInNetworkGraph() {
    // Assume location is risk deployment panel.
    cy.intercept('GET', api.network.deployment).as('getDeployment');
    cy.intercept('GET', api.network.networkGraph).as('getNetworkGraphCluster');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('getNetworkPoliciesCluster');
    cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
    cy.wait(['@getDeployment', '@getNetworkGraphCluster', '@getNetworkPoliciesCluster']);
}
