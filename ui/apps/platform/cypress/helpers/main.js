import * as api from '../constants/apiEndpoints';
import { url } from '../constants/DashboardPage';
import navSelectors from '../selectors/navigation';

import { visit } from './visit';

// visit helpers

export function visitMainDashboardFromLeftNav() {
    cy.intercept('GET', api.risks.riskyDeployments).as('deploymentswithprocessinfo');

    cy.get(`${navSelectors.navLinks}:contains("Dashboard")`).click();

    cy.wait('@deploymentswithprocessinfo');
    cy.get('h1:contains("Dashboard")');
}

export function visitMainDashboard(requestConfig, staticResponseMap) {
    cy.intercept('GET', api.risks.riskyDeployments).as('deploymentswithprocessinfo');

    visit(url, requestConfig, staticResponseMap);

    cy.wait('@deploymentswithprocessinfo');
    cy.get('h1:contains("Dashboard")');
}

// TODO Make this the default once phase one of the PF Dashboard is enabled
export function visitMainDashboardPF() {
    visit(url);
}

export function visitMainDashboardViaRedirectFromUrl(redirectFromUrl) {
    cy.intercept('GET', api.risks.riskyDeployments).as('deploymentswithprocessinfo');

    visit(redirectFromUrl);

    cy.wait('@deploymentswithprocessinfo');
    cy.location('pathname').should('eq', url);
    cy.get('h1:contains("Dashboard")');
}
