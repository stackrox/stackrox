import withAuth from '../../helpers/basicAuth';
import { visit } from '../../helpers/visit';

describe('Risk redirect', () => {
    withAuth();

    it('should redirect /main/risk to /main/risk/workloads', () => {
        visit('/main/risk');

        cy.location('pathname').should('eq', '/main/risk/workloads');
        cy.get('h1:contains("Risk")');
    });

    it('should redirect /main/risk with query params to /main/risk/workloads', () => {
        visit('/main/risk?filteredWorkflowView=Full view');

        cy.location('pathname').should('eq', '/main/risk/workloads');
        cy.location('search').should('contain', 'filteredWorkflowView=Full%20view');
    });

    it('should redirect /main/risk/<deploymentId> to /main/risk/workloads/<deploymentId>', () => {
        visit('/main/risk/fake-deployment-id');

        cy.location('pathname').should('eq', '/main/risk/workloads/fake-deployment-id');
    });

    it('should redirect /main/risk/<deploymentId> with query params', () => {
        visit('/main/risk/fake-deployment-id?filteredWorkflowView=Applications view');

        cy.location('pathname').should('eq', '/main/risk/workloads/fake-deployment-id');
        cy.location('search').should('contain', 'filteredWorkflowView=Applications%20view');
    });
});
