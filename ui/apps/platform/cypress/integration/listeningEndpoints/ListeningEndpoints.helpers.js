import { visitFromLeftNavExpandable } from '../../helpers/nav';

export function visitListeningEndpointsFromLeftNav() {
    visitFromLeftNavExpandable('Network', 'Listening Endpoints');

    cy.get('h1:contains("Listening endpoints")');
}
