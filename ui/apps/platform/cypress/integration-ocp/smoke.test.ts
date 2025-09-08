import { withOcpAuth } from '../helpers/ocpAuth';

describe('Basic connectivity to the OCP plugin', () => {
    it('should open the OCP web console', () => {
        withOcpAuth();

        cy.visit('/');

        cy.get('h1:contains("Overview")');
    });
});
