import React from 'react';

import AdmissionControlStatus from './AdmissionControlStatus';

export default {
    title: 'AdmissionControlStatus',
    component: AdmissionControlStatus,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Admission Control Status';

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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'UNINITIALIZED',
                admissionControlHealthInfo: null,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'UNAVAILABLE',
                admissionControlHealthInfo: null,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'UNHEALTHY',
                admissionControlHealthInfo: {
                    totalReadyPods: 1,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'UNHEALTHY',
                admissionControlHealthInfo: {
                    totalReadyPods: 1,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'DEGRADED',
                admissionControlHealthInfo: {
                    totalReadyPods: 2,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'DEGRADED',
                admissionControlHealthInfo: {
                    totalReadyPods: 2,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'HEALTHY',
                admissionControlHealthInfo: {
                    totalReadyPods: 3,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'HEALTHY',
                admissionControlHealthInfo: {
                    totalReadyPods: 3,
                    totalDesiredPods: 3,
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
        <AdmissionControlStatus
            healthStatus={{
                admissionControlHealthStatus: 'HEALTHY',
                admissionControlHealthInfo: {
                    totalReadyPods: 3,
                    totalDesiredPods: 3,
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
