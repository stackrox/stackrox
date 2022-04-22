import * as api from '../constants/apiEndpoints';
import { url } from '../constants/DashboardPage';

// eslint-disable-next-line import/prefer-default-export
export function visitMainDashboard() {
    cy.intercept('GET', api.risks.riskyDeployments).as('riskyDeployments');
    cy.visit(url);
    cy.wait('@riskyDeployments');
}
