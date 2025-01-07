import React from 'react';
import { Link } from 'react-router-dom';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import { UseURLSortResult } from 'hooks/useURLSort';
import DateDistance from 'Components/DateDistance';
import EmptyTableResults from '../components/EmptyTableResults';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export type DeploymentResources = {
    deploymentCount: number;
    deployments: {
        id: string;
        name: string;
        clusterName: string;
        namespace: string;
        created: string | null;
    }[];
};

export const deploymentResourcesFragment = gql`
    fragment DeploymentResources on Image {
        deploymentCount(query: $query)
        deployments(query: $query, pagination: $pagination) {
            id
            name
            clusterName
            namespace
            created
        }
    }
`;

export type DeploymentResourceTableProps = {
    data: DeploymentResources;
    getSortParams: UseURLSortResult['getSortParams'];
};

function DeploymentResourceTable({ data, getSortParams }: DeploymentResourceTableProps) {
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Deployment')}>Name</Th>
                    <Th sort={getSortParams('Cluster')}>Cluster</Th>
                    <Th sort={getSortParams('Namespace')}>Namespace</Th>
                    <Th>Created</Th>
                </Tr>
            </Thead>
            {data.deployments.length === 0 && <EmptyTableResults colSpan={4} />}
            {data.deployments.map(({ id, name, clusterName, namespace, created }) => {
                return (
                    <Tbody
                        key={id}
                        style={{
                            borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            <Td dataLabel="Name">
                                <Link
                                    to={getAbsoluteUrl(
                                        getWorkloadEntityPagePath(
                                            'Deployment',
                                            id,
                                            vulnerabilityState
                                        )
                                    )}
                                >
                                    {name}
                                </Link>
                            </Td>
                            <Td dataLabel="Cluster">{clusterName}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td dataLabel="Created">
                                <DateDistance date={created} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default DeploymentResourceTable;
