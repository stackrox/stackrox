import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Td, Tbody } from '@patternfly/react-table';
import { Link } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useURLPagination from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { Pagination } from 'services/types';
import { ClusterType } from 'types/cluster.proto';
import { ApiSortOption } from 'types/search';
import { getTableUIState } from 'utils/getTableUIState';

import { DynamicColumnIcon } from 'Components/DynamicIcon';
import { getPaginationParams } from 'utils/searchUtils';
import {
    CLUSTER_KUBERNETES_VERSION_SORT_FIELD,
    CLUSTER_SORT_FIELD,
    CLUSTER_TYPE_SORT_FIELD,
    CVE_COUNT_SORT_FIELD,
} from '../../utils/sortFields';
import { getPlatformEntityPagePath, getRegexScopedQueryString } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';
import { displayClusterType } from '../utils/stringUtils';

const clusterListQuery = gql`
    query getPlatformClusters($query: String, $pagination: Pagination) {
        clusters(query: $query, pagination: $pagination) {
            id
            name
            clusterVulnerabilityCount(query: $query)
            type
            status {
                orchestratorMetadata {
                    version
                }
            }
        }
    }
`;

export const sortFields = [
    CLUSTER_SORT_FIELD,
    CVE_COUNT_SORT_FIELD,
    CLUSTER_TYPE_SORT_FIELD,
    CLUSTER_KUBERNETES_VERSION_SORT_FIELD,
];

export const defaultSortOption = { field: CLUSTER_SORT_FIELD, direction: 'asc' } as const;

export type Cluster = {
    id: string;
    name: string;
    clusterVulnerabilityCount: number;
    type: ClusterType;
    status?: {
        orchestratorMetadata?: {
            version: string;
        };
    };
};

export type ClustersTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function ClustersTable({
    querySearchFilter,
    isFiltered,
    pagination,
    sortOption,
    getSortParams,
    onClearFilters,
}: ClustersTableProps) {
    const { page, perPage } = pagination;
    const { data, previousData, error, loading } = useQuery<
        { clusters: Cluster[] },
        {
            query: string;
            pagination: Pagination;
        }
    >(clusterListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const tableData = data ?? previousData;

    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.clusters,
        error,
        searchFilter: querySearchFilter,
    });

    const colSpan = 4;

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams(CLUSTER_SORT_FIELD)}>Cluster</Th>
                    <Th sort={getSortParams(CVE_COUNT_SORT_FIELD)}>
                        CVEs
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th sort={getSortParams(CLUSTER_TYPE_SORT_FIELD)}>Platform type</Th>
                    <Th sort={getSortParams(CLUSTER_KUBERNETES_VERSION_SORT_FIELD)}>
                        Kubernetes version
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'No secured clusters have been detected' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map(({ id, name, clusterVulnerabilityCount, type, status }) => (
                        <Tbody key={id}>
                            <Tr>
                                <Td dataLabel="Cluster" modifier="nowrap">
                                    <Link to={getPlatformEntityPagePath('Cluster', id)}>
                                        {name}
                                    </Link>
                                </Td>
                                <Td dataLabel="CVEs">
                                    {pluralize(clusterVulnerabilityCount, 'CVE')}
                                </Td>
                                <Td dataLabel="Platform type">{displayClusterType(type)}</Td>
                                <Td dataLabel="Kubernetes version">
                                    {status?.orchestratorMetadata?.version ?? 'Unavailable'}
                                </Td>
                            </Tr>
                        </Tbody>
                    ))
                }
            />
        </Table>
    );
}

export default ClustersTable;
