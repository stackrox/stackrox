import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { interceptAndOverrideFeatureFlags } from '../../helpers/request';

import { clustersAlias, visitClusters } from './Clusters.helpers';

const metadata = {
    version: '5.0.0',
    buildFlavor: 'development',
    releaseBuild: false,
    licenseStatus: 'VALID',
    compatibleSensorVersions: ['4.7', '4.8', '4.9', '5.0', '5.1', '5.2', '5.3'],
};

function makeCluster(name, sensorVersionCompatibility, sensorVersion = '5.0.0') {
    return {
        id: name,
        name,
        type: 'KUBERNETES_CLUSTER',
        labels: {},
        mainImage: 'quay.io/stackrox-io/main',
        collectorImage: '',
        centralApiEndpoint: 'central.stackrox.svc:443',
        runtimeSupport: false,
        collectionMethod: 'CORE_BPF',
        admissionController: false,
        admissionControllerUpdates: false,
        admissionControllerEvents: false,
        status: {
            sensorVersion,
            sensorVersionCompatibility,
            upgradeStatus: {
                upgradability: 'UP_TO_DATE',
                upgradabilityStatusReason: '',
                mostRecentProcess: null,
            },
            certExpiryStatus: { sensorCertExpiry: '2027-01-01T00:00:00Z' },
        },
        dynamicConfig: {
            admissionControllerConfig: {
                enabled: false,
                timeoutSeconds: 3,
                scanInline: false,
                disableBypass: false,
                enforceOnUpdates: false,
            },
            registryOverride: '',
            disableAuditLogs: false,
            autoLockProcessBaselinesConfig: null,
        },
        tolerationsConfig: { disabled: false },
        priority: '0',
        healthStatus: {
            sensorHealthStatus: 'HEALTHY',
            collectorHealthStatus: 'HEALTHY',
            overallHealthStatus: 'HEALTHY',
            admissionControlHealthStatus: 'HEALTHY',
            scannerHealthStatus: 'UNINITIALIZED',
            lastContact: '2026-08-07T00:00:00Z',
            healthInfoComplete: true,
            id: name,
        },
        slimCollector: true,
        helmConfig: null,
        mostRecentSensorId: null,
        auditLogState: {},
        sensorCapabilities: [],
        initBundleId: '',
        managedBy: 'MANAGER_TYPE_MANUAL',
        admissionControllerFailOnError: false,
    };
}

const clustersResponse = {
    clusters: [
        makeCluster('matched-cluster', 'SENSOR_VERSION_COMPATIBILITY_MATCHED'),
        makeCluster(
            'compatible-behind-cluster',
            'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND',
            '4.9.0'
        ),
        makeCluster(
            'compatible-ahead-cluster',
            'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD',
            '5.1.0'
        ),
        makeCluster(
            'incompatible-behind-cluster',
            'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND',
            '4.6.0'
        ),
        makeCluster(
            'incompatible-ahead-cluster',
            'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD',
            '5.4.0'
        ),
        makeCluster('unknown-cluster', 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN', ''),
        makeCluster('unexpected-cluster', 'SOME_FUTURE_ENUM_VALUE', '5.0.0'),
    ],
    clusterIdToRetentionInfo: {},
};

describe('Clusters Sensor Compatibility Status', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_SENSOR_COMPATIBILITY_STATUS')) {
            this.skip();
        }
    });

    beforeEach(() => {
        interceptAndOverrideFeatureFlags({ ROX_SENSOR_COMPATIBILITY_STATUS: true });
        cy.intercept('GET', 'v1/metadata', { body: metadata }).as('metadata');
    });

    it('should display the sensor compatibility status column and cell values', () => {
        visitClusters({
            [clustersAlias]: { body: clustersResponse },
        });
        cy.wait('@metadata');

        cy.get('th:contains("Sensor compatibility status")');

        const compatCell = 'td[data-label="Sensor compatibility status"]';
        cy.get(compatCell).eq(0).should('contain', 'Matched');
        cy.get(compatCell).eq(1).should('contain', 'Compatible (Behind)');
        cy.get(compatCell).eq(2).should('contain', 'Compatible (Ahead)');
        cy.get(compatCell).eq(3).should('contain', 'Incompatible (Behind)');
        cy.get(compatCell).eq(4).should('contain', 'Incompatible (Ahead)');
        cy.get(compatCell).eq(5).should('contain', 'Unknown');
        // Unexpected enum value from server should fall back to Unknown
        cy.get(compatCell).eq(6).should('contain', 'Unknown');
    });
});
