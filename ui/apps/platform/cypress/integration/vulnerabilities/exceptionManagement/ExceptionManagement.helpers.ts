import { visit } from '../../../helpers/visit';

const basePath = '/main/vulnerabilities/exception-management';
export const pendingRequestsPath = `${basePath}/pending-requests`;

export function visitExceptionManagement() {
    visit(pendingRequestsPath);

    cy.get('h1:contains("Exception management")');
    cy.location('pathname').should('eq', pendingRequestsPath);
}
