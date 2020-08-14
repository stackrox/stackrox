import React from 'react';

import CredentialExpiration from './CredentialExpiration';

export default {
    title: 'CredentialExpiration',
    component: CredentialExpiration,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Credential Expiration';

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

// 2 days ago on 7/31/2020
export const isUnhealthyDaysAgoInList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-08-03T21:58:00Z')}
        />
    </InList>
);

// 23 hours ago
export const isUnhealthyHoursAgoAtSide = () => (
    <AtSide>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-08-01T21:58:00Z')}
        />
    </AtSide>
);

// 1 minute ago
export const isUnhealthyMinuteAgoList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-31T22:00:00Z')}
        />
    </InList>
);

// 0 seconds ago
export const isUnhealthyExactTimeAtSide = () => (
    <AtSide>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-31T21:59:00Z')}
        />
    </AtSide>
);

// in 59 minutes
export const isUnhealthyInMinutesInList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-31T21:00:00Z')}
        />
    </InList>
);

// in 7 hours
export const isUnhealthyInHoursAtSide = () => (
    <AtSide>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-31T14:00:00Z')}
        />
    </AtSide>
);

// in 6 days on Friday
export const isUnhealthyInDaysOnDayInList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-24T22:00:00Z')}
        />
    </InList>
);

// in 29 days on 7/31/2020
export const isDegradedInDaysOnDateAtSide = () => (
    <AtSide>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-02T14:00:00Z')}
        />
    </AtSide>
);

// in 1 month on 7/31/2020 (date-fns edge case: 30 days)
export const isHealthyInMonthOnDateInList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-07-01T14:00:00Z')}
        />
    </InList>
);

// in 1 month on 7/31/2020 (date-fns edge case: 59 days)
export const isHealthInMonthOnDateAtSide = () => (
    <AtSide>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-06-02T14:00:00Z')}
        />
    </AtSide>
);

// in 2 months (date-fns edge case: 60 days)
export const isHealthyInMonthsInList = () => (
    <InList>
        <CredentialExpiration
            certExpiryStatus={{ sensorCertExpiry: '2020-07-31T21:59:00Z' }}
            currentDatetime={new Date('2020-06-01T14:00:00Z')}
        />
    </InList>
);
