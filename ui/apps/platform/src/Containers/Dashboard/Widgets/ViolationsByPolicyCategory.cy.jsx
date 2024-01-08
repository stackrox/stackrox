import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';

import ViolationsByPolicyCategory from './ViolationsByPolicyCategory';

function makeFixtureCounts(crit, high, med, low) {
    return [
        { severity: 'CRITICAL_SEVERITY', count: `${crit}` },
        { severity: 'HIGH_SEVERITY', count: `${high}` },
        { severity: 'MEDIUM_SEVERITY', count: `${med}` },
        { severity: 'LOW_SEVERITY', count: `${low}` },
    ];
}

const mock = {
    groups: [
        { counts: makeFixtureCounts(5, 20, 30, 10), group: 'Anomalous Activity' },
        { counts: makeFixtureCounts(5, 2, 30, 5), group: 'Docker CIS' },
        { counts: makeFixtureCounts(10, 20, 5, 5), group: 'Network Tools' },
        { counts: makeFixtureCounts(15, 2, 10, 5), group: 'Security Best Practices' },
        { counts: makeFixtureCounts(20, 10, 2, 10), group: 'Privileges' },
        { counts: makeFixtureCounts(15, 8, 10, 5), group: 'Vulnerability Management' },
    ],
};

beforeEach(() => {
    localStorage.clear();
});

function setup() {
    cy.intercept('GET', '/v1/alerts/summary/counts*', (req) => req.reply(mock));

    cy.mount(
        <ComponentTestProviders>
            <ViolationsByPolicyCategory />
        </ComponentTestProviders>
    );
}

describe(Cypress.spec.relative, () => {
    it('should sort a policy violations by category widget by severity and volume of violations', () => {
        setup();

        cy.findByText('Anomalous Activity');

        // Default sorting should be by severity of critical and high Violations, with critical taking priority.
        cy.get('svg a:eq(0)').should('have.text', 'Anomalous Activity');
        cy.get('svg a:eq(1)').should('have.text', 'Network Tools');
        cy.get('svg a:eq(2)').should('have.text', 'Security Best Practices');
        cy.get('svg a:eq(3)').should('have.text', 'Vulnerability Management');
        cy.get('svg a:eq(4)').should('have.text', 'Privileges');

        // Switch to sort-by-volume, which orders the chart by total violations per category
        cy.findByLabelText('Options').click();
        cy.findByText('Total').click();
        cy.findByLabelText('Options').click();

        cy.get('svg a:eq(0)').should('have.text', 'Security Best Practices');
        cy.get('svg a:eq(1)').should('have.text', 'Vulnerability Management');
        cy.get('svg a:eq(2)').should('have.text', 'Anomalous Activity');
        cy.get('svg a:eq(3)').should('have.text', 'Privileges');
        cy.get('svg a:eq(4)').should('have.text', 'Network Tools');
    });

    it('should allow toggling of severities for a policy violations by category widget', () => {
        setup();

        cy.findByText('Anomalous Activity');

        // Sort by volume, so that enabling lower severity bars changes the order of the chart
        cy.findByLabelText('Options').click();
        cy.findByText('Total').click();
        cy.findByLabelText('Options').click();

        // Toggle on low and medium violations, which are disabled by default
        cy.findByText('Low').click();
        cy.findByText('Medium').click();

        cy.get('svg a:eq(0)').should('have.text', 'Vulnerability Management');
        cy.get('svg a:eq(1)').should('have.text', 'Network Tools');
        cy.get('svg a:eq(2)').should('have.text', 'Privileges');
        cy.get('svg a:eq(3)').should('have.text', 'Docker CIS');
        cy.get('svg a:eq(4)').should('have.text', 'Anomalous Activity');
    });

    it('should contain a button that resets the widget options to default', () => {
        setup();

        const getButton = (name) => cy.findByRole('button', { name });

        cy.findByLabelText('Options').click();

        // Defaults
        getButton('Severity').should('have.attr', 'aria-pressed', 'true');
        getButton('Total').should('have.attr', 'aria-pressed', 'false');
        getButton('All').should('have.attr', 'aria-pressed', 'true');
        getButton('Deploy').should('have.attr', 'aria-pressed', 'false');
        getButton('Runtime').should('have.attr', 'aria-pressed', 'false');

        // Change some options
        getButton('Total').click();
        getButton('Runtime').click();

        getButton('Severity').should('have.attr', 'aria-pressed', 'false');
        getButton('Total').should('have.attr', 'aria-pressed', 'true');
        getButton('All').should('have.attr', 'aria-pressed', 'false');
        getButton('Deploy').should('have.attr', 'aria-pressed', 'false');
        getButton('Runtime').should('have.attr', 'aria-pressed', 'true');

        cy.findByLabelText('Revert to default options').click();
        cy.findByLabelText('Options').click();

        getButton('Severity').should('have.attr', 'aria-pressed', 'true');
        getButton('Total').should('have.attr', 'aria-pressed', 'false');
        getButton('All').should('have.attr', 'aria-pressed', 'true');
        getButton('Deploy').should('have.attr', 'aria-pressed', 'false');
        getButton('Runtime').should('have.attr', 'aria-pressed', 'false');
    });
});
