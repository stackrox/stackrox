import React from 'react';
import { generatePath, Link } from 'react-router-dom';
import { Button, ButtonVariant, Tooltip } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { ClusterCheckStatus } from 'services/ComplianceResultsService';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';

import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';

export type CheckDetailsTableProps = {
    currentDatetime: Date;
    profileName: string;
    tableState: TableUIState<ClusterCheckStatus>;
};

function CheckDetailsTable({ currentDatetime, profileName, tableState }: CheckDetailsTableProps) {
    return (
        <>
            <Table>
                <Thead>
                    <Tr>
                        <Th>Cluster</Th>
                        <Th>Last scanned</Th>
                        <Th>Compliance status</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    errorProps={{
                        title: 'There was an error loading results for this check',
                    }}
                    emptyProps={{
                        message: 'No results found for this check',
                    }}
                    filteredEmptyProps={{
                        title: 'No results found',
                        message: 'Clear all filters and try again',
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((clusterInfo) => {
                                const {
                                    cluster: { clusterId, clusterName },
                                    lastScanTime,
                                    status,
                                } = clusterInfo;
                                const clusterStatusObject = getClusterResultsStatusObject(status);
                                const firstDiscoveredAsPhrase = getDistanceStrictAsPhrase(
                                    lastScanTime,
                                    currentDatetime
                                );

                                return (
                                    <Tr key={clusterId}>
                                        <Td dataLabel="Cluster">
                                            <Link
                                                to={generatePath(coverageClusterDetailsPath, {
                                                    clusterId,
                                                    profileName,
                                                })}
                                            >
                                                {clusterName}
                                            </Link>
                                        </Td>
                                        <Td dataLabel="Last scanned">{firstDiscoveredAsPhrase}</Td>
                                        <Td dataLabel="Compliance status">
                                            <Tooltip content={clusterStatusObject.tooltipText}>
                                                <Button isInline variant={ButtonVariant.link}>
                                                    <IconText
                                                        icon={clusterStatusObject.icon}
                                                        text={clusterStatusObject.statusText}
                                                    />
                                                </Button>
                                            </Tooltip>
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
        </>
    );
}

export default CheckDetailsTable;
