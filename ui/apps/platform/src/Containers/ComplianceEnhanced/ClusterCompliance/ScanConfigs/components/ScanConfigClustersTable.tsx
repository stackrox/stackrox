/* eslint-disable react/jsx-no-comment-textnodes */
// eslint-disable @typescript-eslint/ban-ts-comment
import React, { useState } from 'react';
import {
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardTitle,
    Pagination,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, ThProps, Thead, Tr } from '@patternfly/react-table';

import useURLPagination from 'hooks/useURLPagination';
import { ClusterScanStatus } from 'services/ComplianceEnhancedService';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const mockClusters = [
    {
        id: 'control-cluster',
        name: 'control-cluster',
        type: 'OCP',
        provider: 'AWS',
        region: 'us-east-1',
        status: 'Healthy',
    },
    {
        id: 'prod-cluster',
        name: 'prod-cluster',
        type: 'OSD',
        provider: 'AWS',
        region: 'us-east-1',
        status: 'Healthy',
    },
    {
        id: 'staging-cluster',
        name: 'staging-cluster',
        type: 'OCP',
        provider: 'AWS',
        region: 'us-west-1',
        status: 'Healthy',
    },
    {
        id: 'dev-cluster',
        name: 'dev-cluster',
        type: 'OCP',
        provider: 'GCP',
        region: 'us-central-1',
        status: 'Healthy',
    },
];

type ScanConfigClustersTableProps = {
    clusterScanStatuses: ClusterScanStatus[];
};

function ScanConfigClustersTable({ clusterScanStatuses }: ScanConfigClustersTableProps) {
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
        newPerPage: number,
        newPage: number
    ) => {
        setPerPage(newPerPage);
        setPage(newPage);
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
        const { clusterName, provider, status } = cluster;

        return [clusterName, null, provider, status] as (string | number | null)[];
    };

    // Note that we perform the sort as part of the component's render logic and not in onSort.
    // We shouldn't store the list of data in state because we don't want to have to sync that with props.
    const sortedClusters = clusterScanStatuses.sort((a, b) => {
        const aValue = getSortableRowValues(a)[activeSortIndex];
        const bValue = getSortableRowValues(b)[activeSortIndex];
        if (typeof aValue === 'number') {
            // Numeric sort
            if (activeSortDirection === 'asc') {
                return aValue - (bValue as number);
            }
            return (bValue as number) - aValue;
        }
        if (typeof aValue === 'string') {
            // String sort
            if (activeSortDirection === 'asc') {
                return aValue.localeCompare(bValue as string);
            }
            return (bValue as string).localeCompare(aValue);
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

    return (
        <Card>
            <CardHeader>
                <CardActions>
                    <Pagination
                        itemCount={clusterScanStatuses.length}
                        page={page}
                        onSetPage={onSetPage}
                        perPage={perPage}
                        onPerPageSelect={onPerPageSelect}
                    />
                </CardActions>
                <CardTitle component="h2">Clusters</CardTitle>
            </CardHeader>
            <CardBody>
                <TableComposable borders={false}>
                    <Thead noWrap>
                        <Tr>
                            <Th sort={getSortParams(0)}>Cluster</Th>
                            <Th>Type</Th>
                            <Th sort={getSortParams(2)}>Provider (Region)</Th>
                            <Th sort={getSortParams(3)}>Operator status</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {sortedClusters.map((cluster) => {
                            return (
                                <Tr key={cluster.clusterId}>
                                    <Td>{cluster.clusterName}</Td>
                                    {/* @ts-expect-error api */}
                                    <Td>{cluster.type || '-'}</Td>
                                    <Td>
                                        {/* @ts-expect-error api */}
                                        {cluster.provider || '-'} ({cluster.region || '-'})
                                    </Td>
                                    {/* @ts-expect-error api */}
                                    <Td>{cluster.status || '-'}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </TableComposable>
            </CardBody>
        </Card>
    );
}

export default ScanConfigClustersTable;
