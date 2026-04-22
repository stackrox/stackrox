import type { ReactNode } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Label, LabelGroup, Tooltip } from '@patternfly/react-core';
import { gql } from '@apollo/client';

import { getDateTime } from 'utils/dateUtils';
import type { UseURLSortResult } from 'hooks/useURLSort';
import { generateVisibilityForColumns } from 'hooks/useManagedColumns';
import type { ManagedColumns } from 'hooks/useManagedColumns';
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
        state: string;
        deleted: string | null;
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
            state
            deleted
        }
    }
`;

/**
 * Same fields as DeploymentResources but on ImageV2; when ROX_FLATTEN_IMAGE_DATA is enabled,
 * we call ImageV2 resolver which returns ImageV2 type.
 */
export const deploymentResourcesV2Fragment = gql`
    fragment DeploymentResourcesV2 on ImageV2 {
        deploymentCount(query: $query)
        deployments(query: $query, pagination: $pagination) {
            id
            name
            type
            clusterName
            namespace
            created
            state
            deleted
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
        <Table variant="compact">
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
            {data.deployments.map(
                ({ id, name, type, clusterName, namespace, created, state, deleted }) => {
                    const labels: ReactNode[] = [];
                    if (state === 'DEPLOYMENT_STATE_DELETED' && deleted) {
                        labels.push(
                            <Tooltip key="deleted" content={`Deleted: ${getDateTime(deleted)}`}>
                                <Label isCompact variant="outline" color="red">
                                    Deleted
                                </Label>
                            </Tooltip>
                        );
                    }

                    return (
                        <Tbody key={id}>
                            <Tr
                                style={labels.length !== 0 ? { borderBlockEnd: 'none' } : undefined}
                            >
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
                                <Td
                                    dataLabel="Namespace"
                                    className={getVisibilityClass('namespace')}
                                >
                                    {namespace}
                                </Td>
                                <Td dataLabel="Created" className={getVisibilityClass('created')}>
                                    <DateDistance date={created} />
                                </Td>
                            </Tr>
                            {labels.length !== 0 && (
                                <Tr>
                                    <Td colSpan={4} style={{ paddingTop: 0 }}>
                                        <LabelGroup isCompact numLabels={labels.length}>
                                            {labels}
                                        </LabelGroup>
                                    </Td>
                                </Tr>
                            )}
                        </Tbody>
                    );
                }
            )}
        </Table>
    );
}

export default DeploymentResourceTable;
