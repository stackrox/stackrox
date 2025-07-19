import React from 'react';
import { noop } from 'lodash';

import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import useURLSearch from 'hooks/useURLSearch';
import DiscoveredClustersToolbar from './DiscoveredClustersToolbar';

function Wrapper() {
    const { searchFilter, setSearchFilter } = useURLSearch();

    return (
        <DiscoveredClustersToolbar
            searchFilter={searchFilter}
            setSearchFilter={setSearchFilter}
            count={0}
            page={1}
            perPage={1}
            setPage={noop}
            setPerPage={noop}
        />
    );
}

const setup = () => {
    // Work-around for cy.location('search') assertions.
    // Remove unexpected cypress specPath param before component code executes.
    // https://github.com/cypress-io/cypress/issues/28021#issuecomment-1756646215
    window.history.pushState({}, document.title, window.location.pathname);

    cy.mount(
        <ComponentTestProvider>
            <Wrapper />
        </ComponentTestProvider>
    );
};

const openDropdownAnd = (label, action) => {
    cy.findByLabelText(label).click();
    cy.findByLabelText(label).parent().within(action);
    cy.findByLabelText(label).click();
};

const filterGroup = (name) => cy.findByRole('group', { name });
const filterChip = (label) => cy.get(`li:has(*:contains("${label}"))`);
const removeGroup = (name) =>
    filterGroup(name).within(() => cy.findByLabelText('Close chip group').click());
// Comment out for 4.4 MVP because testers expected partial match instead of exact match.
/*
const removeChip = (name) =>
    filterChip(name).within(() => cy.findByLabelText('Remove filter').click());
*/

describe(Cypress.spec.relative, () => {
    // Skip for 4.4 MVP because testers expected partial match instead of exact match.
    it.skip('should correctly handle the name text input', () => {
        setup();

        // Ensure creating a chip/filter clears the input
        cy.findByLabelText('Filter by name').type('cluster-a{enter}');
        filterGroup('Name').within(() => filterChip('cluster-a'));
        cy.findByLabelText('Filter by name').should('have.value', '');

        // Ensure duplicate filters are not created
        cy.findByLabelText('Filter by name').type('cluster-a{enter}');
        filterGroup('Name').within(() => filterChip('cluster-a').should('have.length', 1));

        // clear filters
        removeGroup('Name');
    });

    it('should handle the "select all" checkbox mutual exclusion in the status and type dropdowns', () => {
        setup();

        // Simple abstraction to handle each dropdown
        const dropdowns = [
            {
                name: 'Status',
                label: 'Status filter menu toggle',
                allOption: 'All statuses',
                options: ['Unsecured', 'Undetermined'],
            },
            {
                name: 'Type',
                label: 'Type filter menu toggle',
                allOption: 'All types',
                options: ['AKS', 'EKS'],
            },
        ];

        dropdowns.forEach(({ label, name, allOption, options: [optionA, optionB] }) => {
            // Select all in the dropdown is the default
            openDropdownAnd(label, () => {
                cy.findByLabelText(allOption).should('be.checked');
            });

            filterGroup(name).should('not.exist');

            // Select a single option and verify that the "all" checkbox is deselected and the chip appears
            openDropdownAnd(label, () => {
                cy.findByLabelText(optionA).click();
                cy.findByLabelText(optionA).should('be.checked');
                cy.findByLabelText(allOption).should('not.be.checked');
            });

            filterGroup(name).within(() => {
                filterChip(optionA);
            });

            // Clicking "all" should remove all other applied filters
            openDropdownAnd(label, () => {
                cy.findByLabelText(allOption).click();
                cy.findByLabelText(allOption).should('be.checked');
                cy.findByLabelText(optionA).should('not.be.checked');
            });

            filterGroup(name).should('not.exist');

            // Deselecting all filters manually should re-select the "all" checkbox
            openDropdownAnd(label, () => {
                cy.findByLabelText(optionA).click();
                cy.findByLabelText(optionB).click();
            });

            filterGroup(name).within(() => {
                filterChip(optionA);
                filterChip(optionB);
            });

            openDropdownAnd(label, () => {
                cy.findByLabelText(optionA).click();
                cy.findByLabelText(optionB).click();
                cy.findByLabelText(allOption).should('be.checked');
                cy.findByLabelText(optionA).should('not.be.checked');
                cy.findByLabelText(optionB).should('not.be.checked');
            });

            filterGroup(name).should('not.exist');
        });
    });

    it('should correctly handle adding and removing filters', () => {
        setup();

        // Default is no filters
        cy.location('search').should('match', /^$/);

        // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
        /*
        // Add filters of each type
        cy.findByLabelText('Filter by name').type('cluster-a{enter}');
        cy.findByLabelText('Filter by name').type('cluster-b{enter}');
        */

        openDropdownAnd('Status filter menu toggle', () => {
            cy.findByLabelText('Unsecured').click();
            cy.findByLabelText('Undetermined').click();
        });

        openDropdownAnd('Type filter menu toggle', () => {
            cy.findByLabelText('AKS').click();
            cy.findByLabelText('EKS').click();
        });

        // Verify that all filters are applied and visible as chips
        // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
        /*
        filterGroup('Name').within(() => {
            filterChip('cluster-a');
            filterChip('cluster-b');
        });
        */

        filterGroup('Status').within(() => {
            filterChip('Unsecured');
            filterChip('Undetermined');
        });

        filterGroup('Type').within(() => {
            filterChip('AKS');
            filterChip('EKS');
        });

        // Verify that the filters exist in the URL
        [
            // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
            /*
            /s\[Cluster\]\[\d\]=cluster-a/,
            /s\[Cluster\]\[\d\]=cluster-b/,
            */
            /s\[Cluster%20Status\]\[\d\]=STATUS_UNSECURED/,
            /s\[Cluster%20Status\]\[\d\]=STATUS_UNSPECIFIED/,
            /s\[Cluster%20Type\]\[\d\]=AKS/,
            /s\[Cluster%20Type\]\[\d\]=EKS/,
        ].forEach((filter) => {
            cy.location('search').should('match', filter);
        });

        // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
        /*
        // Remove some filters and verify that the chips are removed
        filterGroup('Name').within(() => {
            removeChip('cluster-b');
        });
        */

        removeGroup('Type');

        // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
        /*
        // Verify that the correct filters are removed
        filterGroup('Name').within(() => {
            filterChip('cluster-a');
            filterChip('cluster-b').should('not.exist');
        });
        */

        filterGroup('Status').within(() => {
            filterChip('Unsecured');
            filterChip('Undetermined');
        });

        filterGroup('Type').should('not.exist');

        // Verify the next URL state
        [
            // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
            /*
            /s\[Cluster\]\[\d\]=cluster-a/,
            */
            /s\[Cluster%20Status\]\[\d\]=STATUS_UNSECURED/,
            /s\[Cluster%20Status\]\[\d\]=STATUS_UNSPECIFIED/,
        ].forEach((filter) => {
            cy.location('search').should('match', filter);
        });

        // Remove all filters and verify that the chips are removed
        cy.findByRole('button', { name: 'Clear filters' }).click();

        // Comment out for 4.4 MVP because testers expected partial match instead of exact match.
        /*
        filterGroup('Name').should('not.exist');
        */
        filterGroup('Status').should('not.exist');
        filterGroup('Type').should('not.exist');

        // Verify the next URL state
        cy.location('search').should('match', /^$/);
    });
});
