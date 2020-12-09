import cloneDeep from 'lodash/cloneDeep';

import { selectors, clustersUrl } from '../constants/ClustersPage';
import { clusters as clustersApi, metadata as metadataApi } from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import checkFeatureFlag from '../helpers/features';

describe('Clusters page', () => {
    withAuth();

    describe('smoke tests', () => {
        beforeEach(() => {
            cy.visit('/');
            cy.get(selectors.configure).click();
            cy.get(selectors.navLink).click({ force: true });
        });

        it('should be linked in the Platform Configuration menu', () => {
            cy.get(selectors.header).contains('Clusters');
        });

        it('should have a toggle control for the auto-upgrade setting', () => {
            cy.get(selectors.autoUpgradeInput);
        });

        it('should display all the columns expected in clusters list page', () => {
            cy.visit(clustersUrl);

            const expectedHeadings = [
                'Name',
                'Cloud Provider',
                'Cluster Status',
                'Sensor Status',
                'Collector Status',
                'Sensor Upgrade',
                'Credential Expiration',
            ];

            cy.get(selectors.clusters.tableHeadingCell).should(($ths) => {
                let n = 0;
                expectedHeadings.forEach((expectedHeading) => {
                    expect($ths.eq(n).text()).to.equal(expectedHeading);
                    n += 1;
                });
                expect($ths.length).to.equal(n);
            });
        });
    });
});

describe('Cluster Certificate Expiration', () => {
    withAuth();

    let clustersFixture;

    before(function beforeHook() {
        cy.fixture('clusters/health.json').then((clustersArg) => {
            clustersFixture = clustersArg;
        });
    });

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    beforeEach(() => {
        cy.server();
        cy.route('GET', clustersApi.list, clustersFixture).as('GetClusters');
        cy.route('GET', metadataApi, {
            version: '3.0.50.0', // for comparison to `sensorVersion` in clusters fixture
            buildFlavor: 'release',
            releaseBuild: true,
            licenseStatus: 'VALID',
        }).as('GetMetadata');

        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.visit(clustersUrl);
        cy.wait(['@GetClusters', '@GetMetadata']);
    });

    describe('Credential Expiration status is Healthy', () => {
        it('should not show link or form', () => {
            const { clusters } = clustersFixture;
            const n = clusters.findIndex((cluster) => cluster.name === 'kappa-kilogramme-10');
            const cluster = clusters[n];

            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${n + 1})`).click();
            cy.wait('@GetCluster');

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
            const { clusters } = clustersFixture;
            const n = clusters.findIndex((cluster) => cluster.name === 'epsilon-edison-5');
            const cluster = clusters[n];

            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${n + 1})`).click();
            cy.wait('@GetCluster');

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
            const { clusters } = clustersFixture;
            const n = clusters.findIndex((cluster) => cluster.name === 'eta-7');
            const cluster = clusters[n];

            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${n + 1})`).click();
            cy.wait('@GetCluster');

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
            const { clusters } = clustersFixture;
            const n = clusters.findIndex((cluster) => cluster.name === 'eta-7');
            const cluster = cloneDeep(clusters[n]);

            // Mock the result of using an automatic upgrade to re-issue the certificate.
            cluster.status.upgradeStatus.mostRecentProcess = {
                type: 'CERT_ROTATION',
                initiatedAt: currentDatetime,
                progress: {
                    upgradeState: 'UPGRADE_COMPLETE',
                },
            };

            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${n + 1})`).click();
            cy.wait('@GetCluster');

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

describe('Cluster Creation Flow', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('clusters/single.json').as('singleCluster');
        cy.fixture('clusters/new.json').as('newCluster');

        // mocking a ZIP file download
        //   based on: https://github.com/cypress-io/cypress/issues/1956#issuecomment-455157737
        cy.fixture('clusters/sensor-kubernetes-cluster-testinstance.zip').then((dataURI) => {
            const blob = Cypress.Blob.base64StringToBlob(dataURI, 'image/jpeg');
            return cy
                .route({
                    url: clustersApi.zip,
                    method: 'POST',
                    response: '',
                    onResponse: (xhr) => {
                        xhr.response.body = blob; // eslint-disable-line no-param-reassign
                    },
                    headers: {
                        'content-disposition':
                            'attachment; filename="sensor-kubernetes-cluster-testinstance.zip"',
                    },
                })
                .as('download');
        });

        cy.route('GET', clustersApi.list, '@singleCluster').as('clusters');
        cy.route('POST', clustersApi.list, '@newCluster').as('addCluster');
        cy.visit(clustersUrl);
        cy.wait('@clusters');
    });

    xit('Should show a confirmation dialog when trying to delete clusters', () => {
        cy.get(selectors.dialog).should('not.exist');
        cy.get(selectors.checkboxes).check();
        cy.get(selectors.buttons.delete).click({ force: true });
        cy.get(selectors.dialog);
        cy.get(selectors.buttons.cancelDelete).click({ force: true });
    });

    xit('Should be able to fill out the Kubernetes form, download config files and see cluster checked-in', () => {
        cy.get(selectors.buttons.new).click();

        const clusterName = 'Kubernetes Cluster TestInstance';
        cy.get(selectors.clusterForm.nameInput).type(clusterName);
        // The image name should be pre-populated, so we don't type it in to test that the prepopulation works.
        // (The backend WILL error out if the image is empty.)
        cy.get(selectors.clusterForm.endpointInput).clear().type('central.stackrox:443');

        cy.get(selectors.buttons.next).click();
        cy.wait('@addCluster')
            .its('responseBody')
            .then((response) => {
                const clusterId = response.cluster.id;

                cy.get(selectors.buttons.downloadYAML).click();
                cy.wait('@download');

                cy.get('div:contains("Waiting for the cluster to check in successfully...")');

                // make cluster to "check-in" by adding "lastContact"
                cy.route('GET', `${clustersApi.list}/${clusterId}`, {
                    cluster: {
                        id: clusterId,
                        healthStatus: {
                            lastContact: '2018-06-25T19:12:44.955289Z',
                        },
                    },
                }).as('getCluster');
                cy.wait('@getCluster');
                cy.get(
                    'div:contains("Waiting for the cluster to check in successfully...")'
                ).should('not.exist');

                cy.route('GET', clustersApi.list, 'fixture:clusters/couple.json').as('clusters');

                cy.get(selectors.buttons.closePanel).click();

                // clean up after the test by deleting the cluster
                cy.route('DELETE', clustersApi.list, {});
                cy.get(`.rt-tr:contains("${clusterName}") .rt-td input[type="checkbox"]`).check();
                cy.get(selectors.buttons.delete).click();
                cy.get(selectors.buttons.confirmDelete).click();
                cy.route('GET', clustersApi.list, '@singleCluster').as('clusters');
                cy.get(`.rt-tr:contains("${clusterName}")`).should('not.exist');
            });
    });
});

describe('Cluster with Helm management', () => {
    before(function beforeHook() {
        if (checkFeatureFlag('ROX_SENSOR_INSTALLATION_EXPERIENCE', false)) {
            this.skip();
        }
    });

    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('clusters/health.json').as('clusters');
        cy.route('GET', clustersApi.list, '@clusters').as('GetClusters');

        cy.visit(clustersUrl);
        cy.wait(['@GetClusters']);
    });

    it('should indicate which clusters are managed by Helm', () => {
        cy.get(`${selectors.clusters.tableDataCell} [data-testid="cluster-name"]`).each(
            ($nameCell, index) => {
                if (index === 0) {
                    cy.wrap($nameCell.children()).get('img[alt="Managed by Helm"]');
                } else {
                    cy.wrap($nameCell.children()).should('have.length', 0);
                }
            }
        );
    });
});

describe('Cluster Health', () => {
    withAuth();

    let clustersFixture;

    before(function beforeHook() {
        cy.fixture('clusters/health.json').then((clustersArg) => {
            clustersFixture = clustersArg;
        });
    });

    beforeEach(() => {
        cy.server();
        cy.route('GET', clustersApi.list, clustersFixture).as('GetClusters');
        cy.route('GET', metadataApi, {
            version: '3.0.50.0', // for comparison to `sensorVersion` in clusters fixture
            buildFlavor: 'release',
            releaseBuild: true,
            licenseStatus: 'VALID',
        }).as('GetMetadata');

        // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
        const currentDatetime = new Date('2020-08-31T13:01:00Z');
        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.visit(clustersUrl);
        cy.wait(['@GetClusters', '@GetMetadata']);
    });

    const expectedClusters = [
        {
            expectedInListAndSide: {
                clusterName: 'alpha-amsterdam-1',
                cloudProvider: 'Not applicable',
                clusterStatus: 'Uninitialized',
                sensorStatus: 'Uninitialized',
                collectorStatus: 'Uninitialized',
                sensorUpgrade: 'Not applicable',
                credentialExpiration: 'Not applicable',
            },
            expectedInSide: {
                totalReadyPods: null,
                totalDesiredPods: null,
                totalRegisteredNodes: null,
                healthInfoComplete: null,
                sensorVersion: null,
                centralVersion: null,
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'epsilon-edison-5',
                cloudProvider: 'AWS us-west1',
                clusterStatus: 'Unhealthy',
                sensorStatus: 'Unhealthy for 1 hour',
                collectorStatus: 'Healthy 1 hour ago',
                sensorUpgrade: 'Upgrade available',
                credentialExpiration: 'in 6 days on Monday',
            },
            expectedInSide: {
                totalReadyPods: '10',
                totalDesiredPods: '10',
                totalRegisteredNodes: '12',
                healthInfoComplete: null,
                sensorVersion: '3.0.48.0',
                centralVersion: '3.0.50.0',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'eta-7',
                cloudProvider: 'GCP us-west1',
                clusterStatus: 'Unhealthy',
                sensorStatus: 'Healthy',
                collectorStatus: 'Unhealthy',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 29 days on 09/29/2020',
            },
            expectedInSide: {
                totalReadyPods: '3',
                totalDesiredPods: '5',
                totalRegisteredNodes: '6',
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'kappa-kilogramme-10',
                cloudProvider: 'AWS us-central1',
                clusterStatus: 'Degraded',
                sensorStatus: 'Degraded for 2 minutes',
                collectorStatus: 'Healthy 2 minutes ago',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 1 month',
            },
            expectedInSide: {
                totalReadyPods: '10',
                totalDesiredPods: '10',
                totalRegisteredNodes: '12',
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'lambda-liverpool-11',
                cloudProvider: 'GCP us-central1',
                clusterStatus: 'Degraded',
                sensorStatus: 'Healthy',
                collectorStatus: 'Degraded',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 2 months',
            },
            expectedInSide: {
                totalReadyPods: '8',
                totalDesiredPods: '10',
                totalRegisteredNodes: '12',
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'mu-madegascar-12',
                cloudProvider: 'AWS eu-central1',
                clusterStatus: 'Healthy',
                sensorStatus: 'Healthy',
                collectorStatus: 'Unavailable',
                sensorUpgrade: 'Upgrade available',
                credentialExpiration: 'in 12 months',
            },
            expectedInSide: {
                totalReadyPods: null,
                totalDesiredPods: null,
                totalRegisteredNodes: null,
                healthInfoComplete: 'Upgrade Sensor to get Collector health information',
                sensorVersion: '3.0.47.0',
                centralVersion: '3.0.50.0',
            },
        },
        {
            expectedInListAndSide: {
                clusterName: 'nu-york-13',
                cloudProvider: 'AWS ap-southeast1',
                clusterStatus: 'Healthy',
                sensorStatus: 'Healthy',
                collectorStatus: 'Healthy',
                sensorUpgrade: 'Up to date with Central',
                credentialExpiration: 'in 1 year',
            },
            expectedInSide: {
                totalReadyPods: '7',
                totalDesiredPods: '7',
                totalRegisteredNodes: '7',
                healthInfoComplete: null,
                sensorVersion: '3.0.50.0',
                centralVersion: '3.0.50.0',
            },
        },
    ];

    it('should appear in the list', () => {
        /*
         * Some cells have no internal markup (for example, Name or Cloud Provider).
         * Other cells have div and spans for status color versus default color.
         */
        cy.get(selectors.clusters.tableDataCell).should(($tds) => {
            let n = 0;
            expectedClusters.forEach(({ expectedInListAndSide }) => {
                Object.keys(expectedInListAndSide).forEach((key) => {
                    expect($tds.eq(n).text()).to.equal(expectedInListAndSide[key]);
                    n += 1;
                });
            });
            expect($tds.length).to.equal(n);
        });
    });

    expectedClusters.forEach(({ expectedInListAndSide, expectedInSide }, i) => {
        const {
            clusterName,
            clusterStatus,
            sensorStatus,
            collectorStatus,
            sensorUpgrade,
            credentialExpiration,
        } = expectedInListAndSide;
        const {
            totalReadyPods,
            totalDesiredPods,
            totalRegisteredNodes,
            healthInfoComplete,
            sensorVersion,
            centralVersion,
        } = expectedInSide;

        it(`should appear in the form for ${clusterName}`, () => {
            const cluster = clustersFixture.clusters[i];
            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${i + 1})`).click();
            cy.wait('@GetCluster');

            cy.get(selectors.clusterForm.nameInput).should('have.value', clusterName);

            // Cluster Status
            cy.get(selectors.clusterHealth.clusterStatus).should('have.text', clusterStatus);

            // Sensor Status
            cy.get(selectors.clusterHealth.sensorStatus).should('have.text', sensorStatus);

            // Collector Status
            cy.get(selectors.clusterHealth.collectorStatus).should('have.text', collectorStatus);
            if (typeof totalReadyPods === 'string') {
                cy.get(selectors.clusterHealth.totalReadyPods).should('have.text', totalReadyPods);
            }
            if (typeof totalDesiredPods === 'string') {
                cy.get(selectors.clusterHealth.totalDesiredPods).should(
                    'have.text',
                    totalDesiredPods
                );
            }
            if (typeof totalRegisteredNodes === 'string') {
                cy.get(selectors.clusterHealth.totalRegisteredNodes).should(
                    'have.text',
                    totalRegisteredNodes
                );
            }
            if (typeof healthInfoComplete === 'string') {
                cy.get(selectors.clusterHealth.healthInfoComplete).should(
                    'have.text',
                    healthInfoComplete
                );
            }

            // Sensor Upgrade
            cy.get(selectors.clusterHealth.sensorUpgrade).should('have.text', sensorUpgrade);
            if (typeof sensorVersion === 'string') {
                cy.get(selectors.clusterHealth.sensorVersion).should('have.text', sensorVersion);
            }
            if (typeof centralVersion === 'string') {
                cy.get(selectors.clusterHealth.centralVersion).should('have.text', centralVersion);
            }

            // Credential Expiration
            cy.get(selectors.clusterHealth.credentialExpiration).should(
                'have.text',
                credentialExpiration
            );
        });
    });
});
