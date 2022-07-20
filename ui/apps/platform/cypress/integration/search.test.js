import { selectors } from '../constants/SearchPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';

function visitSearch() {
    visitMainDashboard();

    cy.get(selectors.globalSearchButton).click();
}

function searchWithFixture(searchTuples, fixture) {
    cy.intercept('GET', api.search.results, { fixture }).as('getSearchResults');

    cy.get(selectors.globalSearch.input).clear();
    searchTuples.forEach(([category, value]) => {
        cy.get(selectors.globalSearch.input).type(`${category}{enter}`);
        cy.get(selectors.globalSearch.input).type(`${value}{enter}`);
    });

    cy.wait('@getSearchResults');
}

describe('Global Search Modal', () => {
    withAuth();

    it('should have empty state instead of count and tabs if search filter is empty', () => {
        visitSearch();

        cy.get(`${selectors.empty.head}:contains("Search all data")`).should('exist');
        cy.get(
            `${selectors.empty.body}:contains("Choose one or more filter values to search")`
        ).should('exist');

        cy.get(selectors.globalSearchResults.header).should('not.exist');
        cy.get(selectors.tab).should('not.exist');
    });

    it('Should have 6 tabs with the "All" tab selected by default', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(`${selectors.tab}:contains("All")`).should('have.class', 'pf-m-current');
        cy.get(`${selectors.tab}:contains("Violations")`).should('not.have.class', 'pf-m-current');
        cy.get(`${selectors.tab}:contains("Policies")`).should('not.have.class', 'pf-m-current');
        cy.get(`${selectors.tab}:contains("Deployments")`).should('not.have.class', 'pf-m-current');
        cy.get(`${selectors.tab}:contains("Images")`).should('not.have.class', 'pf-m-current');
        cy.get(`${selectors.tab}:contains("Secrets")`).should('not.have.class', 'pf-m-current');
    });

    it('Should filter search results', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(selectors.globalSearchResults.header).should('have.text', '4 search results');
    });

    it('Should send you to the Violations page', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(`section[aria-label="All"] ${selectors.viewOnChip}:contains("Violations")`).click();
        // TODO because 404 for /v1/alerts/6f68ef75-a96d-4121-ad89-92cf8cde0062
        // replace button with anchor and assert on href attribute?
        cy.location('pathname').should(
            'eq',
            '/main/violations/6f68ef75-a96d-4121-ad89-92cf8cde0062'
        );
    });

    it('Should send you to the Risk page', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(`section[aria-label="All"] ${selectors.viewOnChip}:contains("Risk")`).click();
        // TODO because 404 for /v1/deploymentswithrisk/ppqqu24i8x16j7annv2bjphyy
        // replace button with anchor and assert on href attribute?
        cy.location('pathname').should('eq', '/main/risk/ppqqu24i8x16j7annv2bjphyy');
    });

    it('Should send you to the Policies page', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(`section[aria-label="All"] ${selectors.viewOnChip}:contains("Policies")`).click();
        // TODO because 404 for /v1/policies/0ea8d235-b02a-41ee-a61d-edcb2c1b0eac
        // replace button with anchor and assert on href attribute?
        cy.location('pathname').should(
            'eq',
            '/main/policy-management/policies/0ea8d235-b02a-41ee-a61d-edcb2c1b0eac'
        );
    });

    it('Should send you to the Images page', () => {
        visitSearch();
        searchWithFixture([['Cluster:', 'remote']], 'search/globalSearchResults.json');

        cy.get(`section[aria-label="All"] ${selectors.viewOnChip}:contains("Images")`).click();
        // TODO because could not find image for id
        // replace button with anchor and assert on href attribute?
        cy.location('pathname').should(
            'eq',
            '/main/vulnerability-management/images/sha256:9342f82b178a4325aec19f997400e866bf7c6bf9d59dd74e1358f971159dd7b8'
        );
    });
});
