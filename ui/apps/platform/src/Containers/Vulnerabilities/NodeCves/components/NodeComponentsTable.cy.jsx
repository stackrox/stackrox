import NodeComponentsTable from './NodeComponentsTable';

const mockData = [
    {
        name: 'podman',
        source: 'INFRASTRUCTURE',
    },
    {
        name: 'cri-o',
        source: 'KUBELET',
    },
    {
        name: 'kernel',
        source: 'INFRASTRUCTURE',
    },
];

function setup(data) {
    cy.mount(<NodeComponentsTable data={data} />);
}

describe(Cypress.spec.relative, () => {
    describe('client side sorting of the table', () => {
        it('should sort by the component name', () => {
            setup(mockData);

            const componentCell = 'td[data-label="Component"]';

            // Test that the default sort is by name descending
            cy.get(componentCell).eq(0).should('have.text', 'cri-o');
            cy.get(componentCell).eq(1).should('have.text', 'kernel');
            cy.get(componentCell).eq(2).should('have.text', 'podman');

            // Click the Component column sort button to sort descending
            cy.findByRole('columnheader', { name: /component/i }).findByRole('button').click();
            cy.get(componentCell).eq(0).should('have.text', 'podman');
            cy.get(componentCell).eq(1).should('have.text', 'kernel');
            cy.get(componentCell).eq(2).should('have.text', 'cri-o');

            // Click again to toggle back to ascending
            cy.findByRole('columnheader', { name: /component/i }).findByRole('button').click();
            cy.get(componentCell).eq(0).should('have.text', 'cri-o');
            cy.get(componentCell).eq(1).should('have.text', 'kernel');
            cy.get(componentCell).eq(2).should('have.text', 'podman');
        });

        it('should sort by the type', () => {
            setup(mockData);

            const typeCell = 'td[data-label="Type"]';

            // Click the Type column sort button to sort descending (defaultDirection is 'desc')
            cy.findByRole('columnheader', { name: /type/i }).findByRole('button').click();
            cy.get(typeCell).eq(0).should('have.text', 'KUBELET');
            cy.get(typeCell).eq(1).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(2).should('have.text', 'INFRASTRUCTURE');

            // Click again to toggle to ascending
            cy.findByRole('columnheader', { name: /type/i }).findByRole('button').click();
            cy.get(typeCell).eq(0).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(1).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(2).should('have.text', 'KUBELET');
        });
    });
});
