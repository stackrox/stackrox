import React from 'react';
import { Trash2 } from 'react-feather';

import RowActionButton from 'Components/RowActionButton';
import {
    defaultHeaderClassName,
    defaultColumnClassName,
    wrapClassName,
    rtTrActionsClassName,
} from 'Components/Table';

import { formatCloudProvider } from './cluster.helpers';
import ClusterStatus from './Components/ClusterStatus';
import CollectorStatus from './Components/CollectorStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import SensorStatus from './Components/SensorStatus';
import SensorUpgrade from './Components/SensorUpgrade';

export function getColumnsForClusters({ metadata, rowActions }) {
    function renderRowActionButtons(cluster) {
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100">
                <RowActionButton
                    text="Delete cluster"
                    icon={<Trash2 className="my-1 h-4 w-4" />}
                    className="hover:bg-alert-200 text-alert-600 hover:text-alert-700"
                    onClick={rowActions.onDeleteHandler(cluster)}
                />
            </div>
        );
    }

    // Because of fixed checkbox width, total of column ratios must be less than 100%
    // 2/6 + 2/7 + 2/8 + 1/10 = 96.90674%
    const clusterColumnsWithHealth = [
        {
            accessor: 'name',
            Header: 'Name',
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Cloud Provider',
            // eslint-disable-next-line react/prop-types
            Cell: ({ original }) => formatCloudProvider(original.status?.providerMetadata),
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Cluster Status',
            Cell: ({ original }) => (
                <ClusterStatus overallHealthStatus={original.healthStatus?.overallHealthStatus} />
            ),
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Sensor Status',
            Cell: ({ original }) => (
                <SensorStatus healthStatus={original.healthStatus} currentDatetime={new Date()} />
            ),
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Collector Status',
            Cell: ({ original }) => (
                <CollectorStatus
                    healthStatus={original.healthStatus}
                    currentDatetime={new Date()}
                    isList
                />
            ),
            headerClassName: `w-1/6 ${defaultHeaderClassName}`,
            className: `w-1/6 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Sensor Upgrade',
            Cell: ({ original }) => (
                <SensorUpgrade
                    upgradeStatus={original.status?.upgradeStatus}
                    centralVersion={metadata.version}
                    sensorVersion={original.status?.sensorVersion}
                    isList
                    actionProps={{
                        clusterId: original.id,
                        upgradeSingleCluster: rowActions.upgradeSingleCluster,
                    }}
                />
            ),
            headerClassName: `w-1/7 ${defaultHeaderClassName}`,
            className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Credential Expiration',
            Cell: ({ original }) => (
                <CredentialExpiration
                    certExpiryStatus={original.status?.certExpiryStatus}
                    currentDatetime={new Date()}
                    isList
                />
            ),
            headerClassName: `w-1/7 ${defaultHeaderClassName}`,
            className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: '',
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => renderRowActionButtons(original),
        },
    ];

    return clusterColumnsWithHealth;
}

export default {
    getColumnsForClusters,
};
