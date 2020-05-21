import { selectors } from '../constants/SearchPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';

describe('Global Search Modal', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.search.globalSearchWithNoResults, []).as('globalSearchResults');
        cy.fixture('search/globalSearchResults.json').as('globalSearchResultsJson');
        cy.route('GET', api.search.globalSearchWithResults, '@globalSearchResultsJson').as(
            'globalSearchResults'
        );
        cy.fixture('search/metadataOptions.json').as('metadataOptionsJson');
        cy.route('GET', api.search.options, '@metadataOptionsJson').as('globalSearchOptions');
        cy.visit('/main/dashboard');
        cy.get(selectors.globalSearchButton).click();
    });

    it('Should have 6 tabs with the "All" tab selected by default', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.allTab).should('have.class', 'border-primary-400');
        cy.get(selectors.violationsTab).should('not.have.class', 'border-primary-400');
        cy.get(selectors.policiesTab).should('not.have.class', 'border-primary-400');
        cy.get(selectors.deploymentsTab).should('not.have.class', 'border-primary-400');
        cy.get(selectors.imagesTab).should('not.have.class', 'border-primary-400');
        cy.get(selectors.secretsTab).should('not.have.class', 'border-primary-400');
    });

    it('Should filter search results', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.globalSearchResults.header).should('not.have.text', '0 search results');
    });

    it('Should send you to the Violations page', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.viewOnViolationsLabelChip).click();
        cy.location('pathname').should(
            'eq',
            '/main/violations/6f68ef75-a96d-4121-ad89-92cf8cde0062'
        );
    });

    it('Should send you to the Risk page', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.viewOnRiskLabelChip).click();
        cy.location('pathname').should('eq', '/main/risk/ppqqu24i8x16j7annv2bjphyy');
    });

    it('Should send you to the Policies page', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.viewOnPoliciesLabelChip).click();
        cy.location('pathname').should('eq', '/main/policies/0ea8d235-b02a-41ee-a61d-edcb2c1b0eac');
    });

    it('Should send you to the Images page', () => {
        cy.wait('@globalSearchOptions');
        cy.get(selectors.globalSearch.input).type('Cluster:{enter}');
        cy.get(selectors.globalSearch.input).type('remote{enter}');
        cy.get(selectors.viewOnImagesLabelChip).click();
        cy.location('pathname').should(
            'eq',
            '/main/vulnerability-management/images/sha256:9342f82b178a4325aec19f997400e866bf7c6bf9d59dd74e1358f971159dd7b8'
        );
    });
});
