import React, { ReactElement } from 'react';
import { TableComposable, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { UseURLSortResult } from 'hooks/useURLSort';
import { DiscoveredCluster, hasDiscoveredClustersFilter } from 'services/DiscoveredClustersService';
import { SearchFilter } from 'types/search';

import DiscoveredClustersEmptyState from './DiscoveredClustersEmptyState';

const colSpan = 4; // TODO separate Provider from

export type DiscoveredClustersTableProps = {
    clusters: DiscoveredCluster[];
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
};

function DiscoveredClustersTable({
    clusters,
    getSortParams,
    searchFilter,
}: DiscoveredClustersTableProps): ReactElement {
    return (
        <TableComposable variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th sort={getSortParams('TODO')}>Cluster</Th>
                    <Th>Type</Th>
                    <Th>Provider</Th>
                    <Th>Region</Th>
                </Tr>
            </Thead>
            {clusters.length === 0 ? (
                <DiscoveredClustersEmptyState
                    colSpan={colSpan}
                    hasFilter={hasDiscoveredClustersFilter(searchFilter)}
                />
            ) : (
                clusters.map((cluster) => {
                    const { id } = cluster;

                    return (
                        <Tr key={id}>
                            <Td dataLabel="Cluster">{'TODO'}</Td>
                            <Td dataLabel="Type">{'TODO'}</Td>
                            <Td dataLabel="Provider">{'TODO'}</Td>
                            <Td dataLabel="Region">{'TODO'}</Td>
                        </Tr>
                    );
                })
            )}
        </TableComposable>
    );
}

export default DiscoveredClustersTable;
