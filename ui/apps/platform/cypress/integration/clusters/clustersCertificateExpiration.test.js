import cloneDeep from 'lodash/cloneDeep';

import withAuth from '../../helpers/basicAuth';

import {
    clusterAlias,
    visitClusterById,
    visitClusterByNameWithFixtureMetadataDatetime,
    visitClustersWithFixtureMetadataDatetime,
} from './Clusters.helpers';
import { selectors } from './Clusters.selectors';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Clusters Certificate Expiration', () => {
    withAuth();

    const metadata = {
        version: '3.0.50.0', // for comparison to `sensorVersion` in clusters fixture
        buildFlavor: 'release',
        releaseBuild: true,
        licenseStatus: 'VALID',
    };

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    describe('status is Healthy', () => {
        const fixturePath = 'clusters/certExpirationHealthy.json';

        it(`should not show link or form`, () => {
            [
                'healthy-kubectl-1',
                'healthy-kubectl-2',
                'healthy-helm-1',
                'healthy-helm-2',
                'healthy-operator-1',
            ].forEach((clusterName) => {
                visitClusterByNameWithFixtureMetadataDatetime(
                    clusterName,
                    fixturePath,
                    metadata,
                    currentDatetime
                );

                cy.get(selectors.clusterHealth.credentialExpiration).should(
                    'have.text',
                    'in 1 month'
                );
                cy.get(selectors.clusterHealth.reissueCertificatesLink).should('not.exist');
                cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
                cy.get(selectors.clusterHealth.manageTokensButton).should('not.exist');
            });
        });
    });

    describe('Sensor is not up to date with Central', () => {
        const expectedExpiration = 'in 6 days on Monday'; // Unhealthy
        const fixturePath = 'clusters/certExpirationUnhealthy.json';

        it('should disable the upgrade option', () => {
            const clusterName = 'unhealthy-kubectl-1';
            visitClusterByNameWithFixtureMetadataDatetime(
                clusterName,
                fixturePath,
                metadata,
                currentDatetime
            );

            cy.get(selectors.clusterHealth.credentialExpiration).should(
                'have.text',
                expectedExpiration
            );
            cy.get(selectors.clusterHealth.reissueCertificatesLink);
            cy.get(selectors.clusterHealth.downloadToReissueCertificate)
                .should('be.enabled')
                .should('be.checked');
            cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('be.disabled');
            cy.get(selectors.clusterHealth.reissueCertificateButton).should('be.enabled');
            cy.get(selectors.clusterHealth.manageTokensButton).should('not.exist');
        });

        // TODO mock Download YAML file for it('should display a message for success instead of the form')

        it('should display manage tokens button for helm and operator managed clusters when cluster is unhealthy', () => {
            ['unhealthy-helm-1', 'unhealthy-operator-1'].forEach((clusterName) => {
                visitClusterByNameWithFixtureMetadataDatetime(
                    clusterName,
                    fixturePath,
                    metadata,
                    currentDatetime
                );

                cy.get(selectors.clusterHealth.credentialExpiration).should(
                    'have.text',
                    expectedExpiration
                );
                cy.get(selectors.clusterHealth.reissueCertificatesLink);
                cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
                cy.get(selectors.clusterHealth.manageTokensButton);
            });
        });

        it('should not show link or form when no cert expiry', () => {
            const clusterName = 'not-applicable-1';
            visitClusterByNameWithFixtureMetadataDatetime(
                clusterName,
                fixturePath,
                metadata,
                currentDatetime
            );

            cy.get(selectors.clusterHealth.credentialExpiration).should(
                'have.text',
                'Not applicable'
            );
            cy.get(selectors.clusterHealth.reissueCertificatesLink).should('not.exist');
            cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
            cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
            cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
            cy.get(selectors.clusterHealth.manageTokensButton).should('not.exist');
        });
    });

    describe('Sensor is up to date with Central', () => {
        const expectedExpiration = 'in 29 days on 09/29/2020'; // Degraded
        const fixturePath = 'clusters/certExpirationDegraded.json';

        it('should enable the upgrade option', () => {
            const clusterName = 'degraded-kubectl-1';
            visitClusterByNameWithFixtureMetadataDatetime(
                clusterName,
                fixturePath,
                metadata,
                currentDatetime
            );

            cy.get(selectors.clusterHealth.credentialExpiration).should(
                'have.text',
                expectedExpiration
            );
            cy.get(selectors.clusterHealth.reissueCertificatesLink);
            cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('be.enabled');
            cy.get(selectors.clusterHealth.upgradeToReissueCertificate)
                .should('be.enabled')
                .should('be.checked');
            cy.get(selectors.clusterHealth.reissueCertificateButton).should('be.enabled');
        });

        it('should display a message for success instead of the form', () => {
            const clusterName = 'degraded-kubectl-1';
            visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, currentDatetime);

            cy.fixture(fixturePath).then(({ clusters }) => {
                const n = clusters.findIndex((cluster) => cluster.name === clusterName);
                const cluster = cloneDeep(clusters[n]);

                // Mock the result of using an automatic upgrade to re-issue the certificate.
                cluster.status.upgradeStatus.mostRecentProcess = {
                    type: 'CERT_ROTATION',
                    initiatedAt: currentDatetime,
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE',
                    },
                };

                const staticResponseMap = {
                    [clusterAlias]: {
                        body: { cluster, clusterRetentionInfo: null },
                    },
                };
                visitClusterById(cluster.id, staticResponseMap);

                cy.get(selectors.clusterHealth.credentialExpiration).should(
                    'have.text',
                    expectedExpiration
                );
                cy.get(selectors.clusterHealth.reissueCertificatesLink);
                cy.get(selectors.clusterHealth.upgradedToReissueCertificate).should(
                    'have.text',
                    'An automatic upgrade applied new credentials to the cluster 0 seconds ago.'
                );
                cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
            });
        });

        it('should display manage tokens button for helm and operator managed clusters when cluster is degraded', () => {
            ['degraded-helm-1', 'degraded-operator-1'].forEach((clusterName) => {
                visitClusterByNameWithFixtureMetadataDatetime(
                    clusterName,
                    fixturePath,
                    metadata,
                    currentDatetime
                );

                cy.get(selectors.clusterHealth.credentialExpiration).should(
                    'have.text',
                    expectedExpiration
                );
                cy.get(selectors.clusterHealth.reissueCertificatesLink);
                cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
                cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
                cy.get(selectors.clusterHealth.manageTokensButton);
            });
        });
    });
});
