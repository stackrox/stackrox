import { useState } from 'react';

import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import AdministrationEventsToolbar from './AdministrationEventsToolbar';

const eventFilters = {
    'Event Domain': ['Image Scanning'],
    'Event Level': ['ADMINISTRATION_EVENT_LEVEL_ERROR'],
    'Resource Type': ['Image'],
    from: '2026-11-01T00:00:00Z',
    until: '2026-11-11T23:59:59Z',
};

function TestToolbar({ onFilterChange }) {
    const [searchFilter, setSearchFilter] = useState({ ...eventFilters, Cluster: ['prod'] });

    return (
        <AdministrationEventsToolbar
            count={0}
            countAvailable={0}
            isDisabled={false}
            lastUpdatedTime=""
            page={1}
            perPage={10}
            setPage={() => {}}
            setPerPage={() => {}}
            searchFilter={searchFilter}
            setSearchFilter={(newFilter) => {
                setSearchFilter(newFilter);
                onFilterChange(newFilter);
            }}
            updateEvents={() => {}}
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('preserves event filters when removing workload labels', () => {
        cy.intercept('POST', graphqlUrl('autocomplete'), { data: { searchAutocomplete: [] } });
        const onFilterChange = cy.spy().as('onFilterChange');
        cy.mount(
            <ComponentTestProvider>
                <TestToolbar onFilterChange={onFilterChange} />
            </ComponentTestProvider>
        );

        cy.findByRole('list', { name: 'Cluster name' }).should('contain.text', 'prod');
        cy.findByRole('button', { name: 'Clear filters' }).should('not.exist');
        cy.findByRole('list', { name: 'Cluster name' })
            .findByRole('button', { name: 'Remove filter' })
            .click();

        cy.get('@onFilterChange').should('have.been.calledOnceWithExactly', {
            ...eventFilters,
            Cluster: [],
        });
        cy.findByRole('list', { name: 'Cluster name' }).should('not.exist');
        cy.findByRole('button', { name: 'Domain filter menu toggle' }).should(
            'contain.text',
            'Image Scanning'
        );
        cy.findByRole('button', { name: 'Level filter menu toggle' }).should(
            'contain.text',
            'Error'
        );
        cy.findByRole('button', { name: 'Resource type filter menu toggle' }).should(
            'contain.text',
            'Image'
        );
    });
});
