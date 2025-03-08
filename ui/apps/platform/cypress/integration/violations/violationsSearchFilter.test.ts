import withAuth from '../../helpers/basicAuth';
import { selectFilteredWorkflowView, visitViolations } from './Violations.helpers';

function selectCompoundSearchFilterAttribute(attribute: string) {
    cy.get('button[aria-label="compound search filter attribute selector toggle"]').click();
    cy.get(
        `ul[aria-label="compound search filter attribute selector menu"] button:contains("${attribute}")`
    ).click();
}

function selectFirstAutocompleteSearchOption() {
    cy.get(
        'div[aria-labelledby="Filter results menu toggle"] button[aria-label="Menu toggle"]'
    ).click();
    cy.get(`ul[aria-label="Filter results select menu"] li:nth(0) button`).click();
}

describe('Violations - Search Filter', () => {
    withAuth();

    it('should filter violations by policy name', () => {
        visitViolations();

        selectFilteredWorkflowView('All Violations');

        // filter by the first policy name autocomplete option
        selectFirstAutocompleteSearchOption();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');
        cy.wait('@getViolations');

        // should filter using the correct query values
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Policy:');
        });
    });

    it('should filter violations by policy category', () => {
        visitViolations();

        selectFilteredWorkflowView('All Violations');

        // change to the policy category filter
        selectCompoundSearchFilterAttribute('Category');

        // filter by the first policy category autocomplete option
        selectFirstAutocompleteSearchOption();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');

        // should filter using the correct query values
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Category:');
        });
    });

    it('should filter violations by severity', () => {
        visitViolations();

        selectFilteredWorkflowView('All Violations');

        // change to the policy severity filter
        selectCompoundSearchFilterAttribute('Severity');

        // filter by high severity
        cy.get('button[aria-label="Filter by Severity"]').click();
        cy.get('div[aria-label="Filter by Severity select menu"] li')
            .contains('High')
            .parent()
            .find('input[type="checkbox"]')
            .check();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');
        cy.wait('@getViolations');

        // should filter using the correct query values
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Severity:');
        });
    });

    it('should filter violations by lifecycle stage', () => {
        visitViolations();

        selectFilteredWorkflowView('All Violations');

        // change to the policy lifecycle filter
        selectCompoundSearchFilterAttribute('Lifecycle');

        // filter by high severity
        cy.get('button[aria-label="Filter by Lifecycle stage"]').click();
        cy.get('div[aria-label="Filter by Lifecycle stage select menu"] li')
            .contains('Deploy')
            .parent()
            .find('input[type="checkbox"]')
            .check();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');
        cy.wait('@getViolations');

        // should filter using the correct query values
        cy.wait('@getViolations').then((interception) => {
            const queryString = interception.request.query.query;

            expect(queryString).to.contain('Lifecycle Stage:');
        });
    });
});
