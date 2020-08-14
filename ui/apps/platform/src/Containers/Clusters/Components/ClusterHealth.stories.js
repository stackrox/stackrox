import React from 'react';

import ClusterHealth from './ClusterHealth';

export default {
    title: 'ClusterHealth',
    component: ClusterHealth,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';

const AtSide = ({ children }) => <div className={bodyClassName}>{children}</div>;

export const ClusterStatusUninitialized = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'UNINITIALIZED',
                collectorHealthInfo: null,
                healthInfoComplete: false,
                overallHealthStatus: 'UNINITIALIZED',
                sensorHealthStatus: 'UNINITIALIZED',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: null,
                sensorVersion: '3.50.0.0',
                upgradeStatus: { upgradability: 'UP_TO_DATE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2019-08-01T14:00:00Z')}
        />
    </AtSide>
);

export const SensorStatusUnhealthy = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 10,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                overallHealthStatus: 'UNHEALTHY',
                sensorHealthStatus: 'UNHEALTHY',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-07-02T13:00:00Z',
                sensorVersion: '3.50.0.0',
                upgradeStatus: { upgradability: 'UP_TO_DATE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-07-02T14:00:00Z')}
        />
    </AtSide>
);

export const CollectorStatusUnhealthy = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'UNHEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 3,
                    totalDesiredPods: 5,
                    totalRegisteredNodes: 6,
                },
                healthInfoComplete: true,
                overallHealthStatus: 'UNHEALTHY',
                sensorHealthStatus: 'HEALTHY',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-06-01T13:58:01Z',
                sensorVersion: '3.50.0.0',
                upgradeStatus: { upgradability: 'UP_TO_DATE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-06-01T14:00:00Z')}
        />
    </AtSide>
);

export const SensorStatusDegraded = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 10,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                overallHealthStatus: 'DEGRADED',
                sensorHealthStatus: 'DEGRADED',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-06-01T13:57:01Z',
                sensorVersion: '3.50.0.0',
                upgradeStatus: { upgradability: 'UP_TO_DATE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-06-01T14:00:00Z')}
        />
    </AtSide>
);

export const CollectorStatusDegraded = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'DEGRADED',
                collectorHealthInfo: {
                    totalReadyPods: 8,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                overallHealthStatus: 'DEGRADED',
                sensorHealthStatus: 'HEALTHY',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-06-01T13:58:01Z',
                sensorVersion: '3.48.0.0',
                upgradeStatus: { upgradability: 'AUTO_UPGRADE_POSSIBLE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-06-01T14:00:00Z')}
        />
    </AtSide>
);

export const CollectorStatusUnavailable = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'UNAVAILABLE',
                collectorHealthInfo: null,
                healthInfoComplete: false,
                overallHealthStatus: 'HEALTHY',
                sensorHealthStatus: 'HEALTHY',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-07-24T21:59:01Z',
                sensorVersion: '3.47.0.0',
                upgradeStatus: { upgradability: 'AUTO_UPGRADE_POSSIBLE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-07-24T22:00:00Z')}
        />
    </AtSide>
);

export const ClusterStatusHealthy = () => (
    <AtSide>
        <ClusterHealth
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 7,
                    totalDesiredPods: 7,
                    totalRegisteredNodes: 7,
                },
                healthInfoComplete: true,
                overallHealthStatus: 'HEALTHY',
                sensorHealthStatus: 'HEALTHY',
            }}
            status={{
                certExpiryStatus: { sensorCertExpiry: '2020-07-31T21:59:00Z' },
                lastContact: '2020-06-01T13:58:01Z',
                sensorVersion: '3.50.0.0',
                upgradeStatus: { upgradability: 'UP_TO_DATE' },
            }}
            centralVersion="3.50.0.0"
            currentDatetime={new Date('2020-06-01T14:00:00Z')}
        />
    </AtSide>
);
