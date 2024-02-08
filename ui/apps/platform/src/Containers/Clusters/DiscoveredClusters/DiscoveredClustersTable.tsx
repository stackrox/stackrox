import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import { UseURLSortResult } from 'hooks/useURLSort';
import { DiscoveredCluster, hasDiscoveredClustersFilter } from 'services/DiscoveredClusterService';
import { SearchFilter } from 'types/search';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

import {
    getProviderRegionText,
    getStatusIcon,
    getStatusText,
    getTypeText,
} from './DiscoveredCluster';
import DiscoveredClustersEmptyState from './DiscoveredClustersEmptyState';

const colSpan = 6;

export type DiscoveredClustersTableProps = {
    clusters: DiscoveredCluster[];
    currentDatetime: Date;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
};

function DiscoveredClustersTable({
    clusters,
    currentDatetime,
    getSortParams,
    searchFilter,
}: DiscoveredClustersTableProps): ReactElement {
    return (
        <TableComposable variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th width={25} sort={getSortParams('TODO')}>
                        Cluster
                    </Th>
                    <Th width={15}>State</Th>
                    <Th width={10}>Type</Th>
                    <Th width={15} modifier="nowrap">
                        Provider (region)
                    </Th>
                    <Th width={20} modifier="nowrap">
                        Cloud source
                    </Th>
                    <Th width={15} modifier="nowrap" sort={getSortParams('TODO')}>
                        First discovered
                    </Th>
                </Tr>
            </Thead>
            {clusters.length === 0 ? (
                <DiscoveredClustersEmptyState
                    colSpan={colSpan}
                    hasFilter={hasDiscoveredClustersFilter(searchFilter)}
                />
            ) : (
                <Tbody>
                    {clusters.map((cluster) => {
                        const { id, metadata, source, status } = cluster;
                        const { firstDiscoveredAt, name, providerType, region, type } = metadata;
                        const firstDiscoveredAsPhrase = getDistanceStrictAsPhrase(
                            firstDiscoveredAt,
                            currentDatetime
                        );

                        return (
                            <Tr key={id}>
                                <Td dataLabel="Cluster">{name}</Td>
                                <Td dataLabel="Status">
                                    <IconText
                                        icon={getStatusIcon(status)}
                                        text={getStatusText(status)}
                                    />
                                </Td>
                                <Td dataLabel="Type">{getTypeText(type)}</Td>
                                <Td dataLabel="Provider (region)" modifier="nowrap">
                                    {getProviderRegionText(providerType, region)}
                                </Td>
                                <Td dataLabel="Cloud source">{source.name}</Td>
                                <Td dataLabel="First discovered" modifier="nowrap">
                                    {firstDiscoveredAsPhrase}
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            )}
        </TableComposable>
    );
}

export default DiscoveredClustersTable;
