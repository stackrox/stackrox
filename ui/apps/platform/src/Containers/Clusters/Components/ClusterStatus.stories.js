import React from 'react';

import ClusterStatus from './ClusterStatus';

export default {
    title: 'ClusterStatus',
    component: ClusterStatus,
};

// Provide realistic inherited styles.
const bodyClassName = 'font-sans text-base-600 text-base font-600';
const heading = 'Cluster Status';

const AtSide = ({ children }) => (
    <div className={bodyClassName}>
        <table style={{ width: '20rem' }}>
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

export const isUninitializedAtSide = () => (
    <AtSide>
        <ClusterStatus overallHealthStatus="UNINITIALIZED" />
    </AtSide>
);

export const isUnhealthyInList = () => (
    <InList>
        <ClusterStatus overallHealthStatus="UNHEALTHY" />
    </InList>
);

export const isDegradedAtSide = () => (
    <AtSide>
        <ClusterStatus overallHealthStatus="DEGRADED" />
    </AtSide>
);

export const isHealthyInList = () => (
    <InList>
        <ClusterStatus overallHealthStatus="HEALTHY" />
    </InList>
);
