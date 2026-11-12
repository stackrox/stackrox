import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import RelatedDeployments from './RelatedDeployments';

function deployments() {
    return Array.from({ length: 21 }, (_, i) => ({
        __typename: 'Deployment',
        id: `deployment-${i}`,
        name: 'app',
        clusterName: i % 2 === 0 ? 'cluster-a' : 'cluster-b',
        namespace: 'default',
        created: null,
    }));
}

function mountRelatedDeployments() {
    window.history.pushState({}, document.title, '/main/administration-events/event-id');
    cy.intercept('POST', graphqlUrl('getImageDeployments'), (req) => {
        const { offset, limit } = req.body.variables.pagination;
        req.reply({
            data: {
                imageV2: {
                    __typename: 'ImageV2',
                    id: 'image-id',
                    deploymentCount: 21,
                    deployments: deployments().slice(offset, offset + limit),
                },
            },
        });
    }).as('deployments');
    cy.mount(
        <ComponentTestProvider>
            <RelatedDeployments imageId="image-id" />
        </ComponentTestProvider>
    );
    cy.wait('@deployments');
    cy.findByRole('button', { name: 'Related deployments (21)' }).click();
}

describe(Cypress.spec.relative, () => {
    it('keeps a unique ordering when paging and changing page size', () => {
        mountRelatedDeployments();
        cy.findByRole('button', { name: 'Go to next page' }).click();
        cy.wait('@deployments')
            .its('request.body.variables.pagination')
            .should('deep.equal', {
                offset: 10,
                limit: 10,
                sortOptions: [
                    { field: 'Deployment', reversed: false },
                    { field: 'Deployment ID', reversed: false },
                ],
            });
        cy.get('td[data-label="Deployment"] a')
            .first()
            .should('have.attr', 'href')
            .and('include', '/deployments/deployment-10');

        cy.findByRole('button', { name: '11 - 20 of 21' }).click();
        cy.findByRole('menuitem', { name: '20 per page' }).click();
        cy.wait('@deployments').its('request.body.variables.pagination').should('include', {
            offset: 0,
            limit: 20,
        });
        cy.get('td[data-label="Deployment"] a').should('have.length', 20);
    });
});
