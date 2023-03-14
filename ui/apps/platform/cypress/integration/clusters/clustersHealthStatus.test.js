import withAuth from '../../helpers/basicAuth';

import {
    visitClusterByNameWithFixtureMetadataDatetime,
    visitClustersWithFixtureMetadataDatetime,
} from './Clusters.helpers';
import { selectors } from './Clusters.selectors';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Clusters Health Status', () => {
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
                cloudProvider: 'Not available',
                clusterStatus: 'Uninitialized',
                sensorUpgrade: 'Not applicable',
                credentialExpiration: 'Not applicable',
                clusterDeletion: 'Not applicable',
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
                clusterDeletion: 'in 90 days',
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
                clusterDeletion: 'Not applicable',
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
                clusterDeletion: 'Not applicable',
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
                clusterDeletion: 'Not applicable',
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
                clusterDeletion: 'Not applicable',
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
                clusterDeletion: 'Not applicable',
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

        it(`should appear in the form for ${clusterName}`, () => {
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
            cy.get(selectors.clusterHealth.collectorStatus).should('have.text', collectorStatus);
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
                cy.get(selectors.clusterHealth.admissionControlHealthInfo.totalReadyPods).should(
                    'have.text',
                    totalReadyPods
                );
                cy.get(selectors.clusterHealth.admissionControlHealthInfo.totalDesiredPods).should(
                    'have.text',
                    totalDesiredPods
                );
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
