import React from 'react';

import CollectorStatus from './CollectorStatus';

export default {
    title: 'CollectorStatus',
    component: CollectorStatus,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Collector Status';

const AtSide = ({ children }) => (
    <div className={bodyClassName}>
        <table style={{ width: '24rem' }}>
            <tbody>
                <tr className="align-top leading-normal">
                    <th className="pl-0 pr-2 py-1 text-left whitespace-nowrap" scope="row">
                        {heading}
                    </th>
                    <td className="px-0 py-1">{children}</td>
                </tr>
            </tbody>
        </table>
    </div>
);

const InList = ({ children }) => (
    <div className={bodyClassName}>
        <div className="ReactTable" style={{ fontSize: '0.75rem', width: '18rem' }}>
            <div className="rt-table" role="grid">
                <div className="rt-thead pl-3">
                    <div className="rt-tr" role="row">
                        <div className="rt-th px-2 py-4 pb-3 font-700 text-left">
                            <div>{heading}</div>
                        </div>
                    </div>
                </div>
                <div className="rt-tbody">
                    <div className="rt-tr" role="row">
                        <div className="rt-td p-2 flex items-center text-left">{children}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>
);

export const isUninitializedInList = () => (
    <InList>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'UNINITIALIZED',
                collectorHealthInfo: null,
                healthInfoComplete: false,
                sensorHealthStatus: 'UNINITIALIZED',
                lastContact: null,
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isUnavailableAtSide = () => (
    <AtSide>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'UNAVAILABLE',
                collectorHealthInfo: null,
                healthInfoComplete: false,
                sensorHealthStatus: 'HEALTHY',
                lastContact: '2020-07-28T23:59:30Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

// 1 day ago
export const isUnhealthyWithTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'UNHEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 7,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'UNHEALTHY',
                lastContact: '2020-07-28T00:00:00Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isUnhealthyWithoutTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'UNHEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 7,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'HEALTHY',
                lastContact: '2020-07-28T23:59:01Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

// 2 minutes ago
export const isDegradedWithTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'DEGRADED',
                collectorHealthInfo: {
                    totalReadyPods: 8,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'DEGRADED',
                lastContact: '2020-07-28T23:57:01Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

export const isDegradedWithoutTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'DEGRADED',
                collectorHealthInfo: {
                    totalReadyPods: 8,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'HEALTHY',
                lastContact: '2020-07-28T23:59:30Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

// 1 hour ago
export const isHealthyWithTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 10,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'UNHEALTHY',
                lastContact: '2020-07-28T23:00:00Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isHealthyWithoutTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 10,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: true,
                sensorHealthStatus: 'HEALTHY',
                lastContact: '2020-07-28T23:59:30Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

// Possible future scenario:
// Central expects additional collector health information,
// but outdated Sensor version does not provide the info.
export const isHealthyWithoutCompleteHealthInfoAtSide = () => (
    <AtSide>
        <CollectorStatus
            healthStatus={{
                collectorHealthStatus: 'HEALTHY',
                collectorHealthInfo: {
                    totalReadyPods: 10,
                    totalDesiredPods: 10,
                    totalRegisteredNodes: 12,
                },
                healthInfoComplete: false,
                sensorHealthStatus: 'HEALTHY',
                lastContact: '2020-07-28T23:59:30Z',
            }}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);
