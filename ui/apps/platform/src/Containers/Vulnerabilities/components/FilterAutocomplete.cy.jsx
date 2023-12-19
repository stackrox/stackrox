import React from 'react';
import { Router } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { createBrowserHistory } from 'history';
import configureApolloClient from 'configureApolloClient';

import FilterAutocomplete from './FilterAutocomplete';
import { IMAGE_CVE_SEARCH_OPTION, IMAGE_SEARCH_OPTION } from '../searchOptions';
import useURLSearch from 'hooks/useURLSearch';

const cveResponseMock = {
    data: {
        searchAutocomplete: ['CVE-123', 'CVE-456', 'RHSA-123', 'RHSA-456'],
    },
};

const imageResponseMock = {
    data: {
        searchAutocomplete: [
            'docker.io/library/centos:7',
            'docker.io/library/centos:8',
            'quay.io/centos:7',
        ],
    },
};

function Wrapper({ searchOptions }) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    return (
        <FilterAutocomplete
            searchFilter={searchFilter}
            setSearchFilter={setSearchFilter}
            searchOptions={searchOptions}
        />
    );
}

function setup(searchOptions) {
    const history = createBrowserHistory();
    cy.mount(
        <Router history={history}>
            <ApolloProvider client={configureApolloClient()}>
                <Wrapper searchOptions={searchOptions} />
            </ApolloProvider>
        </Router>
    );
}

function mockAutocompleteResponse() {
    cy.intercept('POST', '/api/graphql?opname=autocomplete', (req) => {
        if (req.body.query.includes('CVE')) {
            req.reply(cveResponseMock);
        } else {
            req.reply(imageResponseMock);
        }
    }).as('autocomplete');
}

describe(Cypress.spec.relative, () => {
    it.only('should debounce search requests as the user types', () => {
        mockAutocompleteResponse();
        setup([IMAGE_CVE_SEARCH_OPTION, IMAGE_SEARCH_OPTION]);

        // No request should be made until the search box is interacted with
        cy.get('@autocomplete.all').should('have.length', 0);

        // A single request should be made after the user opens the dropdown
        cy.findByRole('textbox', { name: 'Filter by CVE' }).click();
        cy.get('@autocomplete.all').should('have.length', 1);
        cy.get('@autocomplete').then(({ request }) => {
            expect(request.body.variables.query).to.equal('CVE:');
        });

        // No additional requests should be made as the user is typing
        'CVE'.split('').forEach((char) => {
            cy.findByRole('textbox', { name: 'Filter by CVE' }).type(char);
            cy.get('@autocomplete.all').should('have.length', 1);
        });

        // Another request should be made after the user stops typing
        cy.get('@autocomplete.all').should('have.length', 2);
        cy.get('@autocomplete').then(({ request }) => {
            expect(request.body.variables.query).to.equal('CVE:CVE');
        });

        // No additional requests should be made as the user is typing
        '-123'.split('').forEach((char) => {
            cy.findByRole('textbox', { name: 'Filter by CVE' }).type(char);
            cy.get('@autocomplete.all').should('have.length', 2);
        });

        // Another request should be made after the user stops typing
        cy.get('@autocomplete.all').should('have.length', 3);
        cy.get('@autocomplete').then(({ request }) => {
            expect(request.body.variables.query).to.equal('CVE:CVE-123');
        });

        // Change the search category
        cy.findByLabelText('search options filter menu toggle').click();
        cy.findByText(IMAGE_SEARCH_OPTION.label).click();

        // Opening the dropdown for a new category should make a new request
        cy.findByRole('textbox', { name: 'Filter by Image' }).click();
        cy.get('@autocomplete.all').should('have.length', 4);
        cy.get('@autocomplete').then(({ request }) => {
            expect(request.body.variables.query).to.equal('IMAGE:');
        });

        // No additional requests should be made as the user is typing
        'docker.io'.split('').forEach((char) => {
            cy.findByRole('textbox', { name: 'Filter by Image' }).type(char);
            cy.get('@autocomplete.all').should('have.length', 4);
        });

        // Another request should be made after the user stops typing
        cy.get('@autocomplete.all').should('have.length', 5);
        cy.get('@autocomplete').then(({ request }) => {
            expect(request.body.variables.query).to.equal('IMAGE:docker.io');
        });
    });
});
