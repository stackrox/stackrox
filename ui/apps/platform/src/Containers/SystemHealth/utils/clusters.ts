import {
    findUpgradeState,
    getCredentialExpirationStatus,
    healthStatusStyles,
    sensorUpgradeStyles,
    styleDegraded,
    UpgradeStatus,
} from 'Containers/Clusters/cluster.helpers';

import { CategoryStyle, CountableText, nbsp } from './health';

export const problemText: CountableText = {
    plural: `clusters require your${nbsp}attention`,
    singular: `cluster requires your${nbsp}attention`,
};

// getProblemColor assumes that problem keys are in increasing order of severity:
export type ClusterStatus = 'HEALTHY' | 'UNINITIALIZED' | 'DEGRADED' | 'UNHEALTHY';
export type CollectorStatus =
    | 'HEALTHY'
    | 'UNINITIALIZED' // not relevant in System Health
    | 'UNAVAILABLE' // not relevant in System Health
    | 'DEGRADED'
    | 'UNHEALTHY';
export type AdmissionControlStatus =
    | 'HEALTHY'
    | 'UNINITIALIZED' // not relevant in System Health
    | 'UNAVAILABLE' // not relevant in System Health
    | 'DEGRADED'
    | 'UNHEALTHY';
type SensorUpgradeKey = 'current' | 'download' | 'intervention' | 'failure'; // progress in not relevant

export const clusterStatusHealthyKey: ClusterStatus = 'HEALTHY';

export interface Cluster {
    id: string;
    name: string;
    healthStatus?: {
        sensorHealthStatus: ClusterStatus;
        collectorHealthStatus: CollectorStatus;
        admissionControlHealthStatus: AdmissionControlStatus;
        overallHealthStatus: ClusterStatus;
    };
    status?: {
        certExpiryStatus?: {
            sensorCertExpiry: string;
        };
        upgradeStatus?: UpgradeStatus;
    };
}

type ClusterStatusLabelMap = Record<ClusterStatus, string>;

export const clusterStatusLabelMap: ClusterStatusLabelMap = {
    HEALTHY: 'Healthy',
    UNINITIALIZED: 'Uninitialized',
    DEGRADED: 'Degraded',
    UNHEALTHY: 'Unhealthy',
};

type ClusterStatusStyleMap = Record<ClusterStatus, CategoryStyle>;

export const clusterStatusStyleMap: ClusterStatusStyleMap = healthStatusStyles;

type ClusterStatusCountMap = Record<ClusterStatus, number>;

export const getClusterStatusCountMap = (clusters: Cluster[]): ClusterStatusCountMap => {
    // The order of keys determines the order of list items in Cluster Overview widget.
    const countMap: ClusterStatusCountMap = {
        HEALTHY: 0,
        UNINITIALIZED: 0,
        DEGRADED: 0,
        UNHEALTHY: 0,
    };

    clusters.forEach((cluster) => {
        const status = cluster.healthStatus?.overallHealthStatus;
        if (status != null && countMap[status] !== undefined) {
            countMap[status] += 1;
        }
    });

    return countMap;
};

type CountMap = Record<string, number>;

export const getCollectorStatusCountMap = (clusters: Cluster[]): CountMap => {
    const countMap: CountMap = {
        HEALTHY: 0,
        DEGRADED: 0,
        UNHEALTHY: 0,
    };

    clusters.forEach((cluster) => {
        const status = cluster.healthStatus?.collectorHealthStatus;
        if (status != null && countMap[status] !== undefined) {
            countMap[status] += 1;
        }
    });

    return countMap;
};

export const getAdmissionControlStatusCountMap = (clusters: Cluster[]): CountMap => {
    const countMap: CountMap = {
        HEALTHY: 0,
        DEGRADED: 0,
        UNHEALTHY: 0,
    };

    clusters.forEach((cluster) => {
        const status = cluster.healthStatus?.admissionControlHealthStatus;
        if (status != null && countMap[status] !== undefined) {
            countMap[status] += 1;
        }
    });

    return countMap;
};

export const getSensorStatusCountMap = (clusters: Cluster[]): CountMap => {
    const countMap: CountMap = {
        HEALTHY: 0,
        DEGRADED: 0,
        UNHEALTHY: 0,
    };

    clusters.forEach((cluster) => {
        const status = cluster.healthStatus?.sensorHealthStatus;
        if (status != null && countMap[status] !== undefined) {
            countMap[status] += 1;
        }
    });

    return countMap;
};

export const sensorUpgradeHealthyKey: SensorUpgradeKey = 'current';

type SensorUpgradeLabelMap = Record<SensorUpgradeKey, string>;

export const sensorUpgradeLabelMap: SensorUpgradeLabelMap = {
    current: 'Up to date',
    download: 'Upgrade available',
    intervention: 'Upgrade manually',
    failure: 'Upgrade failure',
};

type SensorUpgradeStyleMap = Record<SensorUpgradeKey, CategoryStyle>;

export const sensorUpgradeStyleMap: SensorUpgradeStyleMap = sensorUpgradeStyles;

// Display outdated sensor version as a problem to solve in System Health:
sensorUpgradeStyles.download.bgColor = styleDegraded.bgColor;
sensorUpgradeStyles.download.fgColor = styleDegraded.fgColor;

export const getSensorUpgradeCountMap = (clusters: Cluster[]): CountMap => {
    const countMap: CountMap = {
        current: 0,
        download: 0,
        intervention: 0,
        failure: 0,
    };

    clusters.forEach((cluster) => {
        const upgradeStateObject = findUpgradeState(cluster.status?.upgradeStatus);
        if (upgradeStateObject) {
            const { type } = upgradeStateObject;
            if (type != null && countMap[type] !== undefined) {
                countMap[type] += 1;
            }
        }
    });

    return countMap;
};

export const credentialExpirationLabelMap = {
    HEALTHY: 'Up to date',
    DEGRADED: 'Expiring in < 30 days',
    UNHEALTHY: 'Expiring in < 7 days',
};

export const getCredentialExpirationCountMap = (
    clusters: Cluster[],
    currentDateTime: Date
): CountMap => {
    const countMap: CountMap = {
        HEALTHY: 0,
        DEGRADED: 0,
        UNHEALTHY: 0,
    };

    clusters.forEach((cluster) => {
        const sensorCertExpiry = cluster.status?.certExpiryStatus?.sensorCertExpiry;
        if (sensorCertExpiry) {
            const status = getCredentialExpirationStatus(sensorCertExpiry, currentDateTime);
            countMap[status] += 1;
        }
    });

    return countMap;
};
