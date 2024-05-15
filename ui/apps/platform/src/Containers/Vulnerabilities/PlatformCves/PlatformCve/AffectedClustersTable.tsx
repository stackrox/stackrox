import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import { Truncate } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLSortResult } from 'hooks/useURLSort';
import { TableUIState } from 'utils/getTableUIState';
import { ClusterType } from 'types/cluster.proto';

import { getPlatformEntityPagePath } from '../../utils/searchUtils';
import { displayClusterType } from '../utils/stringUtils';

export const sortFields = {
    Cluster: 'Cluster',
    ClusterType: 'Cluster Type',
    ClusterKubernetesVersion: 'Cluster Kubernetes Version',
} as const;
export const defaultSortOption = { field: 'Cluster', direction: 'asc' } as const;

export const affectedClusterFragment = gql`
    fragment AffectedClusterFragment on Cluster {
        id
        name
        type
        status {
            orchestratorMetadata {
                version
            }
        }
    }
`;

export type AffectedCluster = {
    id: string;
    name: string;
    type: ClusterType;
    status?: {
        orchestratorMetadata?: {
            version: string;
        };
    };
};

export type AffectedClustersTableProps = {
    tableState: TableUIState<AffectedCluster>;
    getSortParams: UseURLSortResult['getSortParams'];
};

function AffectedClustersTable({ tableState, getSortParams }: AffectedClustersTableProps) {
    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            role="region"
            aria-live="polite"
            aria-busy={tableState.type === 'LOADING' ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams(sortFields.Cluster)}>Cluster</Th>
                    <Th sort={getSortParams(sortFields.ClusterType)}>Cluster type</Th>
                    <Th sort={getSortParams(sortFields.ClusterKubernetesVersion)}>
                        Kubernetes version
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={3}
                emptyProps={{ message: 'No clusters have been reported for this CVE' }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map(({ id, name, type, status }) => (
                            <Tr key={id}>
                                <Td dataLabel="Cluster">
                                    <Link to={getPlatformEntityPagePath('Cluster', id)}>
                                        <Truncate position="middle" content={name} />
                                    </Link>
                                </Td>
                                <Td dataLabel="Cluster type" modifier="nowrap">
                                    {displayClusterType(type)}
                                </Td>
                                <Td dataLabel="Kubernetes version" modifier="nowrap">
                                    {status?.orchestratorMetadata?.version ?? 'Unavailable'}
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default AffectedClustersTable;
