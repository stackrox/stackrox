import React from 'react';
import { Card } from '@patternfly/react-core';
import { Tbody, Tr, Td, TableComposable, Th, Thead } from '@patternfly/react-table';

import { ProcessListeningOnPort } from 'services/ProcessListeningOnPortsService';
import { l4ProtocolLabels } from 'constants/networkFlow';
import { ListDeployment } from 'types/deployment.proto';
import useSet from 'hooks/useSet';
import { GetSortParams } from 'hooks/useURLSort';

function EmbeddedTable({
    deploymentId,
    listeningEndpoints,
}: {
    deploymentId: string;
    listeningEndpoints: ProcessListeningOnPort[];
}) {
    return (
        <TableComposable isNested aria-label="Listening endpoints for deployment">
            <Thead noWrap>
                <Tr>
                    <Th width={20}>Process name</Th>
                    <Th width={10}>PID</Th>
                    <Th width={10}>Port</Th>
                    <Th width={10}>Protocol</Th>
                    <Th width={30}>Pod ID</Th>
                    <Th width={20}>Container name</Th>
                </Tr>
            </Thead>
            <Tbody>
                {listeningEndpoints.map(({ podId, endpoint, signal, containerName }) => (
                    <Tr key={`${deploymentId}/${podId}/${endpoint.port}`}>
                        <Td dataLabel="Process name">{signal.name}</Td>
                        <Td dataLabel="PID">{signal.pid}</Td>
                        <Td dataLabel="Port">{endpoint.port}</Td>
                        <Td dataLabel="Protocol">{l4ProtocolLabels[endpoint.protocol]}</Td>
                        <Td dataLabel="Pod ID">{podId}</Td>
                        <Td dataLabel="Container name">{containerName}</Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export type ListeningEndpointsTableProps = {
    deployments: (ListDeployment & { listeningEndpoints: ProcessListeningOnPort[] })[];
    getSortParams: GetSortParams;
};

function ListeningEndpointsTable({ deployments, getSortParams }: ListeningEndpointsTableProps) {
    const expandedRowSet = useSet<string>();
    return (
        <TableComposable variant="compact" aria-label="Deployment results">
            <Thead noWrap>
                <Tr>
                    <Td width={10}>{/* Header for expanded column */}</Td>
                    <Th width={30} sort={getSortParams('Deployment')}>
                        Deployment
                    </Th>
                    <Th width={30} sort={getSortParams('Namespace')}>
                        Namespace
                    </Th>
                    <Th width={30} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                </Tr>
            </Thead>
            {deployments.map(({ id, name, namespace, cluster, listeningEndpoints }, rowIndex) => {
                const isExpanded = expandedRowSet.has(id);
                return (
                    <Tbody key={id} isExpanded={isExpanded}>
                        <Tr>
                            {listeningEndpoints.length > 0 ? (
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => expandedRowSet.toggle(id),
                                    }}
                                />
                            ) : (
                                <Td />
                            )}
                            <Td dataLabel="Deployment">{name}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td dataLabel="Cluster">{cluster}</Td>
                        </Tr>
                        {listeningEndpoints.length > 0 && (
                            <Tr isExpanded={isExpanded}>
                                <Td colSpan={4}>
                                    <Card className="pf-u-m-md" isFlat>
                                        <EmbeddedTable
                                            deploymentId={id}
                                            listeningEndpoints={listeningEndpoints}
                                        />
                                    </Card>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default ListeningEndpointsTable;
