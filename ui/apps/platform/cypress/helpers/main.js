import * as api from '../constants/apiEndpoints';
import { url } from '../constants/DashboardPage';
import { visit } from './visit';

// eslint-disable-next-line import/prefer-default-export
export function visitMainDashboard() {
    cy.intercept('GET', api.risks.riskyDeployments).as('riskyDeployments');
    visit(url);
    cy.wait('@riskyDeployments');
    cy.get('h1:contains("Dashboard")');
}
