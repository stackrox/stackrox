import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';

import { Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import TbodyUnified from './TbodyUnified';

function setup(tableState, otherProps) {
    cy.mount(
        <ComponentTestProviders>
            <Table>
                <Thead>
                    <Tr>
                        <Th>Test Column</Th>
                    </Tr>
                </Thead>
                <TbodyUnified tableState={tableState} {...otherProps} />
            </Table>
        </ComponentTestProviders>
    );
}

const data = [{ value: 'Test Value' }];
const setupProps = {
    renderer: ({ data }) => (
        <Tbody>
            <Tr>
                <Td>{data[0].value}</Td>
            </Tr>
        </Tbody>
    ),
};

describe(Cypress.spec.relative, () => {
    it('should render the idle state', () => {
        setup({ type: 'IDLE', data }, setupProps);

        // We don't define an IDLE state at this point, so only the header should be visible
        cy.findAllByRole('row').should('have.length', 1);
        cy.findByText('Test Value').should('not.exist');
    });

    it('should render the loading state', () => {
        setup({ type: 'LOADING', data }, setupProps);

        cy.findAllByRole('row').should('have.length', 2);
        cy.findByRole('progressbar');
        cy.findByText('Test Value').should('not.exist');
    });

    it('should render the error state', () => {
        setup({ type: 'ERROR', error: new Error('Error fetching data'), data }, setupProps);

        cy.findAllByRole('row').should('have.length', 2);
        cy.findByText('Error fetching data');
        cy.findByText('Test Value').should('not.exist');
    });

    it('should render the empty state', () => {
        setup(
            {
                type: 'EMPTY',
                data,
            },
            {
                ...setupProps,
                emptyProps: { message: 'No entities exist' },
            }
        );

        cy.findAllByRole('row').should('have.length', 2);
        cy.findByText('No entities exist');
        cy.findByText('Test Value').should('not.exist');
    });

    it('should render the filtered empty state', () => {
        setup(
            {
                type: 'FILTERED_EMPTY',
                data,
            },
            {
                ...setupProps,
                filteredEmptyProps: {
                    message: 'No entities were found with the applied filters',
                },
            }
        );

        cy.findAllByRole('row').should('have.length', 2);
        cy.findByRole('button', { name: 'Clear filters' });
        cy.findByText('No entities were found with the applied filters');
        cy.findByText('Test Value').should('not.exist');
    });

    it('should render the success state', () => {
        setup({ type: 'COMPLETE', data }, setupProps);

        cy.findAllByRole('row').should('have.length', 2);
        cy.findByText('Test Value');
    });
});
