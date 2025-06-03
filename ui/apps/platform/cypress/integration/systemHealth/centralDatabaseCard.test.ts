import withAuth from '../../helpers/basicAuth';
import { visitSystemHealth } from '../../helpers/systemHealth';

describe('Central database card', () => {
    withAuth();

    it('should display no warnings when database version is up to date', () => {
        // Do not mock the response here, the intention is to track UI functionality
        // against the current deployed version of Postgres
        visitSystemHealth();

        cy.get('.pf-v5-c-card:contains("Central database health"):contains("no errors")');
    });
});
