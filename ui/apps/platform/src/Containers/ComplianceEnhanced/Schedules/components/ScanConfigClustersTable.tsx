// eslint-disable @typescript-eslint/ban-ts-comment
import React, { useState } from 'react';
import { Flex, FlexItem, Pagination, Title } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, ThProps, Thead, Tr } from '@patternfly/react-table';

import { ClusterScanStatus } from 'services/ComplianceScanConfigurationService';

import ComplianceClusterStatus from './ComplianceClusterStatus';

type ScanConfigClustersTableProps = {
    clusterScanStatuses: ClusterScanStatus[];
    headingLevel: 'h2' | 'h3';
};

function ScanConfigClustersTable({
    clusterScanStatuses,
    headingLevel,
}: ScanConfigClustersTableProps) {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(20);

    const onSetPage = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPage: number
    ) => {
        setPage(newPage);
    };

    const onPerPageSelect = (
        _event: React.MouseEvent | React.KeyboardEvent | MouseEvent,
        newPerPage: number
    ) => {
        setPerPage(newPerPage);
    };

    // Index of the currently sorted column
    // Note: if you intend to make columns reorderable, you may instead want to use a non-numeric key
    // as the identifier of the sorted column. See the "Compound expandable" example.
    const [activeSortIndex, setActiveSortIndex] = useState<number>(0);

    // Sort direction of the currently sorted column
    const [activeSortDirection, setActiveSortDirection] = useState<'asc' | 'desc'>('asc');

    // Since OnSort specifies sorted columns by index, we need sortable values for our object by column index.
    // This example is trivial since our data objects just contain strings, but if the data was more complex
    // this would be a place to return simplified string or number versions of each column to sort by.
    const getSortableRowValues = (cluster: ClusterScanStatus): (string | number | null)[] => {
        // @ts-expect-error api
        const { clusterName, status } = cluster;

        return [clusterName, status] as (string | number | null)[];
    };

    // Note that we perform the sort as part of the component's render logic and not in onSort.
    // We shouldn't store the list of data in state because we don't want to have to sync that with props.
    const sortedClusters = clusterScanStatuses.sort((a, b) => {
        const aValue = getSortableRowValues(a)[activeSortIndex];
        const bValue = getSortableRowValues(b)[activeSortIndex];
        if (typeof aValue === 'number' && typeof bValue === 'number') {
            // Numeric sort
            if (activeSortDirection === 'asc') {
                return aValue - bValue;
            }
            return bValue - aValue;
        }
        if (typeof aValue === 'string' && typeof bValue === 'string') {
            // String sort
            if (activeSortDirection === 'asc') {
                return aValue.localeCompare(bValue);
            }
            return bValue.localeCompare(aValue);
        }

        // fallback, don't sort
        return 0;
    });

    const getSortParams = (columnIndex: number): ThProps['sort'] => ({
        sortBy: {
            index: activeSortIndex,
            direction: activeSortDirection,
            defaultDirection: 'asc', // starting sort direction when first sorting a column. Defaults to 'asc'
        },
        onSort: (_event, index, direction) => {
            setActiveSortIndex(index);
            setActiveSortDirection(direction);
        },
        columnIndex,
    });

    const startNumber = (page - 1) * perPage;
    const endNumber = page * perPage;
    const clustersWindow = sortedClusters.slice(startNumber, endNumber);

    return (
        <Flex>
            <Flex grow={{ default: 'grow' }}>
                <Title headingLevel={headingLevel}>Clusters</Title>
                <FlexItem align={{ default: 'alignRight' }}>
                    <Pagination
                        itemCount={clusterScanStatuses.length}
                        page={page}
                        onSetPage={onSetPage}
                        perPage={perPage}
                        onPerPageSelect={onPerPageSelect}
                    />
                </FlexItem>
            </Flex>
            <Table variant="compact" borders={false}>
                <Thead noWrap>
                    <Tr>
                        <Th sort={getSortParams(0)}>Cluster</Th>
                        <Th sort={getSortParams(1)} width={20}>
                            Scan schedule status
                        </Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {clustersWindow.map((cluster) => {
                        return (
                            <Tr key={cluster.clusterId}>
                                <Td dataLabel="Cluster">{cluster.clusterName}</Td>
                                <Td dataLabel="Scan schedule status">
                                    <ComplianceClusterStatus errors={cluster.errors} />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
        </Flex>
    );
}

export default ScanConfigClustersTable;
