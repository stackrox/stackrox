import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Flex } from '@patternfly/react-core';
import {
    ActionsColumn,
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useMetadata from 'hooks/useMetadata';
import { Cluster } from 'types/cluster.proto';
import { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import { TableUIState } from 'utils/getTableUIState';

import { formatCloudProvider } from './cluster.helpers';
import { CertExpiryStatus, ClusterHealthStatus } from './clusterTypes';
import ClusterDeletion from './Components/ClusterDeletion';
import ClusterStatus from './Components/ClusterStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import HelmIndicator from './Components/HelmIndicator';
import OperatorIndicator from './Components/OperatorIndicator';
import SensorUpgrade from './Components/SensorUpgrade';

export type ClustersTableProps = {
    clusterIdToRetentionInfo: ClusterIdToRetentionInfo;
    tableState: TableUIState<Cluster>;
    selectedClusterIds: string[];
    onClearFilters: () => void;
    onDeleteCluster: (cluster: Cluster) => (event: React.MouseEvent) => void;
    toggleAllClusters: () => void;
    toggleCluster: (clusterId) => void;
};

export const COL = {
    STATUS: 'status',
    SENSOR: 'sensor',
} as const;
type ColId = (typeof COL)[keyof typeof COL];

type ExpansionMap = Record<string, ColId | null>;

function ClustersTable({
    clusterIdToRetentionInfo,
    tableState,
    selectedClusterIds,
    onClearFilters,
    onDeleteCluster,
    toggleAllClusters,
    toggleCluster,
}: ClustersTableProps) {
    const metadata = useMetadata();
    const [expanded, setExpanded] = useState<ExpansionMap>({});

    function toggle(clusterId: string, col: ColId) {
        setExpanded((prev) => ({
            ...prev,
            [clusterId]: prev[clusterId] === col ? null : col,
        }));
    }

    function isCellExpanded(clusterId: string, col: ColId) {
        return expanded[clusterId] === col;
    }

    function isRowExpanded(clusterId: string) {
        return expanded[clusterId] != null;
    }

    function isHelmManaged(cluster: Cluster) {
        return (
            cluster.managedBy === 'MANAGER_TYPE_HELM_CHART' ||
            (cluster.managedBy === 'MANAGER_TYPE_UNKNOWN' && !!cluster.helmConfig)
        );
    }

    function isOperatorManaged(cluster: Cluster) {
        return cluster.managedBy === 'MANAGER_TYPE_KUBERNETES_OPERATOR';
    }

    const colSpan = 9;

    return (
        <Table>
            <Thead>
                <Tr>
                    <Th
                        select={{
                            onSelect: () => toggleAllClusters(),
                            isSelected:
                                tableState.type === 'COMPLETE' &&
                                tableState.data.length === selectedClusterIds.length,
                        }}
                    />
                    <Th>Cluster</Th>
                    <Th>Provider (Region)</Th>
                    <Th>Status</Th>
                    <Th>Sensor upgrade status</Th>
                    <Th>Credential expiration</Th>
                    <Th>Cluster deletion</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                errorProps={{
                    title: 'There was an error loading cluster information',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) => (
                    <>
                        {data.map((clusterInfo, rowIndex) => {
                            const provider = formatCloudProvider(
                                clusterInfo.status?.providerMetadata
                            );
                            const clusterId = clusterInfo.id;
                            return (
                                <Tbody isExpanded={isRowExpanded(clusterId)} key={clusterInfo.id}>
                                    <Tr key={clusterInfo.id}>
                                        <Td
                                            select={{
                                                rowIndex,
                                                onSelect: () => toggleCluster(clusterId),
                                                isSelected: selectedClusterIds.includes(clusterId),
                                            }}
                                        />
                                        <Td dataLabel="Cluster">
                                            <Flex
                                                alignItems={{ default: 'alignItemsCenter' }}
                                                columnGap={{ default: 'columnGapXs' }}
                                                flexWrap={{ default: 'nowrap' }}
                                            >
                                                <Link to={clusterId} className="">
                                                    {clusterInfo.name}
                                                </Link>
                                                {isHelmManaged(clusterInfo) && <HelmIndicator />}
                                                {isOperatorManaged(clusterInfo) && (
                                                    <OperatorIndicator />
                                                )}
                                            </Flex>
                                        </Td>
                                        <Td dataLabel="Provider (Region)">{provider}</Td>
                                        <Td
                                            dataLabel="Status"
                                            compoundExpand={{
                                                isExpanded: isCellExpanded(clusterId, COL.STATUS),
                                                onToggle: () => toggle(clusterId, COL.STATUS),
                                                rowIndex,
                                                columnIndex: 3,
                                            }}
                                        >
                                            {/* TODO: needs update for upgrade */}
                                            <ClusterStatus
                                                healthStatus={
                                                    clusterInfo?.healthStatus as ClusterHealthStatus
                                                }
                                            />
                                        </Td>
                                        <Td
                                            dataLabel="Sensor upgrade status"
                                            compoundExpand={{
                                                isExpanded: isCellExpanded(clusterId, COL.SENSOR),
                                                onToggle: () => toggle(clusterId, COL.SENSOR),
                                                rowIndex,
                                                columnIndex: 4,
                                            }}
                                        >
                                            {/* TODO: needs update for upgrade */}
                                            <SensorUpgrade
                                                upgradeStatus={clusterInfo.status?.upgradeStatus}
                                                centralVersion={metadata.version}
                                                sensorVersion={clusterInfo.status?.sensorVersion}
                                                isList
                                            />
                                        </Td>
                                        <Td dataLabel="Credential expiration">
                                            {/* TODO: needs update for upgrade */}
                                            <CredentialExpiration
                                                certExpiryStatus={
                                                    clusterInfo.status
                                                        ?.certExpiryStatus as CertExpiryStatus
                                                }
                                                autoRefreshEnabled={clusterInfo.sensorCapabilities?.includes(
                                                    'SecuredClusterCertificatesRefresh'
                                                )}
                                                isList
                                            />
                                        </Td>
                                        <Td dataLabel="Cluster deletion">
                                            <ClusterDeletion
                                                clusterRetentionInfo={
                                                    clusterIdToRetentionInfo[clusterId] ?? null
                                                }
                                            />
                                        </Td>
                                        <Td isActionCell>
                                            <ActionsColumn
                                                items={[
                                                    {
                                                        title: 'Delete cluster',
                                                        onClick: (event) => {
                                                            onDeleteCluster(clusterInfo)(event);
                                                        },
                                                    },
                                                ]}
                                            />
                                        </Td>
                                    </Tr>
                                    {isCellExpanded(clusterId, COL.STATUS) && (
                                        <Tr isExpanded>
                                            <Td colSpan={colSpan}>
                                                <ExpandableRowContent>
                                                    <div className="pf-v5-u-text-align-center">
                                                        *status details*
                                                    </div>
                                                </ExpandableRowContent>
                                            </Td>
                                        </Tr>
                                    )}
                                    {isCellExpanded(clusterId, COL.SENSOR) && (
                                        <Tr isExpanded>
                                            <Td colSpan={colSpan}>
                                                <ExpandableRowContent>
                                                    <div className="pf-v5-u-text-align-center">
                                                        *sensor details*
                                                    </div>
                                                </ExpandableRowContent>
                                            </Td>
                                        </Tr>
                                    )}
                                </Tbody>
                            );
                        })}
                    </>
                )}
            />
        </Table>
    );
}

export default ClustersTable;
