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

    beforeEach(() => {
        // clear the filters before every test
        cy.document().then((doc) => {
            const button = Cypress.$('button:contains("Clear filters")', doc);
            if (button.length > 0) {
                // if button exists, click it
                cy.wrap(button).click();
            }
        });
    });

    it('should filter violations by policy name', () => {
        visitViolations();

        selectFilteredWorkflowView('Full view');

        // filter by the first policy name autocomplete option
        selectFirstAutocompleteSearchOption();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');
        cy.wait('@getViolations');

        cy.get('.search-filter-chips .pf-v5-c-chip .pf-v5-c-chip__text')
            .invoke('text')
            .then((textValue) => {
                cy.get('table tbody tr').each(($row) => {
                    cy.wrap($row)
                        .find('td[data-label="Policy"] a')
                        .invoke('text')
                        .then((linkText) => {
                            expect(linkText.trim()).to.equal(textValue);
                        });
                });
            });
    });

    it('should filter violations by policy category', () => {
        visitViolations();

        selectFilteredWorkflowView('Full view');

        // change to the policy category filter
        selectCompoundSearchFilterAttribute('Category');

        // filter by the first policy category autocomplete option
        selectFirstAutocompleteSearchOption();

        cy.intercept('GET', '/v1/alerts?query=*').as('getViolations');
        cy.wait('@getViolations');
        // this accounts for a slight delay in the table results being updated after a successful fetch
        // this test was the only one that would sometimes fail without this "wait"
        cy.wait(1000);

        cy.get('.search-filter-chips .pf-v5-c-chip .pf-v5-c-chip__text')
            .invoke('text')
            .then((textValue) => {
                cy.get('table tbody tr').each(($row) => {
                    cy.wrap($row)
                        .find('td[data-label="Categories"]')
                        .invoke('text')
                        .then((linkText) => {
                            expect(linkText.trim()).to.equal(textValue);
                        });
                });
            });
    });

    it('should filter violations by severity', () => {
        visitViolations();

        selectFilteredWorkflowView('Full view');

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

        cy.get('.search-filter-chips .pf-v5-c-chip .pf-v5-c-chip__text')
            .invoke('text')
            .then((textValue) => {
                cy.get('table tbody tr').each(($row) => {
                    cy.wrap($row)
                        .find('td[data-label="Severity"]')
                        .invoke('text')
                        .then((linkText) => {
                            expect(linkText.trim()).to.equal(textValue);
                        });
                });
            });
    });

    it('should filter violations by lifecycle stage', () => {
        visitViolations();

        selectFilteredWorkflowView('Full view');

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

        cy.get('.search-filter-chips .pf-v5-c-chip .pf-v5-c-chip__text')
            .invoke('text')
            .then((textValue) => {
                cy.get('table tbody tr').each(($row) => {
                    cy.wrap($row)
                        .find('td[data-label="Lifecycle"]')
                        .invoke('text')
                        .then((linkText) => {
                            expect(linkText.trim()).to.equal(textValue);
                        });
                });
            });
    });
});
