import React from 'react';

import SensorStatus from './SensorStatus';

export default {
    title: 'SensorStatus',
    component: SensorStatus,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Sensor Status';

const AtSide = ({ children }) => (
    <div className={bodyClassName}>
        <table style={{ width: '20rem' }}>
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
        <SensorStatus
            sensorHealthStatus="UNINITIALIZED"
            lastContact={null}
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
        />
    </InList>
);

// Unhealthy if time difference is at least 3 minutes.
export const isUnhealthyAtSide = () => (
    <AtSide>
        <SensorStatus
            sensorHealthStatus="UNHEALTHY"
            lastContact="2020-07-28T23:56:59Z"
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
        />
    </AtSide>
);

// Degraded if time difference between 1 and 3 minutes.
export const isDegradedInList = () => (
    <InList>
        <SensorStatus
            sensorHealthStatus="DEGRADED"
            lastContact="2020-07-28T23:57:01Z"
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
        />
    </InList>
);

// Healthy if time difference less than 1 minute.
export const isHealthyAtSide = () => (
    <AtSide>
        <SensorStatus
            sensorHealthStatus="HEALTHY"
            lastContact="2020-07-28T23:59:01Z"
            currentDatetime={new Date('2020-07-29T00:00:00Z')}
        />
    </AtSide>
);
