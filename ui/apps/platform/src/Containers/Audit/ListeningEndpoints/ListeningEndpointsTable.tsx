import React from 'react';
import { Bullseye } from '@patternfly/react-core';
import { Tbody, Tr, Td, TableComposable, Th, Thead } from '@patternfly/react-table';

import { ProcessListeningOnPort } from 'services/ProcessListeningOnPortsService';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { l4ProtocolLabels } from 'constants/networkFlow';
import { ListDeployment } from 'types/deployment.proto';

export type ListeningEndpointsTableProps = {
    deployments: (ListDeployment & { listeningEndpoints: ProcessListeningOnPort[] })[];
};

function ListeningEndpointsTable({ deployments }: ListeningEndpointsTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>Program name</Th>
                    <Th>PID</Th>
                    <Th>Port</Th>
                    <Th>Protocol</Th>
                    <Th>Deployment</Th>
                    <Th>Namespace</Th>
                    <Th>Cluster</Th>
                    <Th>Pod ID</Th>
                    <Th>Container name</Th>
                </Tr>
            </Thead>
            <Tbody>
                {deployments.length === 0 && (
                    <Tr>
                        <Td colSpan={8}>
                            <Bullseye>
                                <EmptyStateTemplate title="No results found" headingLevel="h2" />
                            </Bullseye>
                        </Td>
                    </Tr>
                )}
                {deployments.flatMap(({ id, name, namespace, cluster, listeningEndpoints }) =>
                    listeningEndpoints.map(({ podId, endpoint, signal, containerName }) => (
                        <Tr key={`${id}/${podId}/${endpoint.port}`}>
                            <Td dataLabel="Program name">{signal.name}</Td>
                            <Td dataLabel="PID">{signal.pid}</Td>
                            <Td dataLabel="Port">{endpoint.port}</Td>
                            <Td dataLabel="Protocol">{l4ProtocolLabels[endpoint.protocol]}</Td>
                            <Td dataLabel="Deployment">{name}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td dataLabel="Cluster">{cluster}</Td>
                            <Td dataLabel="Pod ID">{podId}</Td>
                            <Td dataLabel="Container name">{containerName}</Td>
                        </Tr>
                    ))
                )}
            </Tbody>
        </TableComposable>
    );
}

export default ListeningEndpointsTable;
