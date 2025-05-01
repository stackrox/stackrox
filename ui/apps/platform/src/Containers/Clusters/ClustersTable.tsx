import React, { useState } from 'react';
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
import { Cluster } from 'types/cluster.proto';
import { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import { TableUIState } from 'utils/getTableUIState';

import { formatCloudProvider } from './cluster.helpers';
import { CertExpiryStatus, ClusterHealthStatus } from './clusterTypes';
import ClusterDeletion from './Components/ClusterDeletion';
import ClusterNameWithTypeIcon from './Components/ClusterNameWithTypeIcon';
import ClusterStatus from './Components/ClusterStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import SensorUpgrade from './Components/SensorUpgrade';

export type ClustersTableProps = {
    centralVersion: string;
    clusterIdToRetentionInfo: ClusterIdToRetentionInfo;
    tableState: TableUIState<Cluster>;
    selectedClusterIds: string[];
    onClearFilters: () => void;
    onDeleteCluster: (cluster: Cluster) => (event: React.MouseEvent) => void;
    toggleAllClusters: () => void;
    toggleCluster: (clusterId) => void;
};

export const EXPANDABLE_COLUMN = {
    STATUS: 'status',
    SENSOR: 'sensor',
} as const;
type ExpandableColumnId = (typeof EXPANDABLE_COLUMN)[keyof typeof EXPANDABLE_COLUMN];

type ExpansionMap = Record<string, ExpandableColumnId | null>;

function ClustersTable({
    centralVersion,
    clusterIdToRetentionInfo,
    tableState,
    selectedClusterIds,
    onClearFilters,
    onDeleteCluster,
    toggleAllClusters,
    toggleCluster,
}: ClustersTableProps) {
    const [expanded, setExpanded] = useState<ExpansionMap>({});

    function toggle(clusterId: string, col: ExpandableColumnId) {
        setExpanded((prev) => ({
            ...prev,
            [clusterId]: prev[clusterId] === col ? null : col,
        }));
    }

    function isCellExpanded(clusterId: string, col: ExpandableColumnId) {
        return expanded[clusterId] === col;
    }

    function isRowExpanded(clusterId: string) {
        return expanded[clusterId] != null;
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
                emptyProps={{
                    title: 'No clusters found',
                }}
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
                                    <Tr>
                                        <Td
                                            select={{
                                                rowIndex,
                                                onSelect: () => toggleCluster(clusterId),
                                                isSelected: selectedClusterIds.includes(clusterId),
                                            }}
                                        />
                                        <Td dataLabel="Cluster">
                                            <ClusterNameWithTypeIcon cluster={clusterInfo} />
                                        </Td>
                                        <Td dataLabel="Provider (Region)">{provider}</Td>
                                        <Td
                                            dataLabel="Status"
                                            compoundExpand={{
                                                isExpanded: isCellExpanded(
                                                    clusterId,
                                                    EXPANDABLE_COLUMN.STATUS
                                                ),
                                                onToggle: () =>
                                                    toggle(clusterId, EXPANDABLE_COLUMN.STATUS),
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
                                                isExpanded: isCellExpanded(
                                                    clusterId,
                                                    EXPANDABLE_COLUMN.SENSOR
                                                ),
                                                onToggle: () =>
                                                    toggle(clusterId, EXPANDABLE_COLUMN.SENSOR),
                                                rowIndex,
                                                columnIndex: 4,
                                            }}
                                        >
                                            {/* TODO: needs update for upgrade */}
                                            <SensorUpgrade
                                                upgradeStatus={clusterInfo.status?.upgradeStatus}
                                                centralVersion={centralVersion}
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
                                    {isCellExpanded(clusterId, EXPANDABLE_COLUMN.STATUS) && (
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
                                    {isCellExpanded(clusterId, EXPANDABLE_COLUMN.SENSOR) && (
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
