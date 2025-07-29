import withAuth from '../helpers/basicAuth';

describe('Basic connectivity to the OCP plugin', () => {
    it('should open the OCP web console', () => {
        cy.visit('/');
        // TODO Handle auth/skip auth in dev
        // TODO Handle OCP welcome modal
    });
});
