import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import { UseURLSortResult } from 'hooks/useURLSort';
import { generateVisibilityForColumns, ManagedColumns } from 'hooks/useManagedColumns';
import DateDistance from 'Components/DateDistance';
import EmptyTableResults from '../components/EmptyTableResults';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export const deploymentResourcesTableId = 'DeploymentResourcesTable';

export const defaultColumns = {
    name: {
        title: 'Name',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cluster: {
        title: 'Cluster',
        isShownByDefault: true,
    },
    namespace: {
        title: 'Namespace',
        isShownByDefault: true,
    },
    created: {
        title: 'Created',
        isShownByDefault: true,
    },
} as const;

export type DeploymentResources = {
    deploymentCount: number;
    deployments: {
        id: string;
        name: string;
        type: string;
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
            type
            clusterName
            namespace
            created
        }
    }
`;

export type DeploymentResourceTableProps = {
    data: DeploymentResources;
    getSortParams: UseURLSortResult['getSortParams'];
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function DeploymentResourceTable({
    data,
    getSortParams,
    columnVisibilityState,
}: DeploymentResourceTableProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th className={getVisibilityClass('name')} sort={getSortParams('Deployment')}>
                        Name
                    </Th>
                    <Th className={getVisibilityClass('cluster')} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                    <Th
                        className={getVisibilityClass('namespace')}
                        sort={getSortParams('Namespace')}
                    >
                        Namespace
                    </Th>
                    <Th className={getVisibilityClass('created')}>Created</Th>
                </Tr>
            </Thead>
            {data.deployments.length === 0 && <EmptyTableResults colSpan={4} />}
            {data.deployments.map(({ id, name, type, clusterName, namespace, created }) => {
                return (
                    <Tbody
                        key={id}
                        style={{
                            borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            <Td dataLabel="Name" className={getVisibilityClass('name')}>
                                <Link
                                    to={urlBuilder.workloadDetails(
                                        { id, namespace, name, type },
                                        vulnerabilityState
                                    )}
                                >
                                    {name}
                                </Link>
                            </Td>
                            <Td dataLabel="Cluster" className={getVisibilityClass('cluster')}>
                                {clusterName}
                            </Td>
                            <Td dataLabel="Namespace" className={getVisibilityClass('namespace')}>
                                {namespace}
                            </Td>
                            <Td dataLabel="Created" className={getVisibilityClass('created')}>
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
