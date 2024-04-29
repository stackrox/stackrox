import React from 'react';

import NodeComponentsTable from './NodeComponentsTable';

const mockData = [
    {
        name: 'podman',
        operatingSystem: 'rhel',
        source: 'INFRASTRUCTURE',
    },
    {
        name: 'cri-o',
        operatingSystem: 'Ubuntu',
        source: 'KUBELET',
    },
    {
        name: 'kernel',
        operatingSystem: 'Debian',
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

            // Click the component header to sort by name ascending
            cy.get('th:contains("Component")').click();
            cy.get(componentCell).eq(0).should('have.text', 'podman');
            cy.get(componentCell).eq(1).should('have.text', 'kernel');
            cy.get(componentCell).eq(2).should('have.text', 'cri-o');

            // Click the component header to sort by name descending
            cy.get('th:contains("Component")').click();
            cy.get(componentCell).eq(0).should('have.text', 'cri-o');
            cy.get(componentCell).eq(1).should('have.text', 'kernel');
            cy.get(componentCell).eq(2).should('have.text', 'podman');
        });

        it('should sort by the type', () => {
            setup(mockData);

            const typeCell = 'td[data-label="Type"]';

            // Since this column is not the default sort, the starting sort will be descending
            // Click the type header to sort by type descending
            cy.get('th:contains("Type")').click();
            cy.get(typeCell).eq(0).should('have.text', 'KUBELET');
            cy.get(typeCell).eq(1).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(2).should('have.text', 'INFRASTRUCTURE');

            // Click the type header to sort by type ascending
            cy.get('th:contains("Type")').click();
            cy.get(typeCell).eq(0).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(1).should('have.text', 'INFRASTRUCTURE');
            cy.get(typeCell).eq(2).should('have.text', 'KUBELET');
        });

        it('should sort by the operating system', () => {
            setup(mockData);

            const osCell = 'td[data-label="Operating system"]';

            // Since this column is not the default sort, the starting sort will be descending
            // Click the operating system header to sort by operating system descending
            cy.get('th:contains("Operating system")').click();
            cy.get(osCell).eq(0).should('have.text', 'Ubuntu');
            cy.get(osCell).eq(1).should('have.text', 'rhel');
            cy.get(osCell).eq(2).should('have.text', 'Debian');

            // Click the operating system header to sort by operating system ascending
            cy.get('th:contains("Operating system")').click();
            cy.get(osCell).eq(0).should('have.text', 'Debian');
            cy.get(osCell).eq(1).should('have.text', 'rhel');
            cy.get(osCell).eq(2).should('have.text', 'Ubuntu');
        });
    });
});
