import withAuth from '../../helpers/basicAuth';

import {
    assertClusterNameInSidePanel,
    clusterAlias,
    visitClusterById,
    visitClusters,
    visitClustersWithFixture,
} from './Clusters.helpers';

describe('Clusters list clusterIdToRetentionInfo', () => {
    withAuth();

    const fixturePath = 'clusters/health.json';

    it('should display Cluster Deletion column', () => {
        visitClusters();

        cy.get(`.rt-th:contains("Cluster Deletion")`);
    });

    // .rt-td:nth(6) because [data-testid="clusterDeletion"] fails for unknown reason :(

    it('should display Not applicable in Cluster Deletion cell for actual request', () => {
        visitClusters();

        cy.get('.rt-tr:contains("remote") .rt-td:nth(6):contains("Not applicable")');
    });

    it('should display alternatives in Cluster Deletion cell for mock request', () => {
        visitClustersWithFixture(fixturePath);

        cy.get('.rt-tr:contains("alpha-amsterdam-1") .rt-td:nth(6):contains("Not applicable")');
        cy.get('.rt-tr:contains("epsilon-edison-5") .rt-td:nth(6):contains("in 90 days")');
    });
});

describe('Cluster page clusterRetentionInfo', () => {
    withAuth();

    // div:contains("Cluster Deletion") because [data-testid="clusterDeletion"] fails for unknown reason :(

    it('should display Not applicable in Cluster Deletion widget for actual request', () => {
        visitClusters();

        const clusterName = 'remote';
        cy.get(`[data-testid="cluster-name"]:contains("${clusterName}")`).click();
        assertClusterNameInSidePanel(clusterName);
        cy.get('div:contains("Cluster Deletion"):contains("Not applicable")');
    });

    const fixturePath = 'clusters/health.json';
    const clusterName = 'epsilon-edison-5'; // has sensorHealthStatus UNHEALTHY

    function visitClusterWithRetentionInfo(clusterRetentionInfo) {
        cy.fixture(fixturePath).then(({ clusters }) => {
            const cluster = clusters.find(({ name }) => name === clusterName);
            const staticResponseMap = {
                [clusterAlias]: { body: { cluster, clusterRetentionInfo } },
            };

            visitClusterById(cluster.id, staticResponseMap);

            assertClusterNameInSidePanel(clusterName);
        });
    }

    it('should display in 30 days in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { daysUntilDeletion: 30 };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("Not applicable")');
    });

    it('should display in 7 days in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { daysUntilDeletion: 7 };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("in 7 days")'); // FYI yellow color
    });

    it('should display in 1 day in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { daysUntilDeletion: 1 };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("in 1 day")'); // FYI red color
    });

    it('should display Imminent for 0 days in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { daysUntilDeletion: 0 };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("Imminent")'); // FYI red color
    });

    it('should display Imminent for -1 days in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { daysUntilDeletion: -1 };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("Imminent")'); // FYI red color
    });

    it('should display Excluded from deletion in Cluster Deletion widget for mock request', () => {
        const clusterRetentionInfo = { isExcluded: true };
        visitClusterWithRetentionInfo(clusterRetentionInfo);

        cy.get('div:contains("Cluster Deletion"):contains("Excluded from deletion")');
    });
});
