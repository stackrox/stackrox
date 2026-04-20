import withAuth from '../../helpers/basicAuth';
import { selectFilteredWorkflowView, visitRiskDeployments } from './Risk.helpers';

describe('Risk - Filtered Workflow Views', () => {
    withAuth();

    it('should filter the deployments table when the "Applications view" is selected', () => {
        cy.intercept('GET', '/v1/deploymentswithprocessinfo*').as('getDeployments');

        visitRiskDeployments();

        cy.wait('@getDeployments').then((interception) => {
            const queryString = interception.request.query.query as string;

            expect(queryString).to.contain('Platform Component:false');
        });
    });

    it('should filter the deployments table when the "Platform view" is selected', () => {
        visitRiskDeployments();

        // Ensure the table is fully rendered before setting up the new intercept
        cy.get('table tbody tr').should('exist');

        cy.intercept('GET', '/v1/deploymentswithprocessinfo*').as('getDeploymentsAfterNav');
        selectFilteredWorkflowView('Platform');
        cy.wait('@getDeploymentsAfterNav').then((interception) => {
            const queryString = interception.request.query.query as string;

            expect(queryString).to.contain('Platform Component:true');
        });
    });

    it('should filter the deployments table when the "Full view" is selected', () => {
        visitRiskDeployments();

        // Ensure the table is fully rendered before setting up the new intercept
        cy.get('table tbody tr').should('exist');

        cy.intercept('GET', '/v1/deploymentswithprocessinfo*').as('getDeploymentsAfterNav');
        selectFilteredWorkflowView('All Deployments');
        cy.wait('@getDeploymentsAfterNav').then((interception) => {
            const queryString = (interception.request.query.query as string) ?? '';

            expect(queryString).to.not.contain('Platform Component:true');
            expect(queryString).to.not.contain('Platform Component:false');
        });
    });
});
