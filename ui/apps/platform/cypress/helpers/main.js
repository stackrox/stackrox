import * as api from '../constants/apiEndpoints';
import { url } from '../constants/DashboardPage';

// eslint-disable-next-line import/prefer-default-export
export function visitMainDashboard() {
    cy.intercept('POST', api.dashboard.summaryCounts).as('summaryCounts');
    cy.visit(url);
    cy.wait('@summaryCounts');
}
