import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';
import { violationsBasePath } from 'routePaths';

import ViolationsByPolicySeverity from './ViolationsByPolicySeverity';

const mostRecentAlertsMock = {
    data: {
        alerts: [
            {
                id: '1',
                time: '2022-06-24T00:35:42.299667447Z',
                deployment: {
                    clusterName: 'production',
                    namespace: 'kube-system',
                    name: 'kube-proxy',
                },
                resource: null,
                policy: { name: 'Ubuntu Package Manager in Image', severity: 'CRITICAL_SEVERITY' },
            },
        ],
    },
};

const alertsBySeverityMock = {
    data: {
        LOW_SEVERITY: 220,
        MEDIUM_SEVERITY: 70,
        HIGH_SEVERITY: 140,
        CRITICAL_SEVERITY: 3,
    },
};

function setup() {
    cy.intercept('POST', graphqlUrl('mostRecentAlerts'), (req) => {
        req.reply(mostRecentAlertsMock);
    });

    cy.intercept('POST', graphqlUrl('alertCountsBySeverity'), (req) => {
        req.reply(alertsBySeverityMock);
    });

    cy.mount(
        <ComponentTestProvider>
            <ViolationsByPolicySeverity />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('should display total violations in the title that match the sum of the individual tiles', () => {
        setup();

        cy.get('a').contains(/220\s*Low/);
        cy.get('a').contains(/70\s*Medium/);
        cy.get('a').contains(/140\s*High/);
        cy.get('a').contains(/3\s*Critical/);

        const { data } = alertsBySeverityMock;
        const alertCount =
            data.LOW_SEVERITY + data.MEDIUM_SEVERITY + data.HIGH_SEVERITY + data.CRITICAL_SEVERITY;

        cy.findByText(`${alertCount} policy violations by severity`).should('exist');
    });

    it('should link to the correct violations pages when clicking links in the widget', () => {
        setup();

        // Test the 'View all' violations link button
        cy.findByText('View all').click();
        cy.location('pathname').should('eq', `${violationsBasePath}`);
        cy.location('search').should('include', 'sortOption[direction]=desc');
        cy.location('search').should('include', 'sortOption[field]=Severity');

        // Test links from the violation count tiles
        cy.findByText('Low').click();
        cy.location('pathname').should('eq', `${violationsBasePath}`);
        cy.location('search').should('include', 's[Severity]=LOW_SEVERITY');

        cy.findByText('Critical').click();
        cy.location('pathname').should('eq', `${violationsBasePath}`);
        cy.location('search').should('include', 's[Severity]=CRITICAL_SEVERITY');

        // Test links from the 'most recent violations' section
        cy.findByText(/ubuntu package manager/i).click();
        cy.location('pathname').should(
            'eq',
            `${violationsBasePath}/${mostRecentAlertsMock.data.alerts[0].id}`
        );
    });
});
