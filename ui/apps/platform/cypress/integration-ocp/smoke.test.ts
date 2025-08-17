import { withOcpAuth } from '../helpers/ocpAuth';

describe('Basic connectivity to the OCP plugin', () => {
    it('should open the OCP web console', () => {
        withOcpAuth();

        cy.visit('/');
        // TODO Handle auth/skip auth in dev
        // TODO Handle OCP welcome modal
    });
});
