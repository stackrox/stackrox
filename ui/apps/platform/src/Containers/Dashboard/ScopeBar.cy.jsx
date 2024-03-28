import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import ScopeBar from './ScopeBar';

const mockData = {
    clusters: [
        {
            id: `cluster-0`,
            name: 'production',
            namespaces: [
                { metadata: { id: 'ns-0', name: 'backend' } },
                { metadata: { id: 'ns-1', name: 'default' } },
                { metadata: { id: 'ns-2', name: 'frontend' } },
                { metadata: { id: 'ns-3', name: 'kube-system' } },
                { metadata: { id: 'ns-4', name: 'medical' } },
                { metadata: { id: 'ns-5', name: 'payments' } },
            ],
        },
        {
            id: `cluster-1`,
            name: 'security',
            namespaces: [
                { metadata: { id: 'ns-0', name: 'default' } },
                { metadata: { id: 'ns-1', name: 'kube-system' } },
                { metadata: { id: 'ns-2', name: 'stackrox' } },
            ],
        },
    ],
};

function setup() {
    cy.intercept('POST', graphqlUrl('getAllNamespacesByCluster'), (req) => {
        req.reply({ data: mockData });
    });

    // Work-around for cy.location('search') assertions.
    // Remove unexpected cypress specPath param before component code executes.
    // https://github.com/cypress-io/cypress/issues/28021#issuecomment-1756646215
    window.history.pushState({}, document.title, window.location.pathname);

    cy.mount(
        <ComponentTestProviders>
            <ScopeBar />
        </ComponentTestProviders>
    );
}

const clusterToggle = () => cy.findByLabelText('Select clusters');
const namespaceToggle = () => cy.findByLabelText('Select namespaces');
const dropdownOption = (name, opts) => cy.findByLabelText(name, opts);

describe(Cypress.spec.relative, () => {
    it('should default to all clusters and namespaces selected', () => {
        setup();

        // The default state is all clusters selected, with the ns dropdown disabled
        clusterToggle().should('be.enabled');
        clusterToggle().click();
        dropdownOption('All clusters', { selector: 'input' }).should('be.checked');
        namespaceToggle().should('be.disabled');
    });

    it('allows selection of multiple clusters and namespaces', () => {
        setup();

        // Wait for cluster data to load and the dropdown to be enabled
        clusterToggle().should('be.enabled');

        // Selecting one or more clusters enables the ns dropdown
        clusterToggle().click();
        dropdownOption('production').click();
        clusterToggle().click();
        namespaceToggle().should('be.enabled');
        clusterToggle().contains(/Clusters *1/);

        // Enable some namespaces and check that the select badge updates
        namespaceToggle().click();
        dropdownOption('frontend').click();
        dropdownOption('backend').click();
        dropdownOption('payments').click();
        namespaceToggle().click();
        namespaceToggle().contains(/Namespaces *3/);

        // Selecting another cluster when other namespaces are selected will explicitly select
        // all namespaces in that cluster
        clusterToggle().click();
        dropdownOption('security').click();
        clusterToggle().click();
        namespaceToggle().contains(/Namespaces *6/);

        // Selecting "All Namespaces" and then a single namespaces will result in a single
        // namespace being selected
        namespaceToggle().click();
        dropdownOption('All namespaces').click();
        dropdownOption('frontend').click();
        namespaceToggle().click();
        namespaceToggle().contains(/Namespaces *1/);

        // Selecting "All Clusters" will clear the namespace selection and disable the dropdown
        clusterToggle().click();
        dropdownOption('All clusters').click();
        clusterToggle().click();
        clusterToggle().contains('All clusters');
        namespaceToggle().contains('All namespaces');
        namespaceToggle().should('be.disabled');
    });

    it('will track selected clusters and namespaces in the page URL', () => {
        setup();

        // Check that the default state of "select all" results in empty URL search parameters
        clusterToggle().should('be.enabled');
        cy.location('search').should('eq', '');

        // Select a cluster and verify it has been added to the URL
        clusterToggle().click();
        dropdownOption('production').click();
        clusterToggle().click();
        cy.location('search', { decode: true }).should('eq', '?s[Cluster][0]=production');

        // Select multiple namespaces and verify they have been added to the URL
        const productionNamespaces = mockData.clusters.find(
            (cs) => cs.name === 'production'
        ).namespaces;

        function findNsWithName(name) {
            return ({ metadata }) => metadata.name === name;
        }

        // Get a reference to the mock data object for each namespace so we can compare names against ids
        const frontend = productionNamespaces.find(findNsWithName('frontend'));
        const backend = productionNamespaces.find(findNsWithName('backend'));

        namespaceToggle().click();
        dropdownOption('frontend').click();
        dropdownOption('backend').click();
        namespaceToggle().click();
        cy.location('search', { decode: true }).should(
            'eq',
            `?s[Cluster][0]=production&s[Namespace%20ID][0]=${frontend.metadata.id}&s[Namespace%20ID][1]=${backend.metadata.id}`
        );
    });
});
