import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import SummaryCounts from './SummaryCounts';

const counts = {
    clusterCount: 2,
    nodeCount: 6,
    violationCount: 11,
    deploymentCount: 8,
    imageCount: 14,
    secretCount: 4,
};

function setup() {
    cy.intercept('GET', '/v1/clusters', { clusters: [{ id: 'c1' }, { id: 'c2' }] });
    cy.intercept('GET', '/v1/search?*', {
        results: [],
        counts: [{ category: 'NODES', count: String(counts.nodeCount) }],
    });
    cy.intercept('GET', '/v1/alertscount*', { count: counts.violationCount });
    cy.intercept('GET', '/v1/deploymentscount*', { count: counts.deploymentCount });
    cy.intercept('GET', '/v1/imagescount*', { count: counts.imageCount });
    cy.intercept('GET', '/v1/secretscount*', { count: counts.secretCount });

    cy.mount(
        <ComponentTestProvider>
            <SummaryCounts
                hasReadAccessForResource={{
                    Cluster: true,
                    Node: true,
                    Alert: true,
                    Deployment: true,
                    Image: true,
                    Secret: true,
                }}
            />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render REST summary counts for each permitted resource', () => {
        setup();

        cy.contains('2');
        cy.contains('Clusters');
        cy.contains('6');
        cy.contains('Nodes');
        cy.contains('11');
        cy.contains('Violations');
        cy.contains('8');
        cy.contains('Deployments');
        cy.contains('14');
        cy.contains('Images');
        cy.contains('4');
        cy.contains('Secrets');
        cy.contains(/Last updated/);
        cy.screenshot('after-summary-counts');
    });
});
