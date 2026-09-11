import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { violationsBasePath } from 'routePaths';

import ViolationsByPolicySeverity from './ViolationsByPolicySeverity';

const mostRecentAlerts = [
    {
        id: '1',
        time: '2022-06-24T00:35:42.299667447Z',
        commonEntityInfo: {
            resourceType: 'DEPLOYMENT',
            clusterName: 'production',
            namespace: 'kube-system',
            clusterId: 'cluster-1',
            namespaceId: 'ns-1',
        },
        deployment: {
            clusterName: 'production',
            namespace: 'kube-system',
            name: 'kube-proxy',
            id: 'dep-1',
            clusterId: 'cluster-1',
            inactive: false,
            namespaceId: 'ns-1',
        },
        policy: { name: 'Ubuntu Package Manager in Image', severity: 'CRITICAL_SEVERITY' },
    },
];

const alertsBySeverityCounts = {
    groups: [
        {
            group: '',
            counts: [
                { severity: 'LOW_SEVERITY', count: '220' },
                { severity: 'MEDIUM_SEVERITY', count: '70' },
                { severity: 'HIGH_SEVERITY', count: '140' },
                { severity: 'CRITICAL_SEVERITY', count: '3' },
            ],
        },
    ],
};

function setup() {
    cy.intercept('GET', '/v1/alerts/summary/counts*', (req) => {
        req.reply(alertsBySeverityCounts);
    });

    cy.intercept('GET', '/v1/alerts?*', (req) => {
        req.reply({ alerts: mostRecentAlerts });
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

        const alertCount = 220 + 70 + 140 + 3;

        cy.findByText(`${alertCount} policy violations by severity`).should('exist');
        cy.screenshot('after-violations-by-policy-severity');
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
        cy.location('pathname').should('eq', `${violationsBasePath}/${mostRecentAlerts[0].id}`);
    });
});
