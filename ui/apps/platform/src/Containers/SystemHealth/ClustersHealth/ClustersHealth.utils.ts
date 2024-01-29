import {
    findUpgradeState,
    getCredentialExpirationStatus,
} from 'Containers/Clusters/cluster.helpers';
import { CertExpiryStatus } from 'Containers/Clusters/clusterTypes'; // TODO types/cluster.proto.ts
import { Cluster } from 'types/cluster.proto';

import { HealthVariant } from '../CardHeaderIcons';

export type ClusterStatus = 'HEALTHY' | 'UNHEALTHY' | 'DEGRADED' | 'UNAVAILABLE' | 'UNINITIALIZED';

export type ClusterStatusCounts = Record<ClusterStatus, number>;

function getClusterStatusCountsObject(): ClusterStatusCounts {
    return {
        HEALTHY: 0,
        UNHEALTHY: 0,
        DEGRADED: 0,
        UNAVAILABLE: 0,
        UNINITIALIZED: 0,
    };
}

export function getCertificateExpirationCounts(
    clusters: Cluster[],
    currentDatetime: Date
): ClusterStatusCounts {
    const counts = getClusterStatusCountsObject();

    clusters.forEach((cluster) => {
        switch (cluster.healthStatus.overallHealthStatus) {
            case 'UNAVAILABLE':
            case 'UNINITIALIZED': {
                counts[cluster.healthStatus.overallHealthStatus] += 1;
                break;
            }
            default: {
                const { certExpiryStatus } = cluster.status;
                const key = getCredentialExpirationStatus(
                    certExpiryStatus as CertExpiryStatus,
                    currentDatetime
                );
                counts[key] += 1;
            }
        }
    });

    return counts;
}

export function getSensorUpgradeCounts(clusters: Cluster[]): ClusterStatusCounts {
    const counts = getClusterStatusCountsObject();

    clusters.forEach((cluster) => {
        switch (cluster.healthStatus.overallHealthStatus) {
            case 'UNAVAILABLE':
            case 'UNINITIALIZED': {
                counts[cluster.healthStatus.overallHealthStatus] += 1;
                break;
            }
            default: {
                const { upgradeStatus } = cluster.status;
                const upgradeState = findUpgradeState(upgradeStatus);
                /* eslint-disable no-nested-ternary */
                const key =
                    upgradeState?.type === 'current'
                        ? 'HEALTHY'
                        : upgradeState?.type === 'failure'
                          ? 'UNHEALTHY'
                          : 'DEGRADED';
                /* eslint-enable no-nested-ternary */
                counts[key] += 1;
            }
        }
    });

    return counts;
}

export function getClusterStatusCounts(clusters: Cluster[]): ClusterStatusCounts {
    const counts = getClusterStatusCountsObject();

    clusters.forEach((cluster) => {
        counts[cluster.healthStatus.overallHealthStatus] += 1;
    });

    return counts;
}

export type ClusterHealthStatusKey =
    | 'sensorHealthStatus'
    | 'collectorHealthStatus'
    | 'admissionControlHealthStatus';

export function getClusterBecauseOfStatusCounts(
    clusters: Cluster[],
    key: ClusterHealthStatusKey
): ClusterStatusCounts {
    const counts = getClusterStatusCountsObject();

    clusters.forEach((cluster) => {
        counts[cluster.healthStatus[key]] += 1;
    });

    return counts;
}

export function getClustersHealthPhrase(counts: ClusterStatusCounts): string {
    if (counts.UNHEALTHY !== 0) {
        return `${counts.UNHEALTHY} unhealthy`;
    }
    if (counts.DEGRADED !== 0) {
        return `${counts.DEGRADED} degraded`;
    }
    if (counts.HEALTHY !== 0) {
        return `${counts.HEALTHY} healthy`;
    }
    return '';
}

export function getClustersHealthVariant(counts: ClusterStatusCounts): HealthVariant {
    if (counts.UNHEALTHY !== 0) {
        return 'danger';
    }
    if (counts.DEGRADED !== 0) {
        return 'warning';
    }
    return 'success';
}
