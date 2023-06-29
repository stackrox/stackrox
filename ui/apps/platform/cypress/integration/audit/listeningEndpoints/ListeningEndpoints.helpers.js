import { visit } from '../../../helpers/visit';

const basePath = '/main/audit/listening-endpoints/';

export function visitListeningEndpoints() {
    visit(basePath);

    cy.get('h1:contains("Listening endpoints")');
    cy.location('pathname').should('eq', basePath);
}
