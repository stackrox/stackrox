import dateFns from 'date-fns';
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

            const expectedHeadings = checkFeatureFlag('ROX_CLUSTER_HEALTH_MONITORING', true)
                ? [
                      'Name',
                      'Cloud Provider',
                      'Cluster Status',
                      'Sensor Status',
                      'Collector Status',
                      'Sensor Upgrade',
                      'Credential Expiration',
                  ]
                : [
                      'Name',
                      'Orchestrator',
                      'Runtime Collection',
                      'Admission Controller',
                      'Cloud Provider',
                      'Last check-in',
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

describe('Cluster Cert Expiration', () => {
    withAuth();

    // Make a request to the clusters API, and modify it to have the changes we need.
    // Do it this way to avoid having to deal with the overhead of maintaining full-blown fixtures.
    const getModifiedMockCluster = (
        cluster,
        expiry,
        upgradeStatusOutOfDate,
        recentCertRotationUpgradeTime
    ) => {
        const modifiedCluster = { ...cluster };
        modifiedCluster.status.certExpiryStatus = { sensorCertExpiry: expiry };
        if (upgradeStatusOutOfDate) {
            modifiedCluster.status.upgradeStatus.upgradability = 'AUTO_UPGRADE_POSSIBLE';
        } else {
            modifiedCluster.status.upgradeStatus.upgradability = 'UP_TO_DATE';
        }
        if (recentCertRotationUpgradeTime) {
            modifiedCluster.status.upgradeStatus.mostRecentProcess = {
                type: 'CERT_ROTATION',
                initiatedAt: recentCertRotationUpgradeTime,
                progress: {
                    upgradeState: 'UPGRADE_COMPLETE',
                },
            };
        } else {
            modifiedCluster.status.upgradeStatus.mostRecentProcess = null;
        }
        return modifiedCluster;
    };

    const openSidePanelWithMockedClusters = (mockCluster) => {
        cy.server();
        cy.route('GET', clustersApi.single, { cluster: mockCluster }).as('cluster');

        cy.visit(clustersUrl);
        cy.get(`${selectors.clusters.tableRowGroup}:first-child`).click();
        cy.wait('@cluster');

        cy.get(selectors.sidePanel);
    };

    it('should not show warning if expiration is more than 30 days away', () => {
        const mockExpiry = dateFns.addDays(new Date(), 31);
        cy.fixture('clusters/single-cluster-with-status.json').then((resp) => {
            const certCluster = getModifiedMockCluster(resp.cluster, mockExpiry);

            openSidePanelWithMockedClusters(certCluster);

            cy.get(selectors.credentialExpirationBanner).should('not.exist');
        });
    });

    describe('should show warning if expiration is less than 30 days away', () => {
        const verifyBannerTextEquals = (expectedText) => {
            cy.get(selectors.credentialExpirationBanner)
                .invoke('text')
                .then((text) => {
                    expect(text).to.equal(expectedText);
                });
        };

        const mockExpiry = dateFns.addDays(new Date(), 29);

        it('should not show auto-upgrade link if sensor is not up-to-date', () => {
            cy.fixture('clusters/single-cluster-with-status.json').then((resp) => {
                const outdatedCluster = getModifiedMockCluster(resp.cluster, mockExpiry, true);

                openSidePanelWithMockedClusters(outdatedCluster);

                verifyBannerTextEquals(
                    'This cluster’s credentials expire in 28 days. To use renewed certificates, download this YAML file and apply it to your cluster.'
                );
            });
        });

        it('should show auto-upgrade link if sensor is up-to-date', () => {
            cy.fixture('clusters/single-cluster-with-status.json').then((resp) => {
                const outdatedCluster = getModifiedMockCluster(resp.cluster, mockExpiry);

                openSidePanelWithMockedClusters(outdatedCluster);

                verifyBannerTextEquals(
                    'This cluster’s credentials expire in 28 days. To use renewed certificates, download this YAML file and apply it to your cluster, or apply credentials by using an automatic upgrade.'
                );
            });
        });

        it('should show auto-upgrade link, and banner with time of recent upgrade', () => {
            const mockCertRotationTime = dateFns.addMinutes(new Date(), -5);

            cy.fixture('clusters/single-cluster-with-status.json').then((resp) => {
                const outdatedCluster = getModifiedMockCluster(
                    resp.cluster,
                    mockExpiry,
                    false,
                    mockCertRotationTime
                );

                openSidePanelWithMockedClusters(outdatedCluster);

                cy.get(selectors.credentialExpirationBanner).should(
                    'contain',
                    'This cluster’s credentials expire in 28 days. To use renewed certificates, download this YAML file and apply it to your cluster, or apply credentials by using an automatic upgrade.'
                );
                cy.get(selectors.credentialExpirationBanner).should(
                    'contain',
                    'An automatic upgrade applied renewed credentials on '
                );
            });
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
            return Cypress.Blob.base64StringToBlob(dataURI, 'image/jpeg').then((blob) => {
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
                        status: {
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

describe('Cluster Health', () => {
    withAuth();

    let clustersFixture;

    before(function beforeHook() {
        // skip the test if feature flag is not enabled
        if (checkFeatureFlag('ROX_CLUSTER_HEALTH_MONITORING', false)) {
            this.skip();
        }

        cy.fixture('clusters/health.json').then((clustersArg) => {
            clustersFixture = clustersArg;
        });
    });

    // For comparison to `lastContact` and `sensorCertExpiry` in clusters fixture.
    const currentDatetime = new Date('2020-08-31T13:01:00Z');

    // For comparison to `sensorVersion` in clusters fixture.
    const version = '3.0.50.0';

    beforeEach(() => {
        cy.server();
        cy.route('GET', clustersApi.list, clustersFixture).as('GetClusters');
        cy.route('GET', metadataApi, {
            version,
            buildFlavor: 'release',
            releaseBuild: true,
            licenseStatus: 'VALID',
        }).as('GetMetadata');

        cy.clock(currentDatetime.getTime(), ['Date', 'setInterval']);

        cy.visit(clustersUrl);
        cy.wait(['@GetClusters', '@GetMetadata']);
    });

    const expectedClusters = [
        {
            clusterName: 'alpha-amsterdam-1',
            cloudProvider: 'Not applicable',
            clusterStatus: 'Uninitialized',
            sensorStatus: 'Uninitialized',
            collectorStatus: 'Uninitialized',
            sensorUpgrade: 'Not applicable',
            credentialExpiration: 'Not applicable',
        },
        {
            clusterName: 'epsilon-edison-5',
            cloudProvider: 'AWS us-west1',
            clusterStatus: 'Unhealthy',
            sensorStatus: 'Unhealthy for 1 hour',
            collectorStatus: 'Healthy 1 hour ago',
            sensorUpgrade: 'Up to date with Central',
            credentialExpiration: 'in 6 days on Monday',
        },
        {
            clusterName: 'eta-7',
            cloudProvider: 'GCP us-west1',
            clusterStatus: 'Unhealthy',
            sensorStatus: 'Healthy',
            collectorStatus: 'Unhealthy',
            sensorUpgrade: 'Up to date with Central',
            credentialExpiration: 'in 29 days on 09/29/2020',
        },
        {
            clusterName: 'kappa-kilogramme-10',
            cloudProvider: 'AWS us-central1',
            clusterStatus: 'Degraded',
            sensorStatus: 'Degraded for 2 minutes',
            collectorStatus: 'Healthy 2 minutes ago',
            sensorUpgrade: 'Up to date with Central',
            credentialExpiration: 'in 1 month',
        },
        {
            clusterName: 'lambda-liverpool-11',
            cloudProvider: 'GCP us-central1',
            clusterStatus: 'Degraded',
            sensorStatus: 'Healthy',
            collectorStatus: 'Degraded',
            sensorUpgrade: 'Upgrade available',
            credentialExpiration: 'in 2 months',
        },
        {
            clusterName: 'mu-madegascar-12',
            cloudProvider: 'AWS eu-central1',
            clusterStatus: 'Healthy',
            sensorStatus: 'Healthy',
            collectorStatus: 'Unavailable',
            sensorUpgrade: 'Upgrade available',
            credentialExpiration: 'in 12 months',
        },
        {
            clusterName: 'nu-york-13',
            cloudProvider: 'AWS ap-southeast1',
            clusterStatus: 'Healthy',
            sensorStatus: 'Healthy',
            collectorStatus: 'Healthy',
            sensorUpgrade: 'Up to date with Central',
            credentialExpiration: 'in 1 year',
        },
    ];

    it('should appear in the list', () => {
        /*
         * Some cells have no internal markup (for example, Name or Cloud Provider).
         * Other cells have div and spans for status color versus default color.
         */
        cy.get(selectors.clusters.tableDataCell).should(($tds) => {
            let n = 0;
            expectedClusters.forEach((expectedCluster) => {
                Object.keys(expectedCluster).forEach((key) => {
                    expect($tds.eq(n).text()).to.equal(expectedCluster[key]);
                    n += 1;
                });
            });
            expect($tds.length).to.equal(n);
        });
    });

    expectedClusters.forEach((expectedCluster, i) => {
        const {
            clusterName,
            clusterStatus,
            sensorStatus,
            collectorStatus,
            sensorUpgrade,
            credentialExpiration,
        } = expectedCluster;

        it(`should appear in the form for ${clusterName}`, () => {
            const cluster = clustersFixture.clusters[i];
            cy.route('GET', clustersApi.single, { cluster }).as('GetCluster');
            cy.get(`${selectors.clusters.tableRowGroup}:nth-child(${i + 1})`).click();
            cy.wait('@GetCluster');

            cy.get(selectors.clusterForm.nameInput).should('have.value', clusterName);
            cy.get(selectors.clusterHealth.clusterStatus).should('have.text', clusterStatus);
            cy.get(selectors.clusterHealth.sensorStatus).should('have.text', sensorStatus);
            cy.get(selectors.clusterHealth.collectorStatus).should('have.text', collectorStatus);
            cy.get(selectors.clusterHealth.sensorUpgrade).should('have.text', sensorUpgrade);
            cy.get(selectors.clusterHealth.credentialExpiration).should(
                'have.text',
                credentialExpiration
            );
        });
    });
});
