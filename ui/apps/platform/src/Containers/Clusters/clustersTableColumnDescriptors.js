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
import CredentialExpiration from './Components/CredentialExpiration';
import SensorUpgrade from './Components/SensorUpgrade';
import HelmIndicator from './Components/HelmIndicator';
import OperatorIndicator from './Components/OperatorIndicator';

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

    // Because of fixed checkbox width, total of column ratios must be less than 1
    // 6/8 + 1/9 + 1/10 = 0.961
    const clusterColumnsWithHealth = [
        {
            accessor: 'name',
            Header: 'Name',
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ original }) => (
                <span className="flex items-center" data-testid="cluster-name">
                    {original.name}
                    {(original.managedBy === 'MANAGER_TYPE_HELM_CHART' ||
                        (original.managedBy === 'MANAGER_TYPE_UNKNOWN' &&
                            !!original.helmConfig)) && (
                        <span className="pl-2">
                            <HelmIndicator />
                        </span>
                    )}
                    {original.managedBy === 'MANAGER_TYPE_KUBERNETES_OPERATOR' && (
                        <span className="pl-2">
                            <OperatorIndicator />
                        </span>
                    )}
                </span>
            ),
        },
        {
            Header: 'Cloud Provider',
            Cell: ({ original }) => formatCloudProvider(original.status?.providerMetadata),
            headerClassName: `w-1/9 ${defaultHeaderClassName}`,
            className: `w-1/9 ${wrapClassName} ${defaultColumnClassName}`,
        },
        {
            Header: 'Cluster Status',
            Cell: ({ original }) => {
                const safeHealthStatus = original.healthStatus || {
                    overallHealthStatus: 'UNINITIALIZED',
                };
                return <ClusterStatus healthStatus={safeHealthStatus} isList />;
            },
            headerClassName: `w-1/4 ${defaultHeaderClassName}`,
            className: `w-1/4 ${wrapClassName} ${defaultColumnClassName}`,
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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${wrapClassName} ${defaultColumnClassName}`,
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
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${wrapClassName} ${defaultColumnClassName}`,
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
