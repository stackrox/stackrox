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

import IconText from 'Components/PatternFly/IconText/IconText';
import { ClusterScanStatus } from 'services/ComplianceEnhancedService';
import { getClusterStatusObject } from '../compliance.scanConfigs.utils';

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
                            <Th sort={getSortParams(1)} width={20}>
                                Operator status
                            </Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {clustersWindow.map((cluster) => {
                            const statusObj = getClusterStatusObject(cluster.errors);

                            return (
                                <Tr key={cluster.clusterId}>
                                    <Td dataLabel="Cluster">{cluster.clusterName}</Td>
                                    <Td dataLabel="Operator status">
                                        <IconText
                                            icon={statusObj.icon}
                                            text={statusObj.statusText}
                                        />
                                    </Td>
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
