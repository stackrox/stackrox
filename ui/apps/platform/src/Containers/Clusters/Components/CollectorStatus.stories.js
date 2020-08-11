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
                    <th className="pl-0 pr-2 py-1 text-left whitespace-no-wrap" scope="row">
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
            collectorHealthStatus="UNINITIALIZED"
            collectorHealthInfo={null}
            lastContact={null}
            now={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isUnavailableAtSide = () => (
    <AtSide>
        <CollectorStatus
            collectorHealthStatus="UNAVAILABLE"
            collectorHealthInfo={null}
            lastContact="2020-07-28T23:59:30Z"
            now={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

export const isUnhealthyWithTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            collectorHealthStatus="UNHEALTHY"
            collectorHealthInfo={{
                totalReadyPods: 7,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="UNHEALTHY"
            lastContact="2020-07-28T00:00:00Z"
            now={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isUnhealthyWithoutTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            collectorHealthStatus="UNHEALTHY"
            collectorHealthInfo={{
                totalReadyPods: 7,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="HEALTHY"
            lastContact="2020-07-28T23:59:01"
            now={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

export const isDegradedWithTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            collectorHealthStatus="DEGRADED"
            collectorHealthInfo={{
                totalReadyPods: 8,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="DEGRADED"
            lastContact="2020-07-28T23:57:01"
            now={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);

export const isDegradedWithoutTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            collectorHealthStatus="DEGRADED"
            collectorHealthInfo={{
                totalReadyPods: 8,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="HEALTHY"
            lastContact="2020-07-28T23:59:30"
            now={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isHealthyWithTimeDifferenceInList = () => (
    <InList>
        <CollectorStatus
            collectorHealthStatus="HEALTHY"
            collectorHealthInfo={{
                totalReadyPods: 10,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="UNHEALTHY"
            lastContact="2020-07-28T23:00:00"
            now={new Date('2020-07-29T00:00:00Z')}
            isList
        />
    </InList>
);

export const isHealthyWithoutTimeDifferenceAtSide = () => (
    <AtSide>
        <CollectorStatus
            collectorHealthStatus="HEALTHY"
            collectorHealthInfo={{
                totalReadyPods: 10,
                totalDesiredPods: 10,
                totalRegisteredNodes: 12,
            }}
            sensorHealthStatus="HEALTHY"
            lastContact="2020-07-28T23:59:30"
            now={new Date('2020-07-29T00:00:00Z')}
            isList={false}
        />
    </AtSide>
);
