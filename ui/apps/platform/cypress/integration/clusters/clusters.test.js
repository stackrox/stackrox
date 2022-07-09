import cloneDeep from 'lodash/cloneDeep';

import { selectors } from '../../constants/ClustersPage';
import { clusters as clustersApi } from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    visitClusters,
    visitClustersFromLeftNav,
    visitClustersWithFixtureMetadataDatetime,
    visitClusterByNameWithFixture,
    visitClusterByNameWithFixtureMetadataDatetime,
} from '../../helpers/clusters';

describe('Clusters page', () => {
    withAuth();

    describe('smoke tests', () => {
        it('should be linked in the Platform Configuration menu', () => {
            visitClustersFromLeftNav();
        });

        it('should have a toggle control for the auto-upgrade setting', () => {
            visitClusters();

            cy.get(selectors.autoUpgradeInput);
        });

        it('should display all the columns expected in clusters list page', () => {
            visitClusters();

            [
                'Name',
                'Cloud Provider',
                'Cluster Status',
                'Sensor Upgrade',
                'Credential Expiration',
                // Cluster Deletion: see clusterDeletion.test.js
            ].forEach((heading, index) => {
                /*
                 * Important: nth is pseudo selector for zero-based index of matching cells.
                 * Do not use the one-based nth-child selector,
                 * because tableHeadingCell does not match cells which have first-child and hidden class.
                 */
                cy.get(
                    `${selectors.clusters.tableHeadingCell}:nth(${index}):contains("${heading}")`
                );
            });
        });
    });
});

describe.skip('Cluster Certificate Expiration', () => {
    withAuth();

    const fixturePath = 'clusters/health.json';

    const metadata = {
        version: '3.0.50.0', // for comparison to `sensorVersion` in clusters fixture
        buildFlavor: 'release',
        releaseBuild: true,
        licenseStatus: 'VALID',
    };

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    describe('Credential Expiration status is Healthy', () => {
        it('should not show link or form', () => {
            const clusterName = 'kappa-kilogramme-10';
            visitClusterByNameWithFixtureMetadataDatetime(
                clusterName,
                fixturePath,
                metadata,
                currentDatetime
            );

            cy.get(selectors.clusterHealth.credentialExpiration).should('have.text', 'in 1 month');
            cy.get(selectors.clusterHealth.reissueCertificatesLink).should('not.exist');
            cy.get(selectors.clusterHealth.downloadToReissueCertificate).should('not.exist');
            cy.get(selectors.clusterHealth.upgradeToReissueCertificate).should('not.exist');
            cy.get(selectors.clusterHealth.reissueCertificateButton).should('not.exist');
        });
    });

    describe('Sensor is not up to date with Central', () => {
        const expectedExpiration = 'in 6 days on Monday'; // Unhealthy

        it('should disable the upgrade option', () => {
            const clusterName = 'epsilon-edison-5';
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
        });

        // TODO mock Download YAML file for it('should display a message for success instead of the form')
    });

    describe('Sensor is up to date with Central', () => {
        const expectedExpiration = 'in 29 days on 09/29/2020'; // Degraded

        it('should enable the upgrade option', () => {
            const clusterName = 'eta-7';
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
            const clusterName = 'eta-7';
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

                cy.intercept('GET', clustersApi.single, {
                    body: { cluster },
                }).as('getCluster');
                cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${n + 1})`).click();
                cy.wait('@getCluster');

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
    });
});

// TODO: re-enable and update these tests when we migrate Clusters section to PatternFly
describe.skip('Cluster Creation Flow', () => {
    withAuth();

    it('Should be able to fill out the Kubernetes form, download config files and see cluster checked-in', () => {
        visitClusters();

        cy.get(selectors.buttons.new).click();

        const clusterName = 'Kubernetes Cluster TestInstance';
        cy.get(selectors.clusterForm.nameInput).type(clusterName);

        cy.intercept('POST', clustersApi.list).as('postCluster');
        cy.get(selectors.buttons.next).click();
        // TypeError: Cannot read properties of undefined (reading 'overallHealthStatus')
        cy.wait('@postCluster').then(({ response }) => {
            const clusterId = response.cluster.id;

            // Confirm whether fixture is needed.
            /*
            // mocking a ZIP file download
            //   based on: https://github.com/cypress-io/cypress/issues/1956#issuecomment-455157737
            cy.fixture('clusters/sensor-kubernetes-cluster-testinstance.zip').then((dataURI) => {
                return cy
                    .intercept('POST', clustersApi.zip, {
                        headers: {
                            'content-disposition':
                                'attachment; filename="sensor-kubernetes-cluster-testinstance.zip"',
                        },
                        body: Cypress.Blob.base64StringToBlob(dataURI, 'image/jpeg'),
                    })
                    .as('download');
            });
            */
            cy.intercept('POST', clustersApi.zip).as('download');
            cy.get(selectors.buttons.downloadYAML).click();
            cy.wait('@download');

            cy.get('div:contains("Waiting for the cluster to check in successfully...")');

            // make cluster to "check-in" by adding "lastContact"
            cy.intercept('GET', `${clustersApi.list}/${clusterId}`, {
                body: {
                    cluster: {
                        id: clusterId,
                        healthStatus: {
                            lastContact: '2018-06-25T19:12:44.955289Z',
                        },
                    },
                },
            }).as('getCluster');
            cy.wait('@getCluster');

            cy.get('div:contains("Waiting for the cluster to check in successfully...")').should(
                'not.exist'
            );

            cy.intercept('GET', clustersApi.list).as('getClusters');
            cy.get(selectors.buttons.closePanel).click();
            cy.wait('@getClusters');

            // clean up after the test by deleting the cluster
            cy.intercept('DELETE', clustersApi.list).as('deleteCluster');
            cy.get(`.rt-tr:contains("${clusterName}") .rt-td input[type="checkbox"]`).check();
            cy.get(selectors.buttons.delete).click();
            cy.get(selectors.buttons.confirmDelete).click();
            cy.wait(['@deleteCluster', '@getClusters']);

            cy.get(`.rt-tr:contains("${clusterName}")`).should('not.exist');
        });
    });
});

describe('Cluster management', () => {
    withAuth();

    it('should indicate which clusters are managed by Helm and the Operator', () => {
        const fixturePath = 'clusters/health.json';
        visitClusters({
            clusters: { fixture: fixturePath },
        });

        const helmIndicator = '[data-testid="cluster-name"] img[alt="Managed by Helm"]';
        const k8sOperatorIndicator =
            '[data-testid="cluster-name"] img[alt="Managed by a Kubernetes Operator"]';
        const anyIndicator = '[data-testid="cluster-name"] img';
        cy.get(`${selectors.clusters.tableRowGroup}:eq(0) ${helmIndicator}`).should('exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(1) ${anyIndicator}`).should('not.exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(2) ${k8sOperatorIndicator}`).should('exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(3) ${helmIndicator}`).should('exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(4) ${anyIndicator}`).should('not.exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(5) ${anyIndicator}`).should('not.exist');
        cy.get(`${selectors.clusters.tableRowGroup}:eq(6) ${anyIndicator}`).should('not.exist');
    });
});

describe('Cluster configuration', () => {
    withAuth();

    const fixturePath = 'clusters/health.json';

    const assertConfigurationReadOnly = () => {
        const form = cy.get('[data-testid="cluster-form"]').children();
        [
            'name',
            'mainImage',
            'centralApiEndpoint',
            'collectorImage',
            'admissionControllerEvents',
            'admissionController',
            'admissionControllerUpdates',
            'tolerationsConfig.disabled',
            'slimCollector',
            'dynamicConfig.registryOverride',
            'dynamicConfig.admissionControllerConfig.enabled',
            'dynamicConfig.admissionControllerConfig.enforceOnUpdates',
            'dynamicConfig.admissionControllerConfig.timeoutSeconds',
            'dynamicConfig.admissionControllerConfig.scanInline',
            'dynamicConfig.admissionControllerConfig.disableBypass',
            'dynamicConfig.disableAuditLogs',
        ].forEach((id) => form.get(`input[id="${id}"]`).should('be.disabled'));
        ['Select a cluster type', 'Select a runtime option'].forEach((label) =>
            form.get(`select[aria-label="${label}"]`).should('be.disabled')
        );
    };

    it('should be read-only for Helm-based installations', () => {
        visitClusterByNameWithFixture('alpha-amsterdam-1', fixturePath);
        assertConfigurationReadOnly();
    });

    it('should be read-only for unknown manager installations that have a defined Helm config', () => {
        visitClusterByNameWithFixture('kappa-kilogramme-10', fixturePath);
        assertConfigurationReadOnly();
    });
});

describe.skip('Cluster Health', () => {
    withAuth();

    const fixturePath = 'clusters/health.json';
    const metadata = {
        version: '3.0.50.0', // for comparison to `sensorVersion` in clusters fixture
        buildFlavor: 'release',
        releaseBuild: true,
        licenseStatus: 'VALID',
    };
    const datetimeISOString = '2020-08-31T13:01:00Z'; // for comparison to `lastContact` and `sensorCertExpiry` in clusters fixture

    const expectedClusters = [
        {
            expectedInListAndSide: {
                clusterName: 'alpha-amsterdam-1',
                cloudProvider: 'Not applicable',
                clusterStatus: 'Uninitialized',
                sensorUpgrade: 'Not applicable',
                credentialExpiration: 'Not applicable',
            },
            expectedInSide: {
                admissionControlHealthInfo: null,
                collectorHealthInfo: null,
                healthInfoComplete: null,
                sensorVersion: null,
                centralVersion: null,
                sensorStatus: 'Uninitialized',
                collectorStatus: 'Uninitialized',
                admissionControlStatus: 'Uninitialized',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'epsilon-edison-5',
                cloudProvider: 'AWS us-west1',
                clusterStatus: 'Unhealthy',
                sensorUpgrade: 'Upgrade available',
                credentialExpiration: 'in 6 days on Monday',
            },
            expectedInSide: {
                admissionControlHealthInfo: {
                    totalReadyPods: '3',
                    totalDesiredPods: '3',
                },
                collectorHealthInfo: {
                    totalReadyPods: '10',
                    totalDesiredPods: '10',
                    totalRegisteredNodes: '12',
                },
                healthInfoComplete: null,
                sensorVersion: '3.0.48.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Unhealthy for 1 hour',
                collectorStatus: 'Healthy 1 hour ago',
                admissionControlStatus: 'Healthy 1 hour ago',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'eta-7',
                cloudProvider: 'GCP us-west1',
                clusterStatus: 'Unhealthy',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 29 days on 09/29/2020',
            },
            expectedInSide: {
                admissionControlHealthInfo: {
                    totalReadyPods: '1',
                    totalDesiredPods: '3',
                },
                collectorHealthInfo: {
                    totalReadyPods: '3',
                    totalDesiredPods: '5',
                    totalRegisteredNodes: '6',
                },
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Healthy',
                collectorStatus: 'Unhealthy',
                admissionControlStatus: 'Unhealthy',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'kappa-kilogramme-10',
                cloudProvider: 'AWS us-central1',
                clusterStatus: 'Degraded',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 1 month',
            },
            expectedInSide: {
                admissionControlHealthInfo: {
                    totalReadyPods: '3',
                    totalDesiredPods: '3',
                },
                collectorHealthInfo: {
                    totalReadyPods: '10',
                    totalDesiredPods: '10',
                    totalRegisteredNodes: '12',
                },
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Degraded for 2 minutes',
                collectorStatus: 'Healthy 2 minutes ago',
                admissionControlStatus: 'Healthy 2 minutes ago',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'lambda-liverpool-11',
                cloudProvider: 'GCP us-central1',
                clusterStatus: 'Degraded',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 2 months',
            },
            expectedInSide: {
                admissionControlHealthInfo: {
                    totalReadyPods: '3',
                    totalDesiredPods: '3',
                },
                collectorHealthInfo: {
                    totalReadyPods: '8',
                    totalDesiredPods: '10',
                    totalRegisteredNodes: '12',
                },
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Healthy',
                collectorStatus: 'Degraded',
                admissionControlStatus: 'Healthy',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'mu-madegascar-12',
                cloudProvider: 'AWS eu-central1',
                clusterStatus: 'Healthy',
                sensorUpgrade: 'Upgrade available',
                credentialExpiration: 'in 12 months',
            },
            expectedInSide: {
                admissionControlHealthInfo: null,
                collectorHealthInfo: null,
                healthInfoComplete: {
                    admissionControl: 'Upgrade Sensor to get Admission Control health information',
                    collector: 'Upgrade Sensor to get Collector health information',
                },
                sensorVersion: '3.0.47.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Healthy',
                collectorStatus: 'Unavailable',
                admissionControlStatus: 'Unavailable',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'nu-york-13',
                cloudProvider: 'AWS ap-southeast1',
                clusterStatus: 'Healthy',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 1 year',
            },
            expectedInSide: {
                admissionControlHealthInfo: {
                    totalReadyPods: '3',
                    totalDesiredPods: '3',
                },
                collectorHealthInfo: {
                    totalReadyPods: '7',
                    totalDesiredPods: '7',
                    totalRegisteredNodes: '7',
                },
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
                sensorStatus: 'Healthy',
                collectorStatus: 'Healthy',
                admissionControlStatus: 'Healthy',
            },
        },
    ];

    it('should appear in the list', () => {
        visitClustersWithFixtureMetadataDatetime(fixturePath, metadata, datetimeISOString);

        /*
         * Some cells have no internal markup (for example, Name or Cloud Provider).
         * Other cells have div and spans for status color versus default color.
         */
        // TODO add assertion for Cluster Deletion column after ROX_DECOMMISSIONED_CLUSTER_RETENTION feature flag is deleted.
        cy.get(selectors.clusters.tableDataCell).should(($tds) => {
            let n = 0;
            expectedClusters.forEach(({ expectedInListAndSide }) => {
                Object.keys(expectedInListAndSide).forEach((key) => {
                    if (key === 'clusterStatus') {
                        expect($tds.eq(n).text()).to.include(expectedInListAndSide[key]);
                    } else {
                        expect($tds.eq(n).text()).to.equal(expectedInListAndSide[key]);
                    }
                    n += 1;
                });
            });
            expect($tds.length).to.equal(n);
        });
    });

    expectedClusters.forEach(({ expectedInListAndSide, expectedInSide }) => {
        const { clusterName, clusterStatus, sensorUpgrade, credentialExpiration } =
            expectedInListAndSide;
        const {
            admissionControlHealthInfo,
            collectorHealthInfo,
            healthInfoComplete,
            sensorVersion,
            centralVersion,
            sensorStatus,
            collectorStatus,
            admissionControlStatus,
        } = expectedInSide;

        it(
            `should appear in the form for ${clusterName}`,
            // TODO(ROX-9546): Debug why we have network error here and remove retries
            {
                retries: {
                    runMode: 1,
                    openMode: 0,
                },
            },
            () => {
                visitClusterByNameWithFixtureMetadataDatetime(
                    clusterName,
                    fixturePath,
                    metadata,
                    datetimeISOString
                );

                cy.get(selectors.clusterForm.nameInput).should('have.value', clusterName);

                // Cluster Status
                cy.get(selectors.clusterHealth.clusterStatus).should('have.text', clusterStatus);

                // Sensor Status
                cy.get(selectors.clusterHealth.sensorStatus).should('have.text', sensorStatus);

                // Collector Status
                cy.get(selectors.clusterHealth.collectorStatus).should(
                    'have.text',
                    collectorStatus
                );
                if (collectorHealthInfo !== null) {
                    const { totalReadyPods, totalDesiredPods, totalRegisteredNodes } =
                        collectorHealthInfo;
                    cy.get(selectors.clusterHealth.collectorHealthInfo.totalReadyPods).should(
                        'have.text',
                        totalReadyPods
                    );
                    cy.get(selectors.clusterHealth.collectorHealthInfo.totalDesiredPods).should(
                        'have.text',
                        totalDesiredPods
                    );
                    cy.get(selectors.clusterHealth.collectorHealthInfo.totalRegisteredNodes).should(
                        'have.text',
                        totalRegisteredNodes
                    );
                }
                // Admission Control Status
                cy.get(selectors.clusterHealth.admissionControlStatus).should(
                    'have.text',
                    admissionControlStatus
                );
                if (admissionControlHealthInfo !== null) {
                    const { totalReadyPods, totalDesiredPods } = admissionControlHealthInfo;
                    cy.get(
                        selectors.clusterHealth.admissionControlHealthInfo.totalReadyPods
                    ).should('have.text', totalReadyPods);
                    cy.get(
                        selectors.clusterHealth.admissionControlHealthInfo.totalDesiredPods
                    ).should('have.text', totalDesiredPods);
                }
                if (healthInfoComplete !== null) {
                    cy.get(selectors.clusterHealth.admissionControlInfoComplete).should(
                        'have.text',
                        healthInfoComplete.admissionControl
                    );
                    cy.get(selectors.clusterHealth.collectorInfoComplete).should(
                        'have.text',
                        healthInfoComplete.collector
                    );
                }

                // Sensor Upgrade
                cy.get(selectors.clusterHealth.sensorUpgrade).should('have.text', sensorUpgrade);
                if (typeof sensorVersion === 'string') {
                    cy.get(selectors.clusterHealth.sensorVersion).should(
                        'have.text',
                        sensorVersion
                    );
                }
                if (typeof centralVersion === 'string') {
                    cy.get(selectors.clusterHealth.centralVersion).should(
                        'have.text',
                        centralVersion
                    );
                }

                // Credential Expiration
                cy.get(selectors.clusterHealth.credentialExpiration).should(
                    'have.text',
                    credentialExpiration
                );
            }
        );
    });
});
