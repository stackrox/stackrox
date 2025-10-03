import React, { useState } from 'react';
import type { MouseEvent, ReactElement } from 'react';
import {
    ActionsColumn,
    ExpandableRowContent,
    InnerScrollContainer,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { Cluster } from 'types/cluster.proto';
import type { ClusterIdToRetentionInfo } from 'types/clusterService.proto';
import type { TableUIState } from 'utils/getTableUIState';

import { formatCloudProvider } from './cluster.helpers';
import ClusterStatusGrid from './ClusterStatusGrid';
import type { CertExpiryStatus } from './clusterTypes';
import ClusterDeletion from './Components/ClusterDeletion';
import ClusterNameWithTypeIcon from './Components/ClusterNameWithTypeIcon';
import ClusterStatus from './Components/ClusterStatus';
import CredentialExpiration from './Components/CredentialExpiration';
import SensorUpgrade from './Components/SensorUpgrade';
import SensorUpgradePanel from './Components/SensorUpgradePanel';

export type ClustersTableProps = {
    centralVersion: string;
    clusterIdToRetentionInfo: ClusterIdToRetentionInfo;
    tableState: TableUIState<Cluster>;
    selectedClusterIds: string[];
    onClearFilters: () => void;
    onDeleteCluster: (cluster: Cluster) => (event: MouseEvent) => void;
    toggleAllClusters: () => void;
    toggleCluster: (clusterId) => void;
    upgradeSingleCluster: (clusterId: string) => void;
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
    upgradeSingleCluster,
}: ClustersTableProps): ReactElement {
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

    const colSpan = 8;

    return (
        <InnerScrollContainer>
            <Table>
                <Thead>
                    <Tr>
                        <Th
                            modifier="fitContent"
                            select={{
                                onSelect: () => toggleAllClusters(),
                                isSelected:
                                    tableState.type === 'COMPLETE' &&
                                    tableState.data.length === selectedClusterIds.length,
                            }}
                        />
                        <Th>Cluster</Th>
                        <Th>Provider (Region)</Th>
                        <Th>Cluster status</Th>
                        <Th width={20}>Sensor upgrade status</Th>
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

                                const isStatusUnavailable =
                                    !clusterInfo.healthStatus ||
                                    clusterInfo.healthStatus.overallHealthStatus === 'UNAVAILABLE';

                                return (
                                    <Tbody
                                        isExpanded={isRowExpanded(clusterId)}
                                        key={clusterInfo.id}
                                    >
                                        <Tr>
                                            <Td
                                                select={{
                                                    rowIndex,
                                                    onSelect: () => toggleCluster(clusterId),
                                                    isSelected:
                                                        selectedClusterIds.includes(clusterId),
                                                }}
                                            />
                                            <Td dataLabel="Cluster" style={{ minWidth: '200px' }}>
                                                <ClusterNameWithTypeIcon cluster={clusterInfo} />
                                            </Td>
                                            <Td dataLabel="Provider (Region)">{provider}</Td>
                                            <Td
                                                dataLabel="Cluster status"
                                                compoundExpand={
                                                    !isStatusUnavailable
                                                        ? {
                                                              isExpanded: isCellExpanded(
                                                                  clusterId,
                                                                  EXPANDABLE_COLUMN.STATUS
                                                              ),
                                                              onToggle: () =>
                                                                  toggle(
                                                                      clusterId,
                                                                      EXPANDABLE_COLUMN.STATUS
                                                                  ),
                                                              rowIndex,
                                                              columnIndex: 3,
                                                          }
                                                        : undefined
                                                }
                                            >
                                                <ClusterStatus
                                                    healthStatus={clusterInfo?.healthStatus}
                                                />
                                            </Td>
                                            <Td
                                                dataLabel="Sensor upgrade status"
                                                compoundExpand={
                                                    clusterInfo?.status?.upgradeStatus
                                                        ? {
                                                              isExpanded: isCellExpanded(
                                                                  clusterId,
                                                                  EXPANDABLE_COLUMN.SENSOR
                                                              ),
                                                              onToggle: () =>
                                                                  toggle(
                                                                      clusterId,
                                                                      EXPANDABLE_COLUMN.SENSOR
                                                                  ),
                                                              rowIndex,
                                                              columnIndex: 4,
                                                          }
                                                        : undefined
                                                }
                                            >
                                                <SensorUpgrade
                                                    upgradeStatus={
                                                        clusterInfo.status?.upgradeStatus
                                                    }
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
                                        {clusterInfo.healthStatus &&
                                            isCellExpanded(clusterId, EXPANDABLE_COLUMN.STATUS) && (
                                                <Tr isExpanded>
                                                    <Td colSpan={colSpan}>
                                                        <ExpandableRowContent>
                                                            <ClusterStatusGrid
                                                                healthStatus={
                                                                    clusterInfo.healthStatus
                                                                }
                                                            />
                                                        </ExpandableRowContent>
                                                    </Td>
                                                </Tr>
                                            )}
                                        {isCellExpanded(clusterId, EXPANDABLE_COLUMN.SENSOR) && (
                                            <Tr isExpanded>
                                                <Td colSpan={colSpan}>
                                                    <ExpandableRowContent>
                                                        <SensorUpgradePanel
                                                            centralVersion={centralVersion}
                                                            sensorVersion={
                                                                clusterInfo.status?.sensorVersion
                                                            }
                                                            upgradeStatus={
                                                                clusterInfo.status?.upgradeStatus
                                                            }
                                                            actionProps={{
                                                                clusterId: clusterInfo.id,
                                                                upgradeSingleCluster,
                                                            }}
                                                        />
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
        </InnerScrollContainer>
    );
}

export default ClustersTable;
