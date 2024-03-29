import React, { Dispatch, SetStateAction } from 'react';
import { Button, ButtonVariant, Card, Text } from '@patternfly/react-core';
import { Tbody, Tr, Td, Table, Th, Thead } from '@patternfly/react-table';

import { riskBasePath } from 'routePaths';
import LinkShim from 'Components/PatternFly/LinkShim';
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
        <Table isNested aria-label="Listening endpoints for deployment">
            <Thead noWrap>
                <Tr>
                    <Th width={30}>Exec file path</Th>
                    <Th>PID</Th>
                    <Th>Port</Th>
                    <Th>Protocol</Th>
                    <Th width={30}>Pod ID</Th>
                    <Th width={20}>Container name</Th>
                </Tr>
            </Thead>
            <Tbody>
                {listeningEndpoints.map(({ podId, endpoint, signal, containerName }) => (
                    <Tr key={`${deploymentId}/${podId}/${endpoint.port}`}>
                        <Td dataLabel="Exec file path">{signal.execFilePath}</Td>
                        <Td dataLabel="PID">{signal.pid}</Td>
                        <Td dataLabel="Port">{endpoint.port}</Td>
                        <Td dataLabel="Protocol">{l4ProtocolLabels[endpoint.protocol]}</Td>
                        <Td dataLabel="Pod ID">{podId}</Td>
                        <Td dataLabel="Container name">{containerName}</Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

export type ListeningEndpointsTableProps = {
    deployments: (ListDeployment & { listeningEndpoints: ProcessListeningOnPort[] })[];
    getSortParams: GetSortParams;
    areAllRowsExpanded: boolean;
    setAllRowsExpanded: Dispatch<SetStateAction<boolean>>;
};

function ListeningEndpointsTable({
    deployments,
    getSortParams,
    areAllRowsExpanded,
    setAllRowsExpanded,
}: ListeningEndpointsTableProps) {
    // This is used to track which table rows are in the -opposite- state
    // of the passed expanded state for the entire table.
    const invertedExpansionRowSet = useSet<string>();

    return (
        <Table variant="compact" aria-label="Deployment results">
            <Thead noWrap>
                <Tr>
                    <Th
                        expand={{
                            // Possible PF bug? This boolean seems to need to be inverted based on the render output
                            areAllExpanded: !areAllRowsExpanded,
                            // TODO Awkward type assertion here is fixed in PF 5 https://github.com/patternfly/patternfly-react/issues/8330
                            collapseAllAriaLabel: 'Expand or collapse all rows' as '',
                            onToggle: () => {
                                setAllRowsExpanded(!areAllRowsExpanded);
                                invertedExpansionRowSet.clear();
                            },
                        }}
                        width={10}
                    >
                        {/* Header for expanded column */}
                    </Th>
                    <Th width={30} sort={getSortParams('Deployment')}>
                        Deployment
                    </Th>
                    <Th width={20} sort={getSortParams('Cluster')}>
                        Cluster
                    </Th>
                    <Th width={30} sort={getSortParams('Namespace')}>
                        Namespace
                    </Th>
                    <Th>Count</Th>
                </Tr>
            </Thead>
            {deployments.map(({ id, name, namespace, cluster, listeningEndpoints }, rowIndex) => {
                // A row is expanded if
                //   - the "are all rows expanded" toggle is on and the row is not in the toggled set
                //   - the "are all rows expanded" toggle is off and the row is in the toggled set
                const isExpanded = areAllRowsExpanded
                    ? !invertedExpansionRowSet.has(id)
                    : invertedExpansionRowSet.has(id);
                const count = listeningEndpoints.length;
                return (
                    <Tbody key={id} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => invertedExpansionRowSet.toggle(id),
                                }}
                            />
                            <Td dataLabel="Deployment">
                                <Button
                                    variant={ButtonVariant.link}
                                    isInline
                                    component={LinkShim}
                                    href={`${riskBasePath}/${id}`}
                                >
                                    {name}
                                </Button>
                            </Td>
                            <Td dataLabel="Cluster">{cluster}</Td>
                            <Td dataLabel="Namespace">{namespace}</Td>
                            <Td dataLabel="Listening endpoints count">{count}</Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td colSpan={5}>
                                <Card className="pf-v5-u-m-md" isFlat>
                                    {count > 0 ? (
                                        <EmbeddedTable
                                            deploymentId={id}
                                            listeningEndpoints={listeningEndpoints}
                                        />
                                    ) : (
                                        <Text className="pf-v5-u-p-md">
                                            This deployment does not have any reported listening
                                            endpoints
                                        </Text>
                                    )}
                                </Card>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default ListeningEndpointsTable;
