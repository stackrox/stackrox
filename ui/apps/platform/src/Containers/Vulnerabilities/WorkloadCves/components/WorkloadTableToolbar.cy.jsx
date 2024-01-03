import React from 'react';
import { createStore } from 'redux';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import {
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
} from 'Containers/Vulnerabilities/searchOptions';

import WorkloadTableToolbar from './WorkloadTableToolbar';

const readOnlyReduxStore = createStore((s) => s, {
    app: {
        featureFlags: {
            featureFlags: [],
            loading: false,
            error: null,
        },
        publicConfig: {
            publicConfig: {
                telemetry: null,
            },
        },
    },
});

function setup() {
    cy.intercept('POST', graphqlUrl('autocomplete'), (req) => {
        req.reply({ data: { searchAutocomplete: ['stackrox'] } });
    });

    cy.mount(
        <ComponentTestProviders reduxStore={readOnlyReduxStore}>
            <WorkloadTableToolbar
                searchOptions={[CLUSTER_SEARCH_OPTION, NAMESPACE_SEARCH_OPTION]}
            />
        </ComponentTestProviders>
    );
}

const searchOptionsDropdown = () => cy.findByLabelText('search options filter menu toggle');

describe(Cypress.spec.relative, () => {
    it('should correctly handle applied filters', () => {
        setup();

        // Set the entity type to 'Namespace'
        searchOptionsDropdown().click();
        cy.findByRole('option', { name: 'Namespace' }).click();
        searchOptionsDropdown().click();
        searchOptionsDropdown().should('have.text', 'Namespace');

        // Apply a namespace filter
        cy.findByRole('textbox').click();
        cy.findByRole('textbox').type('stackrox');
        cy.findByRole('option', { name: 'stackrox' }).click();
        cy.findByRole('textbox').click();

        // Apply a severity filter
        cy.findByText('CVE severity').click({ force: true });
        cy.get('label:contains("Critical") input[type="checkbox"]').click();
        cy.get('label:contains("Important") input[type="checkbox"]').click();
        cy.findByText('CVE severity').click({ force: true });

        // Check that the filters are applied in the toolbar chips
        cy.findByRole('group', { name: 'Namespace' }).within(() => {
            cy.get('li:contains("stackrox")');
        });

        cy.findByRole('group', { name: 'Severity' }).within(() => {
            cy.get('li:contains("Critical")');
            cy.get('li:contains("Important")');
            cy.get('li:contains("Moderate")').should('not.exist');
            cy.get('li:contains("Low")').should('not.exist');
        });

        // Test removing filters
        cy.get('li:contains("Important") button[aria-label="Remove filter"]').click();
        cy.get('li:contains("Important")').should('not.exist');

        // Clear remaining filters
        cy.findByText('Clear filters').click();

        // Check that the filters are removed from the toolbar chips
        cy.findByRole('group', { name: 'Severity' }).should('not.exist');
        cy.findByRole('group', { name: 'Namespace' }).should('not.exist');
    });
});
