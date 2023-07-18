import React from 'react';
import { Button, ButtonVariant } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import DateDistanceTd from '../components/DatePhraseTd';
import EmptyTableResults from '../components/EmptyTableResults';
import { getEntityPagePath } from '../searchUtils';

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
    return (
        <TableComposable borders={false} variant="compact">
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
                            borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            <Td>
                                <Button
                                    variant={ButtonVariant.link}
                                    isInline
                                    component={LinkShim}
                                    href={getEntityPagePath('Deployment', id)}
                                >
                                    {name}
                                </Button>
                            </Td>
                            <Td>{clusterName}</Td>
                            <Td>{namespace}</Td>
                            <Td>
                                <DateDistanceTd date={created} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default DeploymentResourceTable;
