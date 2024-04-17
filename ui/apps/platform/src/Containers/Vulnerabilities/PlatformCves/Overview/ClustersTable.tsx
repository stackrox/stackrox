import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Td, Tbody } from '@patternfly/react-table';
import { Link } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';

import TbodyTableState from 'Components/TableStateTemplates/TbodyTableState';
import useURLPagination from 'hooks/useURLPagination';
import { ClusterType } from 'types/cluster.proto';
import { getTableUIState } from 'utils/getTableUIState';
import { ensureExhaustive } from 'utils/type.utils';

import { DynamicColumnIcon } from 'Components/DynamicIcon';
import { getPlatformEntityPagePath, getRegexScopedQueryString } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';

function displayClusterType(type: ClusterType): string {
    switch (type) {
        case 'GENERIC_CLUSTER':
            return 'Generic';
        case 'KUBERNETES_CLUSTER':
            return 'Kubernetes';
        case 'OPENSHIFT_CLUSTER':
        case 'OPENSHIFT4_CLUSTER':
            return 'OCP';
        default:
            return ensureExhaustive(type);
    }
}

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
};

function ClustersTable({ querySearchFilter, isFiltered, pagination }: ClustersTableProps) {
    const { page, perPage } = pagination;
    const { data, previousData, error, loading } = useQuery<
        { clusters: Cluster[] },
        {
            query: string;
            pagination: {
                offset: number;
                limit: number;
            };
        }
    >(clusterListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
            },
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
            role="region"
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th>Cluster</Th>
                    <Th>
                        CVEs
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th>Cluster type</Th>
                    <Th>Kubernetes version</Th>
                </Tr>
            </Thead>
            <TbodyTableState
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'No secured clusters have been detected' }}
                renderWith={({ data }) =>
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
                                <Td dataLabel="Cluster type">{displayClusterType(type)}</Td>
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
